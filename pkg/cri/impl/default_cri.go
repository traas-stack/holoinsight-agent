/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package impl

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/criutils"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8slabels"
	k8smetaextractor "github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta/extractor"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8sutils"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	pb2 "github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"github.com/traas-stack/holoinsight-agent/pkg/util/throttle"
	"github.com/txthinking/socks5"
	"go.uber.org/zap"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"
)

const (
	defaultOpTimeout = 5 * time.Second
	// Copy file maybe slow, so we use a bigger timout
	cpOpTimeout         = 10 * time.Second
	buildTimeout        = 10 * time.Second
	etcLocalTime        = "/etc/localtime"
	zoneinfoDir         = "/usr/share/zoneinfo/"
	unknownIANATimezone = "UNKNOWN"
	maxExecBytes        = 1024 * 1024
)

type (
	// Default cri impl
	defaultCri struct {
		*defaultMetaStore
		syncThrottle          func(func())
		stopCh                chan struct{}
		engine                cri.ContainerEngine
		helperToolLocalMd5sum string
		mutex                 sync.Mutex
		chunkCpCh             chan *cri.Container
		httpProxyServer       *http.Server
		socks5ProxyServer     *socks5.Server
	}
)

var (
	errContainerIsNil = errors.New("container is nil")
	defaultExecUser   = "root"
	// Make sure defaultCri impl cri.Interface
	_ cri.Interface = &defaultCri{}
)

func NewDefaultCri(clientset *kubernetes.Clientset, engine cri.ContainerEngine) cri.Interface {
	return &defaultCri{
		defaultMetaStore: newDefaultMetaStore(clientset),
		syncThrottle:     throttle.ThrottleFirst(time.Second),
		stopCh:           make(chan struct{}),
		engine:           engine,
		chunkCpCh:        make(chan *cri.Container, 128),
	}
}

func (e *defaultCri) Engine() cri.ContainerEngine {
	return e.engine
}

func (e *defaultCri) Start() error {
	if file, err := os.Open(core.HelperToolLocalPath); err == nil {
		defer file.Close()
		md5 := md5.New()
		if _, err := io.Copy(md5, file); err == nil {
			e.helperToolLocalMd5sum = hex.EncodeToString(md5.Sum(nil))
		}
	}

	if err := e.defaultMetaStore.Start(); err != nil {
		return err
	}
	e.syncOnce()

	e.localPodMeta.addEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*v1.Pod)
			logger.Metaz("[local] [k8s] add pod", zap.String("namespace", pod.Namespace), zap.String("name", pod.Name))
			e.maybeSync()
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			pod := newObj.(*v1.Pod)
			logger.Metaz("[local] [k8s] update pod", zap.String("namespace", pod.Namespace), zap.String("name", pod.Name))
			e.maybeSync()
		},
		DeleteFunc: func(obj interface{}) {
			e.maybeSync()
			pod := obj.(*v1.Pod)
			logger.Metaz("[local] [k8s] delete pod", zap.String("namespace", pod.Namespace), zap.String("name", pod.Name))
		},
	})

	go e.syncLoop()
	go e.chunkCpLoop()
	e.registerHttpHandlers()
	e.startHttpProxyServer()
	e.startSocks5ProxyServer()
	e.listenPortForward()
	return nil
}

func (e *defaultCri) chunkCpLoop() {
	for {
		select {
		case c := <-e.chunkCpCh:
			if e.isStopped() {
				return
			}
			func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()

				if e.checkHelperMd5(ctx, c) {
					c.Hacked = cri.HackOk
					return
				}

				begin := time.Now()
				err := e.copyHelper(ctx, c)
				cost := time.Since(begin)

				if err == nil {
					logger.Metaz("[local] retry hack success", zap.String("cid", c.ShortContainerID()), zap.Duration("cost", cost))
					c.Hacked = cri.HackOk
				} else {
					logger.Metaz("[local] retry hack error", zap.String("cid", c.ShortContainerID()), zap.Duration("cost", cost), zap.Error(err))
					c.Hacked = cri.HackRetryError
				}
			}()

		case <-e.stopCh:
			return
		}
	}
}

