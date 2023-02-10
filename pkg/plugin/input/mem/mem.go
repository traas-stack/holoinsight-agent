package mem

import (
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input"
	"github.com/shirou/gopsutil/v3/mem"
)

type (
	memInput struct {
		input.BaseInput
	}
)

func (m *memInput) Collect(ctx *input.CollectContext) ([]*model.DetailData, error) {
	d := model.NewDetailData()

	vm, err := mem.VirtualMemory()
	if err != nil {
		logger.Errorf("get memory stat error %+v", err)
	} else {
		d.Values["mem_total"] = vm.Total
		d.Values["mem_used"] = vm.Used
		d.Values["mem_avail"] = vm.Available
		d.Values["mem_free"] = vm.Free
		d.Values["mem_buff"] = vm.Buffers
		d.Values["mem_cach"] = vm.Cached
		d.Values["mem_dirty"] = vm.Dirty

		// 这里似乎asar的文档有点问题 它说 mem_util = 100 * (total - free) total
		// 但实际上 mem_util = 100 * used / total
		d.Values["mem_util"] = vm.UsedPercent
		d.Values["mem_rutil"] = 100.0 * float64(vm.Total-vm.Available) / float64(vm.Total)

		//d.Values["mem.swap.total"] = vm.SwapTotal
		//d.Values["mem.swap.free"] = vm.SwapFree
		//d.Values["mem.swap.cach"] = vm.Cached
	}

	//swap, err := mem.SwapMemory()
	//if err != nil {
	//	logger.Errorf("get swap memory stat error %+v", err)
	//} else {
	//	d.Values["swap.total"] = swap.Total
	//	d.Values["swap.free"] = swap.Free
	//	d.Values["swap.used"] = swap.Used
	//	if swap.Total > 0 {
	//		d.Values["swap.util"] = 100.0 * float64((swap.Total - swap.Free)) / float64(swap.Total)
	//	} else {
	//		d.Values["swap.util"] = 0.0
	//	}
	//}

	return model.MakeDetailDataSlice(d), nil
}
