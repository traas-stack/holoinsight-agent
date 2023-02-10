package mem

import (
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input"
)

type (
	swapInput struct {
		input.BaseInput
	}
)

func (m *swapInput) Collect(ctx *input.CollectContext) ([]*model.DetailData, error) {
	d := model.NewDetailData()

	vm, err := mem.VirtualMemory()
	if err != nil {
		logger.Errorf("get memory stat error %+v", err)
	} else {
		d.Values["swap_total"] = vm.SwapTotal
		d.Values["swap_free"] = vm.SwapFree
		d.Values["swap_cach"] = vm.Cached
	}

	// 这个貌似取不到值 算了
	//swap, err := mem.SwapMemory()
	//if err != nil {
	//	logger.Errorf("get swap memory stat error %+v", err)
	//} else {
	//	d.Values["swap2.total"] = swap.Total
	//	d.Values["swap2.free"] = swap.Free
	//	d.Values["swap2.used"] = swap.Used
	//	if swap.Total > 0 {
	//		d.Values["swap2.util"] = 100.0 * float64((swap.Total - swap.Free)) / float64(swap.Total)
	//	} else {
	//		d.Values["swap2.util"] = 0.0
	//	}
	//}

	return model.MakeDetailDataSlice(d), nil
}