func (e *defaultCri) Stop() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if e.isStopped() {
		return
	}

	close(e.stopCh)
	e.defaultMetaStore.Stop()

	if e.httpProxyServer != nil {
		logger.Infoz("[netproxy] close http proxy server")
		e.httpProxyServer.Close()
	}

	if e.socks5ProxyServer != nil {
		logger.Infoz("[netproxy] close socks5 proxy server")
		e.socks5ProxyServer.Shutdown()
	}
}

func (e *defaultCri) CopyToContainer(ctx context.Context, c *cri.Container, srcPath, dstPath string) (err error) {
	if c == nil {
		return errContainerIsNil
	}
	begin := time.Now()
	method := "unknown"
	defer func() {
		cost := time.Now().Sub(begin)
		logger.Criz("[digest] copy to container",
			zap.String("engine", e.engine.Type()),
			zap.String("cid", c.ShortContainerID()),
			zap.String("runtime", c.Runtime),
			zap.String("method", method),
			zap.String("src", srcPath),
			zap.String("dst", dstPath),
			zap.Duration("cost", cost),
			zap.Error(err))
	}()
	method, err = criutils.CopyToContainer(ctx, e, c, srcPath, dstPath)
	return
}

func (e *defaultCri) CopyFromContainer(ctx context.Context, c *cri.Container, srcPath, dstPath string) (err error) {
	if c == nil {
		return errContainerIsNil
	}
	begin := time.Now()
	method := "unknown"
	defer func() {
		logger.Criz("[digest] copy from container",
			zap.String("engine", e.engine.Type()),
			zap.String("cid", c.ShortContainerID()),
			zap.String("runtime", c.Runtime),
			zap.String("method", method),
			zap.String("src", srcPath),
			zap.String("dst", dstPath),
			zap.Duration("cost", time.Now().Sub(begin)),
			zap.Error(err))
	}()
	method, err = criutils.CopyFromContainer(ctx, e, c, srcPath, dstPath)
	return
}

func (e *defaultCri) Exec(ctx context.Context, c *cri.Container, req cri.ExecRequest) (r cri.ExecResult, err error) {
	invalidResult := cri.ExecResult{Cmd: strings.Join(req.Cmd, " "), ExitCode: -1}
	if c == nil {
		return invalidResult, errContainerIsNil
	}

	begin := time.Now()
	defer func() {
		cost := time.Now().Sub(begin)

		var stdout, stderr string
		if len(req.Cmd) > 0 && (req.Cmd[0] == "cat" || req.Cmd[0] == "tar") {
			stdout, stderr = r.SampleOutputLength(128)
		} else {
			stdout, stderr = r.SampleOutput()
		}

		logger.Criz("[digest] exec",
			zap.String("engine", e.engine.Type()),
			zap.String("cid", c.ShortContainerID()),
			zap.String("runtime", c.Runtime),
			zap.Strings("cmd", req.Cmd),
			zap.Int("code", r.ExitCode),
			zap.String("stdout", stdout),
			zap.String("stderr", stderr),
			zap.Duration("cost", cost),
			zap.Error(err))
	}()

	execBytes := 0
	for _, s := range req.Cmd {
		execBytes += len(s)
	}
	for _, s := range req.Env {
		execBytes += len(s)
	}
	// Executing large size requests is very dangerous, in my test environment it will cause subsequent exec requests to the same container to hang.
	if execBytes >= maxExecBytes {
		return invalidResult, errors.New("exec req too big")
	}

	if req.User == "" {
		req.User = defaultExecUser
	}

	return e.engine.Exec(ctx, c, req)
}

func (e *defaultCri) listContainers() ([]*cri.EngineSimpleContainer, error) {
	begin := time.Now()
	defer func() {
		logger.Criz("[digest] list all containers", //
			zap.String("engine", e.engine.Type()), //
			zap.Duration("cost", time.Now().Sub(begin)))
	}()

	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()

	return e.engine.ListAllContainers(ctx)
}

