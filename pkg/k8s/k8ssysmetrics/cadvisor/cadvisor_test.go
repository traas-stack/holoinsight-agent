package cadvisor

import (
	"fmt"
	"github.com/TRaaSStack/holoinsight-agent/pkg/k8s/k8slabels"
	cadvisorclientv1 "github.com/google/cadvisor/client"
	v1 "github.com/google/cadvisor/info/v1"
	"github.com/spf13/cast"
	"testing"
)

func TestCadvisor(t *testing.T) {
	{
		c, err := cadvisorclientv1.NewClient("http://127.0.0.1:8080")
		if err != nil {
			panic(err)
		}
		containers, err := c.SubcontainersInfo("", &v1.ContainerInfoRequest{NumStats: 1})
		for _, container := range containers {
			if len(container.Stats) == 0 {
				continue
			}
			if container.Namespace != "docker" {
				continue
			}
			stat := container.Stats[len(container.Stats)-1]
			used := uint64(0)
			for i := range stat.Filesystem {
				used += stat.Filesystem[i].Usage
			}

			fmt.Println(container.Name + " " + k8slabels.GetPodName(container.Spec.Labels) + " " + cast.ToString(used/1024/1024))
			//if k8slabels.GetNamespace(container.Spec.Labels) == "holoinsight-server" {
			//}
		}
	}
	//{
	//
	//	c, err := cadvisorclientv2.NewClient("http://127.0.0.1:8080")
	//	if err != nil {
	//		panic(err)
	//	}
	//	infos, err := c.Stats("/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod4e302eb6_40ef_43d3_b9bc_5e4ed5a96b11.slice", &v2.RequestOptions{
	//		Count:     1,
	//		Recursive: true,
	//		// MaxAge:    nil,
	//	})
	//	fmt.Println(err)
	//	fmt.Println(util.ToJsonString(infos))
	//
	//}
	//fmt.Println(c.VersionInfo())
	//mi, _ := c.MachineInfo()
	//fmt.Println(util.ToJsonString(mi))
	//ar, _ := c.Attributes()
	//fmt.Println(util.ToJsonString(ar))
	//fmt.Println(c.MachineStats())
	//containers, err := c.SubcontainersInfo("", &v1.ContainerInfoRequest{NumStats: 1})
	//if err != nil {
	//	panic(err)
	//}
	//for _, container := range containers {
	//	if k8slabels.GetNamespace(container.Spec.Labels) == "holoinsight-server" {
	//		fmt.Println(util.ToJsonString(container))
	//	}
	//}
}
