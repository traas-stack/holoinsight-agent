/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package impl

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/cricore"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/criutils"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8slabels"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta"
	k8smetaextractor "github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta/extractor"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8sutils"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"github.com/traas-stack/holoinsight-agent/pkg/util/throttle"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultOpTimeout = 5 * time.Second
	execSampleLength = 1024
)

type (
	defaultCri struct {
		*defaultMetaStore
		k8smm        *k8smeta.Manager
		syncThrottle func(func())
		stopCh       chan struct{}
		engine       cri.ContainerEngine
	}
)

var (
	errContainerIsNil = errors.New("container is nil")
	defaultExecUser   = "root"
)

func New(k8smm *k8smeta.Manager, engine cri.ContainerEngine) cri.Interface {
	return &defaultCri{
		defaultMetaStore: &defaultMetaStore{
			state: newInternalState(),
		},
		k8smm:        k8smm,
		syncThrottle: throttle.ThrottleFirst(time.Second),
		stopCh:       make(chan struct{}),
		engine:       engine,
	}
}

func (e *defaultCri) Stop() {
	e.defaultMetaStore.Stop()
	close(e.stopCh)
}

func (e *defaultCri) Engine() cri.ContainerEngine {
	return e.engine
}

func (e *defaultCri) Start() error {
	if err := e.defaultMetaStore.Start(); err != nil {
		return err
	}
	e.syncOnce()

	e.k8smm.PodMeta.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if e.isLocalPod(obj) {
				pod := obj.(*v1.Pod)
				logger.Metaz("[local] [k8s] add pod", zap.String("namespace", pod.Namespace), zap.String("name", pod.Name))
				e.maybeSync()
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			trigger := false
			if e.isLocalPod(oldObj) {
				trigger = true
			}
			if e.isLocalPod(newObj) {
				trigger = true
			}
			if trigger {
				// oldPod := newObj.(*v1.Pod)
				pod := newObj.(*v1.Pod)
				logger.Metaz("[local] [k8s] update pod", zap.String("namespace", pod.Namespace), zap.String("name", pod.Name))
				e.maybeSync()
			}
		},
		DeleteFunc: func(obj interface{}) {
			if e.isLocalPod(obj) {
				e.maybeSync()
				pod := obj.(*v1.Pod)
				logger.Metaz("[local] [k8s] delete pod", zap.String("namespace", pod.Namespace), zap.String("name", pod.Name))
			}
		},
	})

	go e.syncLoop()
	e.registerHttpHandlers()
	return nil
}

func (e *defaultCri) listDockerContainers() ([]*cri.EngineSimpleContainer, error) {
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
		stdout := ""
		stderr := ""

		if r.Stdout != nil {
			stdout = string(util.SubBytesMax(r.Stdout.Bytes(), execSampleLength))
		}
		if r.Stderr != nil {
			stderr = string(util.SubBytesMax(r.Stderr.Bytes(), execSampleLength))
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

	if req.User == "" {
		req.User = defaultExecUser
	}

	return e.engine.Exec(ctx, c, req)
}

func (e *defaultCri) getEtcTimezone(c *cri.Container) (string, error) {
	tz, err := e.getEtcTimezone0(c)
	if tz == "" {
		// If /etc/localtime is missing, the default "UTC" timezone is used.
		tz = "UTC"
	}
	return tz, err
}

func (e *defaultCri) getEtcTimezone0(c *cri.Container) (string, error) {
	// ref: https://man7.org/linux/man-pages/man5/localtime.5.html

	// /etc/localtime 控制着系统级别的时区, 如果不存在则默认为UTC, 如果存在则必须是 /usr/share/zoneinfo/ 下的一个符号链接!
	// 每个进程的TZ环境变量则可以强制覆盖本进程的时区

	if c.Runtime == cri.Runc {
		hostPath, err := cri.TransferToHostPathForContainer(c, "/etc/localtime", false)
		if err != nil {
			return "", err
		}
		st, err := os.Lstat(hostPath)

		if err != nil {
			// If /etc/localtime is missing, the default "UTC" timezone is used.
			if os.IsNotExist(err) {
				return "UTC", nil
			}

			// 按照规范 应该是一个 link 但实践下来发现有一些 regular file
			return "", err
		}
		if st.Mode()&os.ModeSymlink != os.ModeSymlink {
			return "", fmt.Errorf("/etc/localtime must be a symbol link, hostPath=%s", hostPath)
		}
		link, err := os.Readlink(hostPath)
		if err != nil {
			return "", err
		}

		// 实测如到的结果可能是 "../usr/share/zoneinfo/UTC" 于是这里做特殊处理
		if strings.HasPrefix(link, "..") {
			link = link[2:]
		}

		// /usr/share/zoneinfo/Asia/Shanghai
		if s := parseTimezoneNameFromLink(link); s != "" {
			return s, nil
		}
		// 这里只能读出内容 然后
		// time.LoadLocationFromTZData()
		return "", errors.New("unknown link: " + link)
	}

	// TODO add a helper method to parse timezone in container ?
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()
	r, err := e.Exec(ctx, c, cri.ExecRequest{Cmd: []string{"readlink", "/etc/localtime"}})
	if err != nil {
		return "", err
	}
	// if /etc/localtime is a regular file or not exist, exitcode == 1
	// ends with \n
	link := strings.TrimSpace(r.Stdout.String())
	if s := parseTimezoneNameFromLink(link); s != "" {
		return s, nil
	}
	return "", errors.New("unknown link: " + link)
}

func parseTimezoneNameFromLink(link string) string {
	if strings.HasPrefix(link, "/usr/share/zoneinfo/") {
		return link[len("/usr/share/zoneinfo/"):]
	}
	return ""
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

	criContainer.EnvTz = criContainer.Env["TZ"]

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

		criContainer.EtcLocaltime, err = e.getEtcTimezone(criContainer)
		if err != nil {
			logger.Metaz("[local] fail to parse /etc/localtime",
				zap.String("ns", criPod.Namespace), //
				zap.String("pod", criPod.Name),     //
				zap.String("cid", criContainer.ShortContainerID()),
				zap.Error(err))
		}

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

func (e *defaultCri) isLocalPod(obj interface{}) bool {
	if pod, ok := obj.(*v1.Pod); ok {
		return e.k8smm.LocalMeta.IsLocalPod(pod)
	}
	return false
}

func (e *defaultCri) syncOnce() {
	begin := time.Now()

	containers, err := e.listDockerContainers()
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

	localPods := e.k8smm.GetLocalHostPods()

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

				cached := oldState.ContainerMap[container.ID]

				if cached != nil && !isContainerChanged(cached.engineContainer, container) {
					cached.criContainer.Pod = criPod

					newState.ContainerMap[container.ID] = &cachedContainer{
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
					newState.ContainerMap[container.ID] = cached
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

		newState.Pods = append(newState.Pods, criPod)
	}
	newState.build()

	logger.Metaz("[local] sync once done", //
		zap.String("engine", e.engine.Type()), //
		zap.Bool("changed", changed),
		zap.Int("pods", len(newState.Pods)), //
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