// setupTimezone setups timezone info of container
func (e *defaultCri) setupTimezone(ctx context.Context, c *cri.Container) {
	if c.Tz.EnvTz != "" {
		// https://man7.org/linux/man-pages/man3/tzset.3.html

		// :Asia/Shanghai
		if strings.HasPrefix(c.Tz.EnvTz, ":") {
			c.Tz.EnvTz = c.Tz.EnvTz[1:]
		}

		// /usr/share/zoneinfo/Asia/Shanghai
		if strings.HasPrefix(c.Tz.EnvTz, zoneinfoDir) {
			c.Tz.EnvTz = c.Tz.EnvTz[len(zoneinfoDir):]
		}

		if tzObj, err := time.LoadLocation(c.Tz.EnvTz); err == nil {
			c.Tz.TzObj = tzObj
			c.Tz.Name = c.Tz.EnvTz
		}
	}

	tzName, tzObj, err := e.getEtcTimezone0(ctx, c)
	if err != nil {
		logger.Errorz("[local] parse /etc/localtime error", zap.String("cid", c.Id), zap.Error(err))
	}

	c.Tz.EtcLocaltime = tzName
	if c.Tz.TzObj == nil && err == nil {
		c.Tz.TzObj = tzObj
		c.Tz.Name = c.Tz.EtcLocaltime
	}

	if c.Tz.TzObj == nil {
		c.Tz.Name = "UTC"
		c.Tz.TzObj = time.UTC
	}

	c.Tz.Zone, c.Tz.Offset = time.Now().In(c.Tz.TzObj).Zone()
}

func (e *defaultCri) getEtcTimezone0(ctx context.Context, c *cri.Container) (string, *time.Location, error) {
	// ref: https://man7.org/linux/man-pages/man5/localtime.5.html

	// /etc/localtime must be a link to file under /usr/share/zoneinfo/.
	if c.Runtime == cri.Runc {
		hostPath, err := cri.TransferToHostPathForContainer(c, etcLocalTime, false)
		if err != nil {
			return "", nil, err
		}
		st, err := os.Lstat(hostPath)
		if err != nil {
			// If /etc/localtime is missing, the default "UTC" timezone is used.
			if os.IsNotExist(err) {
				return "UTC", time.UTC, nil
			}
			return "", nil, err
		}

		if st.Mode()&os.ModeSymlink != os.ModeSymlink {
			logger.Metaz("[local] /etc/localtime is a regular file", zap.String("cid", c.Id), zap.String("ns", c.Pod.Namespace), zap.String("pod", c.Pod.Name))
			// According to the specification, /etc/localtime should be a symbol link, but it may be a regular file in practice.
			// At this point we have to use its contents to parse the timezone.
			b, err := criutils.ReadContainerFileUsingExecCat(ctx, ioc.Crii, c, etcLocalTime)
			if err != nil {
				return "", nil, err
			}
			return parseTzData(b)
		}

		link, err := os.Readlink(hostPath)
		if err != nil {
			return "", nil, err
		}

		return e.parseTimezoneFromLink(ctx, c, link)
	}

	if r, err := e.Exec(ctx, c, cri.ExecRequest{Cmd: []string{"readlink", etcLocalTime}}); err == nil {
		link := strings.TrimSpace(r.Stdout.String())
		return e.parseTimezoneFromLink(ctx, c, link)
	}

	b, err := criutils.ReadContainerFileUsingExecCat(ctx, ioc.Crii, c, etcLocalTime)
	if err != nil {
		return "", nil, err
	}
	return parseTzData(b)
}

