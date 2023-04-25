/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package meta

import (
	"bytes"
	"context"
	"fmt"
	"github.com/bep/debounce"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	dockersdk "github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/dockerutils"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/pouch"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8slabels"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta"
	k8smetaextractor "github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta/extractor"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8ssysmetrics/common"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output/gateway"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry"
	"github.com/traas-stack/holoinsight-agent/pkg/util/trigger"
	"go.uber.org/zap"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultSyncInterval      = time.Minute
	listContainersTimeout    = 10 * time.Second
	inspectContainersTimeout = 3 * time.Second
)

type (
	// TODO 别暴露
	DockerLocalMetaImpl struct {
		docker       *dockersdk.Client
		state        *internalState
		syncDebounce func(func())
		rs           *registry.Service
		k8smm        *k8smeta.Manager
		oomRecoder   *oomRecoder
	}
	internalState struct {
		Pods                 []*cri.Pod
		RunningPodMap        map[string]*cri.Pod         `json:"-"`
		ContainerMap         map[string]*CachedContainer `json:"-"`
		shortCidContainerMap map[string]*CachedContainer `json:"-"`
	}
	CachedContainer struct {
		DockerContainer *types.ContainerJSON
		CriContainer    *cri.Container
	}
)

func (s *internalState) build() {
	for id, c := range s.ContainerMap {
		s.shortCidContainerMap[id[:12]] = c
	}
	s.RunningPodMap = make(map[string]*cri.Pod)
	for _, pod := range s.Pods {
		if pod.IsRunning() {
			s.RunningPodMap[pod.Namespace+"/"+pod.Name] = pod
		}

		for _, container := range pod.All {
			// source 长的优先
			cri.SortMountPointsByLongSourceFirst(container.Mounts)
		}
	}
}

func New(rs *registry.Service, k8smm *k8smeta.Manager, docker *dockersdk.Client) *DockerLocalMetaImpl {
	return &DockerLocalMetaImpl{
		rs:     rs,
		docker: docker,
		k8smm:  k8smm,
		state: &internalState{
			RunningPodMap:        make(map[string]*cri.Pod),
			ContainerMap:         make(map[string]*CachedContainer),
			shortCidContainerMap: make(map[string]*CachedContainer),
		},
		// 函数去抖:
		// 每次k8s元数据变化后,
		syncDebounce: debounce.New(time.Second),
		oomRecoder:   newOOMRecorder(),
	}
}

func (l *DockerLocalMetaImpl) GetAllPods() []*cri.Pod {
	return l.state.Pods
}

func (l *DockerLocalMetaImpl) isLocalPod(obj interface{}) bool {
	if pod, ok := obj.(*v1.Pod); ok {
		return l.k8smm.LocalMeta.IsLocalPod(pod)
	}
	return false
}

func (l *DockerLocalMetaImpl) Start() {
	// add 后立即触发, 结束之后如果还有则立即再触发

	l.syncOnce()

	go l.listenDockerLoop()

	l.k8smm.PodMeta.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if l.isLocalPod(obj) {
				pod := obj.(*v1.Pod)
				logger.Metaz("[local] [k8s] add pod", zap.String("namespace", pod.Namespace), zap.String("name", pod.Name))
				l.maybeSync()
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			trigger := false
			if l.isLocalPod(oldObj) {
				trigger = true
			}
			if l.isLocalPod(newObj) {
				trigger = true
			}
			if trigger {
				// oldPod := newObj.(*v1.Pod)
				pod := newObj.(*v1.Pod)
				logger.Metaz("[local] [k8s] update pod", zap.String("namespace", pod.Namespace), zap.String("name", pod.Name))
				l.maybeSync()
			}
		},
		DeleteFunc: func(obj interface{}) {
			if l.isLocalPod(obj) {
				l.maybeSync()
				pod := obj.(*v1.Pod)
				logger.Metaz("[local] [k8s] delete pod", zap.String("namespace", pod.Namespace), zap.String("name", pod.Name))
			}
		},
	})
	go l.syncLoop()
	go l.emitOOMMetrics()
	l.registerHttpHandlers()
}

