/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package silence

import (
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/meta"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"sync"
	"sync/atomic"
	"time"
)

type (
	PodUpdateListener struct {
		clientset          *kubernetes.Clientset
		mutex              sync.Mutex
		stopCh             chan struct{}
		controllerManagers []ControllerManager
		podReady           int32
	}
	// A helper interface for ControllerManager impl
	Helper interface {
		IsPodReady() bool
		OnPodAdd(obj interface{})
		OnPodDel(obj interface{})
		EnterSilence(pod *v1.Pod)
		LeaveSilence(pod *v1.Pod)
	}
)

var (
	controllerManagerProviders []func(helper Helper) ControllerManager
)

func RegisterControllerManagerProviders(p func(helper Helper) ControllerManager) {
	controllerManagerProviders = append(controllerManagerProviders, p)
}

func NewPodOperationListener(clientset *kubernetes.Clientset) *PodUpdateListener {
	l := &PodUpdateListener{
		clientset: clientset,
		stopCh:    make(chan struct{}),
	}
	l.controllerManagers = append(l.controllerManagers, newDefaultControllerInformer(l))
	for _, provider := range controllerManagerProviders {
		l.controllerManagers = append(l.controllerManagers, provider(l))
	}
	return l
}

func (l *PodUpdateListener) IsPodReady() bool {
	return atomic.LoadInt32(&l.podReady) == 1
}

func (l *PodUpdateListener) Start() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	listWatch := cache.NewListWatchFromClient(l.clientset.CoreV1().RESTClient(), string(v1.ResourcePods), v1.NamespaceAll, fields.Everything())
	informer := cache.NewSharedInformer(listWatch, &v1.Pod{}, 0)

	for _, mgr0 := range l.controllerManagers {
		mgr := mgr0
		go mgr.Run(l.stopCh)
		ph := mgr.PodHandler()
		controlled := make(map[types.UID]struct{})
		informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*v1.Pod)
				if mgr.IsUnderControl(pod) {
					controlled[pod.UID] = struct{}{}
					if l.IsPodReady() {
						ph.OnAdd(obj)
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				pod := newObj.(*v1.Pod)
				if _, ok := controlled[pod.UID]; ok {
					ph.OnUpdate(oldObj, newObj)
				}
			},
			DeleteFunc: func(obj interface{}) {
				pod := obj.(*v1.Pod)
				if _, ok := controlled[pod.UID]; ok {
					delete(controlled, pod.UID)
					ph.OnDelete(obj)
				}
			},
		})
	}
	informer.SetWatchErrorHandler(func(r *cache.Reflector, err error) {
		logger.Errorf("[k8s-op] [pod] watch error %+v", err)
	})

	go informer.Run(l.stopCh)

	go func() {
		for {
			if informer.HasSynced() {
				logger.Metaz("[k8s-op] PodUpdateListener init done")
				atomic.StoreInt32(&l.podReady, 1)
				break
			}
			if l.isStopped() {
				return
			}
			logger.Metaz("[k8s-op] PodUpdateListener wait")
			time.Sleep(time.Second)
		}
	}()
}

func (l *PodUpdateListener) Stop() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.isStopped() {
		return
	}

	close(l.stopCh)
}

func (l *PodUpdateListener) isStopped() bool {
	select {
	case <-l.stopCh:
		return true
	default:
		return false
	}
}

func (l *PodUpdateListener) OnPodAdd(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		return
	}
	logger.Metaz("[k8s-op] [pod] added", //
		zap.String("namespace", pod.Namespace), //
		zap.String("pod", pod.Name))            //
	l.EnterSilence(pod)
}

func (l *PodUpdateListener) OnPodDel(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		return
	}
	logger.Metaz("[k8s-op] [pod] deleted", //
		zap.String("namespace", pod.Namespace), //
		zap.String("pod", pod.Name))            //

	l.EnterSilence(pod)
}

func (l *PodUpdateListener) EnterSilence(pod *v1.Pod) {
	logger.Metaz("[k8s-op] [pod] enter silence", zap.String("ns", pod.Namespace), zap.String("name", pod.Name))

	tags := meta.ExtractPodCommonTags(pod)
	tags["event"] = "enter"
	ioc.RegistryService.ReportEventAsync(&pb.ReportEventRequest_Event{
		EventTimestamp: time.Now().UnixMilli(),
		EventType:      "DIGEST",
		PayloadType:    "pod_silence",
		Tags:           tags,
	})

}

func (l *PodUpdateListener) LeaveSilence(pod *v1.Pod) {
	logger.Metaz("[k8s-op] [pod] leave silence", zap.String("ns", pod.Namespace), zap.String("name", pod.Name))

	tags := meta.ExtractPodCommonTags(pod)
	tags["event"] = "leave"
	ioc.RegistryService.ReportEventAsync(&pb.ReportEventRequest_Event{
		EventTimestamp: time.Now().UnixMilli(),
		EventType:      "DIGEST",
		PayloadType:    "pod_silence",
		Tags:           tags,
	})
}