func (e *defaultCri) parseTimezoneFromLink(ctx context.Context, c *cri.Container, link string) (string, *time.Location, error) {
	// The link may be like  "../usr/share/zoneinfo/UTC"
	if strings.HasPrefix(link, "..") {
		link = link[2:]
	}

	if !strings.HasPrefix(link, zoneinfoDir) {
		return "", nil, errors.New("unknown /etc/localtime: " + link)
	}
	name := link[len(zoneinfoDir):]
	if name == "" {
		return "", nil, errors.New("invalid /etc/localtime:" + link)
	}

	_, tzObj, err := loadLocation(name)
	if err != nil {
		return "", nil, err
	}

	if tzObj2, err := e.readTimezoneObjFromLink(ctx, c, link); err != nil {
		logger.Metaz("[local] fail to read and parse container timezone file", zap.String("cid", c.Id), zap.String("link", link), zap.Error(err))
	} else {
		now := time.Now()
		name1, offset1 := now.In(tzObj).Zone()
		name2, offset2 := now.In(tzObj2).Zone()
		if name1 != name2 || offset1 != offset2 {
			logger.Metaz("[local] timezone mismatch",
				zap.String("cid", c.Id),
				zap.String("ns", c.Pod.Namespace),
				zap.String("pod", c.Pod.Name),
				zap.String("/etc/localtime", link),
				zap.String("name1", name1),
				zap.Int("offset1", offset1),
				zap.String("name2", name2),
				zap.Int("offset2", offset2))

			// tzObj2 is more accurate
			name = unknownIANATimezone
			tzObj = tzObj2
		}
	}

	return name, tzObj, nil
}

// readTimezoneObjFromLink read timezone obj from linke
func (e *defaultCri) readTimezoneObjFromLink(ctx context.Context, c *cri.Container, link string) (*time.Location, error) {
	// When the user mounts /usr/share/zoneinfo/Asia/Shanghai of the physical machine to /etc/localtime of the container, the real result may be:
	// 1. /etc/localtime in the container is still a symbol link, pointing to /usr/share/zoneinfo/UTC
	// 2. The content of /usr/share/zoneinfo/UTC in the container becomes the content of Asia/Shanghai.
	// The reason for this phenomenon is that the **k8s mount action will follow symbol link**.

	// In order to get correct results, we must read the timezone file once.
	// In fact, /usr/share/zoneinfo/UTC is covered by mount, but it cannot be seen from the mounts information (because there are some symbol links in the middle).
	// Therefore, the read request must be initiated from inside the container.

	if b, err := criutils.ReadContainerFileUsingExecCat(ctx, ioc.Crii, c, link); err != nil {
		return nil, err
	} else if _, tzObj, err := parseTzData(b); err != nil {
		return nil, err
	} else {
		return tzObj, nil
	}
}

func (e *defaultCri) isSidecar(c *cri.Container) bool {
	return k8smetaextractor.DefaultPodMetaService.IsSidecar(c)
}

func (e *defaultCri) getHostname(container *cri.Container) (string, error) {
	hostname := container.Env["HOSTNAME"]
	if hostname != "" {
		return hostname, nil
	}

	if !container.IsRunning() {
		return "", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()
	result, err := e.Exec(ctx, container, cri.ExecRequest{Cmd: []string{"hostname"}})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout.String()), nil
}

func (e *defaultCri) checkHelperMd5(ctx context.Context, c *cri.Container) bool {
	if e.helperToolLocalMd5sum == "" {
		return false
	}
	if md5, err := criutils.Md5sum(ctx, e, c, core.HelperToolPath); err == nil {
		logger.Metaz("[local] helper exists",
			zap.String("cid", c.ShortContainerID()),
			zap.String("md5", md5),
			zap.String("local-md5", e.helperToolLocalMd5sum),
		)
		if md5 == e.helperToolLocalMd5sum {
			logger.Metaz("[local] already hack",
				zap.String("cid", c.ShortContainerID()),
				zap.String("ns", c.Pod.Namespace),
				zap.String("pod", c.Pod.Name))
			return true
		}
	}
	return false
}

func (e *defaultCri) copyHelper(ctx context.Context, c *cri.Container) error {
	return e.CopyToContainer(ctx, c, core.HelperToolLocalPath, core.HelperToolPath)
}

