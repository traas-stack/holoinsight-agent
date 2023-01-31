package api

import (
	"encoding/json"
	"github.com/TRaaSStack/holoinsight-agent/pkg/model"
	"time"
)

type (
	// TODO 改个名字, 底层不一定用 nsenter 去实现, 也可能用 exec API, 只要能实现了就行
	// 如果某个插件依赖 nsenter 那么它需要实现该接口
	// 执行流程
	// [由实例去实现] 调用 SerializeRequest, 实例将要传递的入参序列化成 []byte
	// [由实例去实现] nsneter 切换到指定ns, 然后执行 helper bin, 内部会 new 出相同的 input 实例, 然后调用其 ExecuteRequest 方法
	// [由实例去实现] 实现 ExecuteRequest 方法, 入参是上一步返回的 []byte
	// [由流程去实现] 流程调用 ProcessResponse, 入参上一步的返回的 []byte
	// 这样做的好处是, 插件自身只是要处理数据的序列化, 不需要处理 nsenter 的调用流程
	InputExtNsEnter interface {
		// 执行该插件的网络模式, 目前应该返回 "POD"
		NetworkMode() string

		// 产出请求参数
		// 入参: 状态, actionType, 请求体, 超时时间, 错误
		// 状态: 一般为nil, 可用于 ProcessResponse 关联状态
		// actionType: 将会传递给 helper 程序的 actionType
		SerializeRequest() (interface{}, string, []byte, time.Duration, error)

		// 处理请求
		// 入参: 请求体
		ExecuteRequest([]byte) ([]byte, error)

		// 处理响应
		// 入参: 状态, 响应体, 错误, 结果收集器
		ProcessResponse(interface{}, []byte, error, Accumulator) error
	}
)

func NsEnterHelpProcesResponse(respBytes []byte, accumulator Accumulator) error {
	var metrics []*model.Metric

	err := json.Unmarshal(respBytes, &metrics)
	if err != nil {
		return err
	}

	for _, metric := range metrics {
		accumulator.AddMetric(metric)
	}

	return nil
}