// 这个似乎没什么用? 因为我们的更新主要是靠pods来驱动的
func (l *DockerLocalMetaImpl) listenDockerLoop() {
	filter := filters.NewArgs()

	filter.Add("type", "container")

	// 创建容器
	filter.Add("event", "create")
	// 启动
	filter.Add("event", "start")
	// 容器退出
	filter.Add("event", "die")
	// 销毁容器
	filter.Add("event", "destroy")
	// 容器尝试突破内存限制
	filter.Add("event", "oom")

	for {
		func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			msgCh, errCh := l.docker.Events(ctx, types.EventsOptions{
				Filters: filter,
			})
			for {
				select {
				case msg := <-msgCh:
					action := dockerutils.ExtractEventAction(msg.Action)
					if action == "oom" {
						l.handleOOM(msg)
					} else {
						logger.Metaz("[docker] [event]", zap.String("cid", msg.ID), zap.String("action", action), zap.Any("msg", msg))
					}
				case err := <-errCh:
					logger.Metaz("[docker] [event] error", zap.Error(err))
					// 低频case 稍微等一下 避免消耗太多CPU
					time.Sleep(time.Second)
					return
				}
			}
		}()
	}
}

func (l *DockerLocalMetaImpl) syncLoop() {
	go func() {
		t := time.NewTicker(defaultSyncInterval)
		for range t.C {
			l.maybeSync()
		}
	}()
}

func (l *DockerLocalMetaImpl) listDockerContainers() ([]types.Container, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listContainersTimeout)
	defer cancel()
	return l.docker.ContainerList(ctx, types.ContainerListOptions{All: true})
}

func (l *DockerLocalMetaImpl) inspectDockerContainers(cid string) (types.ContainerJSON, error) {
	ctx, cancel := context.WithTimeout(context.Background(), inspectContainersTimeout)
	defer cancel()
	return l.docker.ContainerInspect(ctx, cid)
}

// useCache 是否能使用缓存
func (l *DockerLocalMetaImpl) syncOnce() {
	begin := time.Now()

	dockerContainers, err := l.listDockerContainers()
	logger.Metaz("[local] [docker] list containers", zap.Int("count", len(dockerContainers)), zap.Duration("cost", time.Now().Sub(begin)))
	if err != nil {
		return
	}

	oldState := l.state
	newState := &internalState{
		ContainerMap:         make(map[string]*CachedContainer, len(oldState.ContainerMap)),
		shortCidContainerMap: make(map[string]*CachedContainer, len(oldState.ContainerMap)),
	}

	// 建立一个索引, key是 namespace+ pod value 是 所有的containers
	dockerContainersByPod := make(map[string][]*types.ContainerJSON)

	for i := range dockerContainers {
		simpleDc := &dockerContainers[i]

		namespace := k8slabels.GetNamespace(simpleDc.Labels)
		podName := k8slabels.GetPodName(simpleDc.Labels)
		// 跳过非k8s管控的容器
		if namespace == "" || podName == "" {
			continue
		}

		// 这个 inspect 是必要的, 这样才能拿到容器 start 时间戳
		dc, err := l.inspectDockerContainers(simpleDc.ID)
		if err != nil {
			logger.Metaz("[local] [docker] inspect error", zap.String("ns", namespace), zap.String("pod", podName), zap.String("cid", simpleDc.ID), zap.Error(err))
			continue
		}

		key := namespace + "/" + podName
		dockerContainersByPod[key] = append(dockerContainersByPod[key], &dc)
	}

	// 本机负责的pods
	localPods := l.k8smm.GetLocalHostPods()

	// 每个阶段的pod的数量统计
	podPhaseCount := make(map[v1.PodPhase]int)

	for _, pod := range localPods {
		podPhaseCount[pod.Status.Phase]++
		criPod := &cri.Pod{
			Pod: pod,
			App: k8smetaextractor.DefaultPodMetaService.ExtractApp(pod),
		}

		// 对应的docker容器
		dcs := dockerContainersByPod[pod.Namespace+"/"+pod.Name]

		for _, dc := range dcs {

			// 忽略初始化容器
			if isInitContainer(pod, dc) {
				continue
			}

			cached := oldState.ContainerMap[dc.ID]

			if cached != nil && !isContainerChanged(cached.DockerContainer, dc) {
				// 认为容器没有任何变化
				cached.CriContainer.Pod = criPod

				newState.ContainerMap[dc.ID] = &CachedContainer{
					CriContainer:    cached.CriContainer,
					DockerContainer: dc,
				}

				if logger.DebugEnabled {
					logger.Metaz("[local] [docker] use old container meta",
						zap.String("ns", cached.CriContainer.Pod.Namespace),
						zap.String("pod", cached.CriContainer.Pod.Name),
						zap.String("container", cached.CriContainer.Name),
						zap.String("cid", dc.ID))
				}
			} else {
				criContainer := l.buildCriContainer(criPod, dc)
				cached = &CachedContainer{
					CriContainer:    criContainer,
					DockerContainer: dc,
				}
				newState.ContainerMap[dc.ID] = cached
			}

			criPod.All = append(criPod.All, cached.CriContainer)
			if cached.CriContainer.Sandbox {
				criPod.Sandbox = cached.CriContainer
			} else if cached.CriContainer.Sidecar {
				criPod.Sidecar = append(criPod.Sidecar, cached.CriContainer)
			} else {
				criPod.Biz = append(criPod.Biz, cached.CriContainer)
			}

			if logger.DebugEnabled {
				logger.Metaz("[local] [docker] container info", zap.Any("container", cached.CriContainer))
			}
		}

		logger.Metaz("[local] [docker] pod",
			zap.String("ns", pod.Namespace),
			zap.String("pod", pod.Name),
			zap.String("app", criPod.App),
			zap.Int("all", len(criPod.All)),
			zap.Int("biz", len(criPod.Biz)),
			zap.Int("sidecar", len(criPod.Sidecar)))
		newState.Pods = append(newState.Pods, criPod)
	}
	newState.build()

	logger.Metaz("[local] build", //
		zap.String("cri", "docker"),         //
		zap.Int("pods", len(newState.Pods)), //
		zap.Int("containers", len(dockerContainers)),
		zap.Duration("cost", time.Now().Sub(begin)), //
		zap.Any("phase", podPhaseCount),             //
	)

	l.state = newState
}