func (e *defaultCri) buildCriContainer(criPod *cri.Pod, dc *cri.EngineDetailContainer) *cri.Container {
	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

	k8sContainerName := k8slabels.GetContainerName(dc.Labels)
	if k8sContainerName == "" && dc.IsSandbox {
		k8sContainerName = "POD"
	}
	criContainer := &cri.Container{
		Id:               dc.ID,
		State:            dc.State,
		ContainerName:    dc.Name,
		K8sContainerName: k8sContainerName,
		Pod:              criPod,
		Labels:           dc.Labels,
		Env:              util.ParseStringSliceEnvToMap(dc.Env),
		Hostname:         dc.Hostname,
		SandboxID:        dc.SandboxId,
		Runtime:          dc.Runtime,
		NetworkMode:      dc.NetworkMode,
		Hacked:           cri.HackInit,
	}

	if criContainer.Hostname == "" {
		criContainer.Hostname = criPod.Pod.Spec.Hostname
	}

	criContainer.Tz.EnvTz = criContainer.Env["TZ"]

	if dc.IsSandbox {
		criContainer.Sandbox = true
	} else if e.isSidecar(criContainer) {
		criContainer.Sidecar = true
	} else {
		criContainer.MainBiz = true
	}

	if !dc.IsSandbox {

		if criContainer.Runtime == cri.Runc && dc.MergedDir != "" {
			criContainer.MergedDir = filepath.Join(core.GetHostfs(), dc.MergedDir)
		}

		for _, m := range dc.Mounts {
			source := filepath.Join(core.GetHostfs(), m.Source)

			if !m.RW {
				continue
			} else if st, err := os.Stat(source); err != nil {
				continue
			} else if !st.IsDir() {
				continue
			}

			criContainer.Mounts = append(criContainer.Mounts, &cri.MountPoint{
				Source:      source,
				Destination: m.Destination,
				RW:          true,
			})
		}
	}

	criPod.All = append(criPod.All, criContainer)

	if criContainer.IsRunning() && criContainer.Hacked == cri.HackInit && criContainer.MainBiz {
		criContainer.Hacked = cri.HackIng

		var err error

		e.setupTimezone(ctx, criContainer)

		if criContainer.Hostname == "" {
			criContainer.Hostname, err = e.getHostname(criContainer)
			if err != nil {
				logger.Metaz("[local] fail to get hostname",
					zap.String("ns", criPod.Namespace), //
					zap.String("pod", criPod.Name),     //
					zap.String("cid", criContainer.ShortContainerID()),
					zap.Error(err))
			}
		}

		// skip kube-system containers
		if !strings.HasPrefix(criPod.Namespace, "kube-") {
			alreadyExists := false
			if e.checkHelperMd5(ctx, criContainer) {
				alreadyExists = true
				criContainer.Hacked = cri.HackOk
			}

			if !alreadyExists {
				err = e.copyHelper(ctx, criContainer)
				if err == nil {
					criContainer.Hacked = cri.HackOk
					logger.Metaz("[local] hack success",
						zap.String("cid", criContainer.ShortContainerID()),
						zap.String("ns", criPod.Namespace),
						zap.String("pod", criPod.Name),
						zap.Error(err))
				} else {
					logger.Metaz("[local] hack error",
						zap.String("cid", criContainer.ShortContainerID()),
						zap.String("ns", criPod.Namespace),
						zap.String("pod", criPod.Name),
						zap.Error(err))

					ioc.RegistryService.ReportEventAsync(&pb2.ReportEventRequest_Event{
						EventTimestamp: time.Now().UnixMilli(),
						EventType:      "DIGEST",
						PayloadType:    "init_container_error",
						Tags: map[string]string{
							"namespace": criPod.Namespace,
							"pod":       criPod.Name,
							"cid":       criContainer.ShortContainerID(),
							"agent":     e.localAgentMeta.PodName(),
						},
						Numbers: nil,
						Strings: map[string]string{
							"err": err.Error(),
						},
						Logs: nil,
						Json: "",
					})

					// It makes sense to retry with a timeout
					if context.DeadlineExceeded == err {
						time.AfterFunc(3*time.Second, func() {
							select {
							case e.chunkCpCh <- criContainer:
							default:
							}
						})
					}
				}
			}
		} else {
			criContainer.Hacked = cri.HackSkipped
		}
	}

	return criContainer
}

