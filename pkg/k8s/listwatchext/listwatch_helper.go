package listwatchext

import (
	"context"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/pager"
	"k8s.io/utils/clock"
	"math/rand"
	"sync"
	"time"
)

// TODO 这个类稍微有点复杂, 解释一下

var (
	// 这是k8s的默认值
	minWatchTimeout = 5 * time.Minute
)

type (
	ListFunc  func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error)
	WatchFunc func(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	ListWatch struct {
		ListFunc  ListFunc
		WatchFunc WatchFunc
	}
	ListWatchHelper struct {
		lw                  *ListWatch
		callback            ListWatchCallback
		stop                <-chan struct{}
		lastResourceVersion string
		mutex               sync.Mutex
		relistRequired      bool
	}
	ListResult struct {
		Result          runtime.Object
		ResourceVersion string
		PaginatedResult bool
		Err             error
	}
	ListWatchCallback struct {
		// 用户的方法不允许返回异常, 如果想返回异常, 请自己处理并调用reset重新开始监听
		OnList  func(items []runtime.Object)
		OnEvent func(e watch.Event)
	}
)

func NewListWatchHelper(lw *ListWatch, callback ListWatchCallback) *ListWatchHelper {
	return &ListWatchHelper{
		lw:       lw,
		callback: callback,
	}
}

func NewListWatchFromClient(c cache.Getter, resource string, namespace string) *ListWatch {
	listFunc := func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec).
			Do(ctx).
			Get()
	}
	watchFunc := func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec).
			Watch(ctx)
	}
	return &ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func untilSuccess(run func() bool, backoff wait.BackoffManager) {
	for {
		if run() {
			return
		}
		<-backoff.Backoff().C()
	}
}

func (h *ListWatchHelper) listUntilSuccess() {
	// list 退避
	listBackoff := wait.NewExponentialBackoffManager(800*time.Millisecond, 30*time.Second, 2*time.Minute, 2.0, 1.0, clock.RealClock{})

	untilSuccess(func() bool {
		var listCh = make(chan ListResult, 1)
		var panicCh = make(chan interface{}, 1)

		// 500 是k8s默认值
		go doPageList(context.Background(), pager.ListPageFunc(h.lw.ListFunc), metav1.ListOptions{Limit: 500}, listCh, panicCh)

		//p:=v1.Pod{}
		//p.Spec.Containers[0].Env[0].
		select {
		case <-h.stop:
			return true
		case p := <-panicCh:
			// TODO 在这种辅助类里调用 logger.Metaz 里是不是不太好
			logger.Metaz("[k8s] page list panic", zap.Any("panic", p))
			return false
		case r := <-listCh:
			// list完成, 记得检查err
			if r.Err != nil {
				logger.Metaz("[k8s] page list error", zap.Error(r.Err))
				return false
			}
			// 记录 lastResourceVersion 用于后续 watch
			h.lastResourceVersion = r.ResourceVersion
			list, err := meta.ExtractList(r.Result)
			if err != nil {
				logger.Metaz("[k8s] meta ExtractList error", zap.Error(r.Err))
				return false
			}
			h.callback.OnList(list)
			return true
		}
	}, listBackoff)
}

func (h *ListWatchHelper) watch0(ctx context.Context) {
	watchBackoff := wait.NewExponentialBackoffManager(800*time.Millisecond, 30*time.Second, 2*time.Minute, 2.0, 1.0, clock.RealClock{})

	var wi watch.Interface
	var err error

	// 第1个 for 是针对 watch 不断重试直到第一次 watch 成功
	for {
		if h.isStop() {
			return
		}

		// 随机打散
		timeoutSeconds := int64(minWatchTimeout.Seconds() * (rand.Float64() + 1.0))

		wi, err = h.lw.WatchFunc(ctx, metav1.ListOptions{
			AllowWatchBookmarks: true,
			ResourceVersion:     h.lastResourceVersion,
			TimeoutSeconds:      &timeoutSeconds,
		})
		if err != nil {
			// 这是可以重试的case, 不过要注意退避
			if utilnet.IsConnectionRefused(err) || apierrors.IsTooManyRequests(err) {
				<-watchBackoff.Backoff().C()
				continue
			}

			logger.Metaz("[k8s] watch error", zap.Error(err))
			// 出现这种错误时我们无法重试 只能重新list&watch
			return
		}
		break
	}

	// 第2层 for 是一旦我们 watch 成功就不断消费
	for {
		select {
		case <-h.stop:
			wi.Stop()
			return
		case e, ok := <-wi.ResultChan():

			// 这里可能是网络出了什么问题, 导致管道关闭了
			if !ok {
				logger.Metaz("[k8s] watch chan stopped")
				wi.Stop()
				return
			}

			if e.Type == watch.Error {
				err2 := apierrors.FromObject(e.Object)
				logger.Metaz("[k8s] watch returns Error event", zap.Any("event", e), zap.Error(err2))
				// 无法重试 只能重新来过
				<-watchBackoff.Backoff().C()
				wi.Stop()
				return
			}
			metaobj, err := meta.Accessor(e.Object)
			if err != nil {
				// 几乎不可能, 如果有那么就是100%必现的错误
				logger.Metaz("[k8s] meta accessor error", zap.Any("event", e), zap.Error(err))
				continue
			}
			h.lastResourceVersion = metaobj.GetResourceVersion()

			// Bookmark 表示这是一个仅仅包含 ResourceVersion 的 Event
			// 防止因为过滤导致客户端 ResourceVersion 不更新
			// 处理event
			if e.Type == watch.Bookmark {
				continue
			}
			h.callback.OnEvent(e)
		}
	}

}

func (h *ListWatchHelper) watch() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 异步watch
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		h.watch0(ctx)
	}()

	// 这是为了确保 当前 goroutine 能否快速响应 stop
	// 如果 watch0 能够快速响应 stop, 那么可以省掉我当前这些步骤
	select {
	case <-h.stop:
	case <-watchDone:
	}

}

func (h *ListWatchHelper) isStop() bool {
	select {
	case <-h.stop:
		return true
	default:
		return false
	}
}

func (h *ListWatchHelper) Run(stop <-chan struct{}) {
	h.stop = stop
	for {
		if h.isStop() {
			break
		}

		h.listUntilSuccess()

		if h.isStop() {
			break
		}

		h.watch()
	}
}

func doPageList(ctx context.Context, pageFunc pager.ListPageFunc, opts metav1.ListOptions, resultCh chan<- ListResult, panicCh chan<- interface{}) {
	// 分页list
	var lastResourceVersion string
	pagerInstance := pager.New(func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
		resp, err := pageFunc(ctx, opts)
		if err == nil {
			if metaObj, err2 := meta.Accessor(resp); err2 == nil {
				lastResourceVersion = metaObj.GetResourceVersion()
			}
		}
		return resp, err
	})

	var listRO runtime.Object
	var paginatedResult bool
	var err error

	defer func() {
		if r := recover(); r != nil {
			panicCh <- r
		}
	}()

	listRO, paginatedResult, err = pagerInstance.List(ctx, opts)

	if err == nil {
		if mobj, err2 := meta.ListAccessor(listRO); err2 == nil {
			lastResourceVersion = mobj.GetResourceVersion()
		}
	}

	resultCh <- ListResult{
		Result:          listRO,
		ResourceVersion: lastResourceVersion,
		PaginatedResult: paginatedResult,
		Err:             err,
	}
	close(resultCh)
}