func isInitContainer(pod *v1.Pod, container *types.ContainerJSON) bool {
	containerName := k8slabels.GetContainerName(container.Config.Labels)
	for i := range pod.Spec.InitContainers {
		ic := &pod.Spec.InitContainers[i]
		if ic.Name == containerName {
			return true
		}
	}
	return false
}

func isContainerChanged(oldc *types.ContainerJSON, newc *types.ContainerJSON) bool {
	return oldc.State.StartedAt != newc.State.StartedAt
}

func parseEnv(envs []string) map[string]string {
	ret := make(map[string]string, len(envs))
	for _, pair := range envs {
		ss := strings.SplitN(pair, "=", 2)
		if len(ss) == 2 {
			ret[ss[0]] = ss[1]
		}
	}
	return ret
}

func (l *DockerLocalMetaImpl) maybeSync() {
	l.syncDebounce(l.syncOnce)
}

func (l *DockerLocalMetaImpl) GetPod(ns, pod string) (*cri.Pod, bool) {
	state := l.state
	// TODO 低效率
	for _, p := range state.Pods {
		if p.Namespace == ns && p.Name == pod {
			return p, true
		}
	}
	return nil, false
}

func (l *DockerLocalMetaImpl) GetPodByHostname(hostname string) (*cri.Pod, bool) {
	state := l.state
	// TODO 低效率
	for _, p := range state.Pods {
		if k8smetaextractor.DefaultPodMetaService.ExtractHostname(p.Pod) == hostname {
			return p, true
		}
	}
	return nil, false
}

func ValidateOutputPathFileMode(fileMode os.FileMode) error {
	switch {
	case fileMode&os.ModeDevice != 0:
		return errors.New("got a device")
	case fileMode&os.ModeIrregular != 0:
		return errors.New("got an irregular file")
	}
	return nil
}

func (l *DockerLocalMetaImpl) CopyToContainer(ctx context.Context, c *cri.Container, srcPath, dstPath string) (err error) {
	if c == nil {
		return errors.New("main container is nil")
	}
	begin := time.Now()
	defer func() {
		cost := time.Now().Sub(begin)
		logger.Infoz("[docker] [cp] digest", zap.String("cid", c.Id),
			zap.String("src", srcPath),
			zap.String("dst", dstPath),
			zap.Duration("cost", cost),
			zap.Error(err))
	}()
	return l.copyToContainerByMount(ctx, c, srcPath, dstPath)
}