func (e *defaultCri) maybeSync() {
	e.syncThrottle(e.syncOnce)
}

func (e *defaultCri) syncLoop() {
	go func() {
		syncTimer, _ := util.NewAlignedTimer(time.Minute, 40*time.Second, true, false)
		defer syncTimer.Stop()

		for {
			select {
			case <-syncTimer.C:
				e.maybeSync()
				syncTimer.Next()
			case <-e.stopCh:
				return
			}
		}
	}()
}

func (e *defaultCri) isStopped() bool {
	select {
	case <-e.stopCh:
		return true
	default:
		return false
	}
}

func (e *defaultCri) syncOnce() {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.isStopped() {
		return
	}

	begin := time.Now()

	containers, err := e.listContainers()
	if err != nil {
		return
	}

	oldState := e.state
	newState := newInternalState()
	newStateLock := &sync.Mutex{}

	// containers index by labels["io.kubernetes.pod.uid"]
	containersByPod := make(map[string][]*cri.EngineDetailContainer)

	for i := range containers {
		simpleContainer := containers[i]

		// Skip containers which are not controlled by k8s
		uid := k8slabels.GetPodUID(simpleContainer.Labels)
		if uid == "" {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
		detail, err := e.engine.GetContainerDetail(ctx, simpleContainer.ID)
		cancel()
		if err != nil {
			logger.Criz("[digest] inspect error", zap.String("cid", simpleContainer.ID), zap.Error(err))
			continue
		}

		containersByPod[uid] = append(containersByPod[uid], detail)
	}

	localPods := e.localPodMeta.getAllPods()

	podPhaseCount := make(map[v1.PodPhase]int)

	anyChanged := false
	expiredContainers := 0
	semaphore := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for _, pod0 := range localPods {
		pod := pod0
		begin := time.Now()

		podPhaseCount[pod.Status.Phase]++
		if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
			logger.Metaz("[local] skip pod", zap.String("ns", pod.Namespace), zap.String("pod", pod.Name), zap.String("phase", string(pod.Status.Phase)))
			continue
		}

		semaphore <- struct{}{}
		wg.Add(1)
		go func() {
			defer func() {
				<-semaphore
				wg.Done()
			}()

			criPod, podExpiredContainers, podChanged, _ := e.buildPod(pod, oldState, newState, newStateLock, containersByPod)
			if podChanged {
				anyChanged = true
			}

			var sandboxCid string
			if criPod.Sandbox != nil {
				sandboxCid = criPod.Sandbox.Id
			}

			cost := time.Since(begin)
			if podChanged {
				logger.Metaz("[local] build pod",
					zap.String("ns", pod.Namespace),
					zap.String("pod", pod.Name),
					zap.Bool("changed", podChanged), //
					zap.String("sandbox", cri.ShortContainerId(sandboxCid)),
					zap.Int("all", len(criPod.All)),
					zap.Int("biz", len(criPod.Biz)),
					zap.Int("sidecar", len(criPod.Sidecar)),
					zap.Int("expired", podExpiredContainers),
					zap.Duration("cost", cost))
			}

			newStateLock.Lock()
			newState.pods = append(newState.pods, criPod)
			newStateLock.Unlock()
		}()
	}

	wg.Wait()
	newState.build()
	logger.Metaz("[local] sync once done", //
		zap.String("engine", e.engine.Type()), //
		zap.Bool("changed", anyChanged),
		zap.Int("pods", len(newState.pods)), //
		zap.Int("containers", len(containers)),
		zap.Duration("cost", time.Now().Sub(begin)), //
		zap.Int("expired", expiredContainers),       //
		zap.Any("phase", podPhaseCount),             //
	)

	e.state = newState
	if anyChanged {
		e.firePodChange()
	}
}

func (e *defaultCri) firePodChange() {
	e.defaultMetaStore.mutex.Lock()
	defer e.defaultMetaStore.mutex.Unlock()

	for _, listener := range e.listeners {
		listener.OnAnyPodChanged()
	}
}

