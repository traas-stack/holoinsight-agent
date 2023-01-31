package resources

import (
	"github.com/spf13/cast"
	"os"
	"runtime"
	"runtime/debug"
)

// SetupCpuAndMemoryLimit setups cpu and memory limit if running in k8s env
func SetupCpuAndMemoryLimit() {
	if os.Getenv("GOMAXPROCS") == "" {
		if str := os.Getenv("K8S_RESOURCES_LIMIT_CPU"); str != "" {
			if cpu := cast.ToInt(str); cpu > 0 {
				runtime.GOMAXPROCS(cpu)
			}
		}
	}

	if os.Getenv("GOMEMLIMIT") == "" {
		if str := os.Getenv("K8S_RESOURCES_LIMIT_MEMORY"); str != "" {
			if memory := cast.ToInt(str); memory > 0 {
				debug.SetMemoryLimit(int64(float64(memory) * 0.9))
			}
		}
	}
}