func (l *DockerLocalMetaImpl) copyToContainerByMount(ctx context.Context, c *cri.Container, srcPath, dstPath string) error {
	// 转成宿主机地址
	hostPath := filepath.Join(core.GetHostfs(), "proc", cast.ToString(c.State.Pid), "root", dstPath)
	//hostPath, err := cri.TransferToHostPath0(c, dstPath, true)
	//if err != nil {
	//	return err
	//}

	if true {
		dir := filepath.Dir(hostPath)
		if _, err := os.Stat(dir); err != nil && os.IsNotExist(err) {
			err := os.MkdirAll(dir, 0777)
			if err != nil {
				return err
			}
		}
		err := exec.CommandContext(ctx, "/usr/bin/cp", srcPath, hostPath).Run()
		if err != nil {
			err = errors.Wrapf(err, " src=[%s] dst=[%s]", srcPath, hostPath)
		}
		return err
	} else {
		in, err := os.Open(srcPath)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.Create(hostPath)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, in)
		return err
	}
}

// 通过标准api复制进去
func (l *DockerLocalMetaImpl) copyToContainerByDockerAPI(ctx context.Context, c *cri.Container, srcPath, dstPath string) error {
	return copyToContainerByDockerAPI(l.docker, ctx, c, srcPath, dstPath)
}

func (l *DockerLocalMetaImpl) NsEnterExec(ctx context.Context, nsEnterTypes []cri.NsEnterType, c *cri.Container, cmd []string, env []string, workingDir string, input io.Reader) (cri.ExecResult, error) {
	begin := time.Now()
	defer func() {
		cost := time.Now().Sub(begin)
		logger.Infoz("[docker] [exec] [nsenter] digest", zap.String("cid", c.Id), zap.Strings("cmd", cmd), zap.Duration("cost", cost))
	}()

	return execNsEnter(ctx, core.GetHostfs(), nsEnterTypes, c.State.Pid, cmd, env, workingDir, input)
}

func (l *DockerLocalMetaImpl) ExecSync(ctx context.Context, c *cri.Container, cmd []string, env []string, workingDir string, input io.Reader) (cri.ExecResult, error) {
	begin := time.Now()
	defer func() {
		cost := time.Now().Sub(begin)
		logger.Infoz("[docker] [exec] digest", zap.String("cid", c.Id), zap.Strings("cmd", cmd), zap.Duration("cost", cost))
	}()
	return execSync(l.docker, ctx, c, cmd, env, workingDir, input)
}