func (e *defaultCri) buildPod(pod *v1.Pod, oldState *internalState, newState *internalState, newStateLock *sync.Mutex, containersByPod map[string][]*cri.EngineDetailContainer) (*cri.Pod, int, bool, error) {

	criPod := &cri.Pod{
		Pod: pod,
	}

	// Get all containers belonging to this pod, including exited containers
	detailContainers := containersByPod[string(pod.UID)]

	// Find newest sandbox
	var sandboxContainer *cri.EngineDetailContainer
	multiSandbox := false
	podExpiredContainers := 0
	expiredContainers := 0
	changed := false

	for _, container := range detailContainers {
		if container.IsSandbox && container.State.IsRunning() {
			if sandboxContainer == nil {
				sandboxContainer = container
			} else {
				multiSandbox = true
				break
			}
		}
	}

	if multiSandbox {
		logger.Metaz("[local] multi sandbox for pod", zap.String("ns", pod.Namespace), zap.String("pod", pod.Name))
	} else if sandboxContainer == nil {
		logger.Metaz("[local] no sandbox for pod", zap.String("ns", pod.Namespace), zap.String("pod", pod.Name))
	} else {

		for _, container := range detailContainers {
			if !container.State.IsRunning() || container.ID != sandboxContainer.ID && container.SandboxId != sandboxContainer.ID {
				logger.Metaz("[local] ignore expired container",
					zap.String("ns", pod.Namespace),
					zap.String("pod", pod.Name),
					zap.String("sandbox", sandboxContainer.ID),
					zap.String("cid", container.ID))
				podExpiredContainers++
				expiredContainers++
				continue
			}

			// Ignore init containers
			if k8sutils.IsInitContainer(pod, container.Labels) {
				continue
			}

			cached := oldState.containerMap[container.ID]

			if cached != nil && !isContainerChanged(cached.engineContainer, container) {
				cached.criContainer.Pod = criPod

				newStateLock.Lock()
				newState.containerMap[container.ID] = &cachedContainer{
					criContainer:    cached.criContainer,
					engineContainer: container,
				}
				newStateLock.Unlock()

			} else {
				if cached != nil {
					logger.Metaz("container changed", zap.String("cid", cached.criContainer.ShortContainerID()))
				}
				changed = true
				criContainer := e.buildCriContainer(criPod, container)
				cached = &cachedContainer{
					engineContainer: container,
					criContainer:    criContainer,
				}
				newStateLock.Lock()
				newState.containerMap[container.ID] = cached
				newStateLock.Unlock()
			}

			criPod.All = append(criPod.All, cached.criContainer)
			if cached.criContainer.Sandbox {
				criPod.Sandbox = cached.criContainer
			} else if cached.criContainer.Sidecar {
				criPod.Sidecar = append(criPod.Sidecar, cached.criContainer)
			} else {
				criPod.Biz = append(criPod.Biz, cached.criContainer)
			}
		}
	}

	return criPod, expiredContainers, changed, nil
}

func isContainerChanged(oldContainer *cri.EngineDetailContainer, newContainer *cri.EngineDetailContainer) bool {
	return oldContainer.State.Pid != newContainer.State.Pid
}

// parseTzData parse timezone info from []byte read from /usr/share/zoneinfo/Xxx/Xxxx
func parseTzData(b []byte) (string, *time.Location, error) {
	// must import _ "time/tzdata"
	tz, err := time.LoadLocationFromTZData("", b)
	if err != nil {
		return "", nil, err
	}

	// In fact, it doesn't matter whether it can solve an IANA Time Zone, the important thing is to solve *Time.Location,
	// because it is actually involved in time parsing. So here we return UNKNOWN as its name.
	return unknownIANATimezone, tz, nil
}

// loadLocation load *Time.location
func loadLocation(name string) (string, *time.Location, error) {
	tzObj, err := time.LoadLocation(name)
	return name, tzObj, err
}

func (e *defaultCri) ExecAsync(ctx context.Context, c *cri.Container, req cri.ExecRequest) (cri.ExecAsyncResult, error) {
	if req.User == "" {
		req.User = defaultExecUser
	}
	return e.engine.ExecAsync(ctx, c, req)
}
