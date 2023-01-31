package inspect

import (
	"encoding/json"
	"fmt"
	"github.com/TRaaSStack/holoinsight-agent/pkg/core"
	"github.com/TRaaSStack/holoinsight-agent/pkg/server/registry/pb"
	"reflect"
	"testing"
)

func TestName(t *testing.T) {
	var resp *pb.InspectResponse
	makeType := reflect.StructOf([]reflect.StructField{
		{
			Name:      "HelperBaseResp",
			Type:      reflect.TypeOf(core.HelperBaseResp{}),
			Anonymous: true,
		},
		{
			Name: "Data",
			Type: reflect.TypeOf(resp),
			Tag:  "json:\"data\"",
		},
	})

	value := reflect.New(makeType)
	fmt.Println(value.CanAddr())
	fmt.Println(value.CanInterface())

	i := value.Interface()

	err := json.Unmarshal([]byte("{\"message\":\"foo\"}"), i)
	if err != nil {
		panic(err)
	}

	fmt.Println(value.Elem().Field(0).Field(1).String())

	//type tempResp struct {
	//	core.HelperBaseResp
	//	Resp *pb.InspectResponse `json:"data"`
	//}
	//fmt.Println(((*tempResp)(unsafe.Pointer(value.Pointer()))).Message)
}