func (l *DockerLocalMetaImpl) getEtcTimezone(ctx context.Context, c *cri.Container) (string, error) {
	tz, err := l.getEtcTimezone0(ctx, c)
	if tz == "" {
		// If /etc/localtime is missing, the default "UTC" timezone is used.
		tz = "UTC"
	}
	return tz, err
}
func (l *DockerLocalMetaImpl) getEtcTimezone0(ctx context.Context, c *cri.Container) (string, error) {
	// 参考文档
	// https://man7.org/linux/man-pages/man5/localtime.5.html

	// /etc/localtime 控制着系统级别的时区, 如果不存在则默认为UTC, 如果存在则必须是 /usr/share/zoneinfo/ 下的一个符号链接!
	// 每个进程的TZ环境变量则可以强制覆盖本进程的时区

	// 但我们无法感知日志内容是哪个进程所写(非要用lsof也行...), 所以我们只能假定写日志的进程使用的是系统时区

	hostPath, err := cri.TransferToHostPath0(c, "/etc/localtime", false)
	if err == nil {
		st, err := os.Lstat(hostPath)

		if err != nil {
			// If /etc/localtime is missing, the default "UTC" timezone is used.
			if os.IsNotExist(err) {
				return "UTC", nil
			}

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
	} else {
		er, err := l.ExecSync(ctx, c, []string{"readlink", "/etc/localtime"}, nil, "", nil)
		if err != nil {
			return "", err
		}
		if er.ExitCode != 0 {
			return "", errors.New("bad exitcode")
		}
		link := er.Stdout.String()
		if s := parseTimezoneNameFromLink(link); s != "" {
			return s, nil
		}
		return "", errors.New("unknown link: " + link)
	}
}

func parseTimezoneNameFromLink(link string) string {
	if strings.HasPrefix(link, "/usr/share/zoneinfo/") {
		return link[len("/usr/share/zoneinfo/"):]
	}
	return ""
}

// input must be io.reader / []byte / string
func (l *DockerLocalMetaImpl) NsEnterHelperExec(ctx context.Context, c *cri.Container, args []string, env []string, workingDir string, input interface{}) (cri.ExecResult, error) {
	ioReader, err := convertToIOReader(input)
	if err != nil {
		return cri.ExecResult{}, err
	}
	cmd := make([]string, 0, 1+len(args))
	cmd = append(cmd, core.HelperToolPath)
	cmd = append(cmd, args...)
	return l.NsEnterExec(ctx, nil, c, cmd, env, workingDir, ioReader)
}

// 判断容器是否是一个k8s管控的容器
func (l *DockerLocalMetaImpl) isK8sContainer(labels map[string]string) bool {
	return k8slabels.GetNamespace(labels) != "" && k8slabels.GetPodName(labels) != ""
}

// 判断目标容器是否是一个 sandbox
func (l *DockerLocalMetaImpl) isSandbox(c *cri.Container) bool {
	return k8smetaextractor.DefaultPodMetaService.IsSandbox(c)
}

func (l *DockerLocalMetaImpl) isSidecar(c *cri.Container) bool {
	return k8smetaextractor.DefaultPodMetaService.IsSidecar(c)
}

func convertToIOReader(input interface{}) (io.Reader, error) {
	if input == nil {
		return nil, nil
	}
	switch x := input.(type) {
	case string:
		return strings.NewReader(x), nil
	case []byte:
		return bytes.NewBuffer(x), nil
	case io.Reader:
		return x, nil
	default:
		return nil, errors.New("invalid input")
	}
}

func (l *DockerLocalMetaImpl) GetContainerByCid(cid string) (*cri.Container, bool) {
	// docker 12位 cid
	// fa5799111150
	if c, ok := l.state.shortCidContainerMap[cid]; ok {
		return c.CriContainer, true
	}
	// docker 完整长度 cid
	if c, ok := l.state.ContainerMap[cid]; ok {
		return c.CriContainer, true
	}
	return nil, false
}

func (l *DockerLocalMetaImpl) CheckSandboxByLabels(labels map[string]string) bool {
	return dockerutils.IsSandbox(labels) || pouch.IsSandbox(labels)
}

func (l *DockerLocalMetaImpl) getHostname(ctx context.Context, container *cri.Container) (string, error) {

	hostname := container.Env["HOSTNAME"]
	if hostname != "" {
		return hostname, nil
	}

	if !container.IsRunning() {
		return "", nil
	}

	ctx2, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	result, err := l.ExecSync(ctx2, container, []string{"hostname"}, nil, "", nil)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout.String()), nil
}

