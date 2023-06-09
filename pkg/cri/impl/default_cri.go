/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package impl

import (
	"context"
	"github.com/pkg/errors"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/cricore"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/criutils"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8slabels"
	k8smetaextractor "github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta/extractor"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8sutils"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"github.com/traas-stack/holoinsight-agent/pkg/util/throttle"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"os"
	"path/filepath"
	"strings"
	"time"
	_ "time/tzdata"
)

const (
	defaultOpTimeout    = 5 * time.Second
	etcLocalTime        = "/etc/localtime"
	unknownIANATimezone = "UNKNOWN"
)

type (
	// Default cri impl
	defaultCri struct {
		*defaultMetaStore
		syncThrottle func(func())
		stopCh       chan struct{}
		engine       cri.ContainerEngine
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
	}
}

func (e *defaultCri) Engine() cri.ContainerEngine {
	return e.engine
}

func (e *defaultCri) Start() error {
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
	e.registerHttpHandlers()
	return nil
}

func (e *defaultCri) Stop() {
	close(e.stopCh)
	e.defaultMetaStore.Stop()
}

func (e *defaultCri) CopyToContainer(ctx context.Context, c *cri.Container, srcPath, dstPath string) (err error) {
	if c == nil {
		return errContainerIsNil
	}
	begin := time.Now()
	defer func() {
		cost := time.Now().Sub(begin)
		logger.Criz("[digest] copy to container",
			zap.String("engine", e.engine.Type()),
			zap.String("cid", c.ShortContainerID()),
			zap.String("runtime", c.Runtime),
			zap.String("src", srcPath),
			zap.String("dst", dstPath),
			zap.Duration("cost", cost),
			zap.Error(err))
	}()

	switch c.Runtime {
	case cri.Runc:
		return cricore.CopyToContainerForRunC(ctx, c, srcPath, dstPath)
	default:
		if e.engine.Supports(cri.ContainerEngineFeatureCopy) {
			return e.engine.CopyToContainer(ctx, c, srcPath, dstPath)
		} else {
			return criutils.CopyToContainerByMountAndExec(ctx, e, c, srcPath, dstPath)
		}
	}
}

func (e *defaultCri) CopyFromContainer(ctx context.Context, c *cri.Container, srcPath, dstPath string) (err error) {
	if c == nil {
		return errContainerIsNil
	}
	begin := time.Now()
	defer func() {
		logger.Criz("[digest] copy from container",
			zap.String("engine", e.engine.Type()),
			zap.String("cid", c.ShortContainerID()),
			zap.String("runtime", c.Runtime),
			zap.String("src", srcPath),
			zap.String("dst", dstPath),
			zap.Duration("cost", time.Now().Sub(begin)),
			zap.Error(err))
	}()

	switch c.Runtime {
	case cri.Runc:
		return cricore.CopyFromContainerForRunC(ctx, c, srcPath, dstPath)
	default:
		if e.engine.Supports(cri.ContainerEngineFeatureCopy) {
			return e.engine.CopyFromContainer(ctx, c, srcPath, dstPath)
		} else {
			return criutils.CopyFromContainerByMountAndExec(ctx, e, c, srcPath, dstPath)
		}
	}
}

