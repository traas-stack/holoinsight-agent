package meta

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/cri"
	"sync"
)

type (
	oomRecoder struct {
		mutex  sync.Mutex
		record map[string]*recordItem
	}
	recordItem struct {
		container *cri.Container
		count     int
	}
)

func newOOMRecorder() *oomRecoder {
	return &oomRecoder{
		record: make(map[string]*recordItem),
	}
}

func (r *oomRecoder) add(ctr *cri.Container) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	item, ok := r.record[ctr.Id]
	if !ok {
		item = &recordItem{
			container: ctr,
			count:     0,
		}
		r.record[ctr.Id] = item
	}
	item.count++
}

func (r *oomRecoder) getAndClear() map[string]*recordItem {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	ret := r.record
	r.record = make(map[string]*recordItem)
	return ret
}