func (l *DockerLocalMetaImpl) buildCriContainer(criPod *cri.Pod, dc *types.ContainerJSON) *cri.Container {
	// 容器的大部分参数其实不会变化, 最多就是状态变了
	// 因此我们这里没有必要
	criContainer := &cri.Container{
		Id: dc.ID,
		State: cri.ContainerState{
			Pid:       dc.State.Pid,
			StartedAt: dc.State.StartedAt,
			Status:    dc.State.Status,
		},
		// 这个就是对应k8s yaml里的 container name 注意不是 dc.Name 坑死了
		Name:      k8slabels.GetContainerName(dc.Config.Labels),
		Pod:       criPod,
		Labels:    dc.Config.Labels,
		Env:       parseEnv(dc.Config.Env),
		LogPath:   filepath.Join(core.GetHostfs(), dc.LogPath),
		Hostname:  dc.Config.Hostname,
		SandboxID: k8slabels.GetSandboxID(dc.Config.Labels),
		Runtime:   dc.HostConfig.Runtime,
	}

	criContainer.EnvTz = criContainer.Env["TZ"]

	// 识别 container 类型
	if l.isSandbox(criContainer) {
		criContainer.Sandbox = true
	} else if l.isSidecar(criContainer) {
		criContainer.Sidecar = true
	} else {
		criContainer.MainBiz = true
	}

	// dc.GraphDriver.Name == "overlay2"
	for k, v := range dc.GraphDriver.Data {
		if k == dockerutils.MergedDir && v != "" {
			criContainer.MergedDir = filepath.Join(core.GetHostfs(), v)
			break
		}
	}

	for _, m := range dc.Mounts {
		source := filepath.Join(core.GetHostfs(), m.Source)

		if !m.RW {
			// 不能读写 一般不是我们想要的挂载目录
			continue
		} else if st, err := os.Stat(source); err != nil {
			// 在宿主机上stat报错也不可行
			continue
		} else if !st.IsDir() {
			continue
		}

		criContainer.Mounts = append(criContainer.Mounts, &cri.MountPoint{
			Source:      source,
			Destination: m.Destination,
		})
	}

	if !criContainer.Sandbox {
		var err error
		if criContainer.IsRunning() {
			// pause 容器不需要
			// TODO 不推荐 TZ 环境变量
			criContainer.EtcLocaltime, err = l.getEtcTimezone(context.Background(), criContainer)
			if err != nil {
				logger.Metaz("[local] [docker] fail to parse /etc/localtime",
					zap.String("ns", criPod.Namespace), //
					zap.String("pod", criPod.Name),     //
					zap.String("cid", criContainer.Id),
					zap.Error(err))
			}
		}

		if criContainer.Hostname == "" {
			criContainer.Hostname, err = l.getHostname(context.Background(), criContainer)
			if err != nil {
				logger.Metaz("[local] [docker] fail to get hostname",
					zap.String("ns", criPod.Namespace), //
					zap.String("pod", criPod.Name),     //
					zap.String("cid", criContainer.Id),
					zap.Error(err))
			}
		}
	}

	criPod.All = append(criPod.All, criContainer)

	if criContainer.IsRunning() && !criContainer.Hacked && criContainer.MainBiz {
		// 仅对主容器这样做
		criContainer.Hacked = true
		if !strings.HasPrefix(criPod.Namespace, "kube-") && !criContainer.Sandbox {
			begin := time.Now()
			// TODO 或许我们直接复制到根目录下 /.holoinsight-agent-helper 这样更简单一些? 因为这样肯定不需要创建父目录
			// . 开头是为了隐藏文件
			err := l.CopyToContainer(context.Background(), criContainer, core.HelperToolLocalPath, core.HelperToolPath)
			cost := time.Now().Sub(begin)
			if err != nil {
				logger.Metaz("[local] [docker] hack error",
					zap.String("cid", criContainer.Id),
					zap.String("ns", criPod.Namespace),
					zap.String("pod", criPod.Name),
					zap.Duration("cost", cost),
					zap.Error(err))
			} else {
				logger.Metaz("[local] [docker] hack success",
					zap.String("cid", criContainer.Id),
					zap.String("ns", criPod.Namespace),
					zap.String("pod", criPod.Name),
					zap.Duration("cost", cost),
					zap.Error(err))
			}
		}
	}

	return criContainer
}

func (l *DockerLocalMetaImpl) handleOOM(msg events.Message) {
	ctr, ok := l.GetContainerByCid(msg.ID)
	if !ok || ctr.Sandbox {
		// 当发生oom时, sandbox和container都会产生oom
		return
	}

	logger.Metaz("[docker] [oom]",
		zap.String("ns", ctr.Pod.Namespace),
		zap.String("pod", ctr.Pod.Name),
		zap.String("container", ctr.Name),
		zap.Any("msg", msg))

	l.oomRecoder.add(ctr)
}

func (l *DockerLocalMetaImpl) emitOOMMetrics() {
	trg := trigger.WithFixedRate(time.Minute, 2*time.Second)

	next := trg.Next(nil)

	for {
		time.Sleep(next.Sub(time.Now()))
		alignTime := next.Add(-time.Minute - 2*time.Second)
		next = trg.Next(nil)
		record := l.oomRecoder.getAndClear()
		if len(record) == 0 {
			continue
		}

		// k8s_pod_oom
		var metrics []*model.Metric
		for _, item := range record {
			tags := common.ExtractContainerCommonTags(item.container)

			metrics = append(metrics, &model.Metric{
				// TODO 名字带有k8s对吗?
				Name:      "k8s_pod_oom",
				Tags:      tags,
				Timestamp: alignTime.UnixMilli(),
				Value:     float64(item.count),
			})
		}

		if gtw, err := gateway.Acquire(); err == nil {
			defer gateway.GatewaySingletonHolder.Release()

			begin := time.Now()
			_, err := gtw.WriteMetricsV1Extension2(context.Background(), nil, metrics)
			cost := time.Now().Sub(begin)

			logger.Infoz("[docker] [oom]", zap.Int("metrics", len(metrics)), zap.Duration("cost", cost), zap.Error(err))

		}

	}

}