func (e *defaultCri) Exec(ctx context.Context, c *cri.Container, req cri.ExecRequest) (r cri.ExecResult, err error) {
	if c == nil {
		return cri.ExecResult{ExitCode: -1}, errContainerIsNil
	}
	begin := time.Now()
	defer func() {
		cost := time.Now().Sub(begin)
		stdout, stderr := r.SampleOutput()

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
func (e *defaultCri) setupTimezone(c *cri.Container) {
	if c.Tz.EnvTz != "" {
		if tzObj, err := time.LoadLocation(c.Tz.EnvTz); err == nil {
			c.Tz.TzObj = tzObj
			c.Tz.Name = c.Tz.EnvTz
		}
	}

	tzName, tzObj, err := e.getEtcTimezone0(c)
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

func (e *defaultCri) getEtcTimezone0(c *cri.Container) (string, *time.Location, error) {
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
			ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
			defer cancel()
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

		return e.parseTimezoneFromLink(c, link)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()
	if r, err := e.Exec(ctx, c, cri.ExecRequest{Cmd: []string{"readlink", etcLocalTime}}); err == nil {
		link := strings.TrimSpace(r.Stdout.String())
		return e.parseTimezoneFromLink(c, link)
	}

	b, err := criutils.ReadContainerFileUsingExecCat(ctx, ioc.Crii, c, etcLocalTime)
	if err != nil {
		return "", nil, err
	}
	return parseTzData(b)
}

func (e *defaultCri) parseTimezoneFromLink(c *cri.Container, link string) (string, *time.Location, error) {
	// The link may be like  "../usr/share/zoneinfo/UTC"
	if strings.HasPrefix(link, "..") {
		link = link[2:]
	}

	if !strings.HasPrefix(link, "/usr/share/zoneinfo/") {
		return "", nil, errors.New("unknown /etc/localtime: " + link)
	}
	name := link[len("/usr/share/zoneinfo/"):]

	_, tzObj, err := loadLocation(name)
	if err != nil {
		return "", nil, err
	}

	if tzObj2, err := e.readTimezoneObjFromLink(c, link); err != nil {
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
func (e *defaultCri) readTimezoneObjFromLink(c *cri.Container, link string) (*time.Location, error) {
	// When the user mounts /usr/share/zoneinfo/Asia/Shanghai of the physical machine to /etc/localtime of the container, the real result may be:
	// 1. /etc/localtime in the container is still a symbol link, pointing to /usr/share/zoneinfo/UTC
	// 2. The content of /usr/share/zoneinfo/UTC in the container becomes the content of Asia/Shanghai.
	// The reason for this phenomenon is that the **k8s mount action will follow symbol link**.

	// In order to get correct results, we must read the timezone file once.
	// In fact, /usr/share/zoneinfo/UTC is covered by mount, but it cannot be seen from the mounts information (because there are some symbol links in the middle).
	// Therefore, the read request must be initiated from inside the container.

	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()

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

func (e *defaultCri) buildCriContainer(criPod *cri.Pod, dc *cri.EngineDetailContainer) *cri.Container {
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

	if criContainer.IsRunning() && !criContainer.Hacked && criContainer.MainBiz {
		criContainer.Hacked = true

		var err error

		e.setupTimezone(criContainer)

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
			ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
			defer cancel()
			err := e.CopyToContainer(ctx, criContainer, core.HelperToolLocalPath, core.HelperToolPath)

			//if err == nil {
			//	ctx2, cancel2 := context.WithTimeout(context.Background(), defaultOpTimeout)
			//	defer cancel2()
			//	_, err = e.Exec(ctx2, criContainer, cri.ExecRequest{Cmd: []string{core.HelperToolPath, "hello"}})
			//}

			if err == nil {
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
			}
		}
	}

	return criContainer
}

func (e *defaultCri) maybeSync() {
	e.syncThrottle(e.syncOnce)
}

func (e *defaultCri) syncLoop() {
	go func() {
		timer, _ := util.NewAlignedTimer(time.Minute, 40*time.Second, true, false)
		defer timer.Stop()

		for {
			select {
			case <-timer.C:
				e.maybeSync()
				timer.Next()
			case <-e.stopCh:
				return
			}
		}
	}()
}

func (e *defaultCri) syncOnce() {

	begin := time.Now()

	containers, err := e.listContainers()
	if err != nil {
		return
	}

	oldState := e.state
	newState := newInternalState()

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

	changed := false
	expiredContainers := 0
	for _, pod := range localPods {
		podPhaseCount[pod.Status.Phase]++
		if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
			logger.Metaz("[local] skip pod", zap.String("ns", pod.Namespace), zap.String("pod", pod.Name), zap.String("phase", string(pod.Status.Phase)))
			continue
		}

		criPod := &cri.Pod{
			Pod: pod,
		}

		// Get all containers belonging to this pod, including exited containers
		detailContainers := containersByPod[string(pod.UID)]

		// Find newest sandbox
		var sandboxContainer *cri.EngineDetailContainer
		multiSandbox := false
		podExpiredContainers := 0
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

					newState.containerMap[container.ID] = &cachedContainer{
						criContainer:    cached.criContainer,
						engineContainer: container,
					}
				} else {
					changed = true
					criContainer := e.buildCriContainer(criPod, container)
					cached = &cachedContainer{
						engineContainer: container,
						criContainer:    criContainer,
					}
					newState.containerMap[container.ID] = cached
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

		var sandboxCid string
		if sandboxContainer != nil {
			sandboxCid = sandboxContainer.ID
		}
		logger.Metaz("[local] build pod",
			zap.String("ns", pod.Namespace),
			zap.String("pod", pod.Name),
			zap.String("sandbox", cri.ShortContainerId(sandboxCid)),
			zap.Int("all", len(criPod.All)),
			zap.Int("biz", len(criPod.Biz)),
			zap.Int("sidecar", len(criPod.Sidecar)),
			zap.Int("expired", podExpiredContainers))

		newState.pods = append(newState.pods, criPod)
	}
	newState.build()

	logger.Metaz("[local] sync once done", //
		zap.String("engine", e.engine.Type()), //
		zap.Bool("changed", changed),
		zap.Int("pods", len(newState.pods)), //
		zap.Int("containers", len(containers)),
		zap.Duration("cost", time.Now().Sub(begin)), //
		zap.Int("expired", expiredContainers),       //
		zap.Any("phase", podPhaseCount),             //
	)

	e.state = newState
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
