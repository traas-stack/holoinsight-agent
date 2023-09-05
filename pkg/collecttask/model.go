/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package collecttask

const (
	TargetLocalhost = "localhost"
	TargetPod       = "pod"
	TargetSlsShard  = "sls_shard"
	TargetNone      = "none"
	TargetContainer = "container"

	// ext
	TargetObNodeTenant = "ob_node_tenant"
)

type (
	// CollectTask = CollectConfig + CollectTarget
	CollectTask struct {
		// collect task key
		Key string `json:"key"`
		// collect task version
		Version string `json:"version"`
		// collect config
		Config *CollectConfig `json:"config"`
		// collect target
		Target *CollectTarget `json:"target"`
	}

	// 描述一个采集配置
	CollectConfig struct {
		Key  string `json:"key"`
		Type string `json:"type"`
		// 需要有一个版本号之类的东西, key相同的通过version判断新旧, 不用判断新旧, 只要版本不一样就直接替换
		Version string `json:"version"`
		// 站在reg的角度, 它就是给agent发 "一坨配置", 就是一堆二进制格式的数据
		// 怎么去解释这些数据是业务上的事情, 产品层与agent需要约定好内容格式
		Content []byte `json:"-"`
		// ContentObj is for internal usage
		ContentObj interface{} `json:"contentObj"`
	}

	// 采集目标
	CollectTarget struct {
		Key string `json:"key"`
		// localhost 其实现在只有 localhost/pod
		Type    string `json:"type"`
		Version string `json:"version"`
		// Meta的内容根据type解释
		Meta map[string]string `json:"meta"`
	}
)

func (t *CollectTask) IsDifferentWith(o *CollectTask) bool {
	return t.Config.Version != o.Config.Version || t.Target.Version != o.Target.Version || isMapEqual(t.Target.Meta, o.Target.Meta)
}

func isMapEqual(m1, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k, v1 := range m1 {
		if v2, ok := m2[k]; !ok || v1 != v2 {
			return false
		}
	}
	return true
}

func (t *CollectTarget) IsTypePod() bool {
	return t.Type == TargetPod || t.Meta["type"] == TargetPod
}

func (t *CollectTarget) IsTypeLocalhost() bool {
	return t.Type == TargetLocalhost
}

func (t *CollectTarget) GetIP() string {
	return t.Meta["ip"]
}

func (t *CollectTarget) GetApp() string {
	return t.Meta["app"]
}

func (t *CollectTarget) GetHostname() string {
	return t.Meta["hostname"]
}

func (t *CollectTarget) GetNamespace() string {
	return t.Meta["namespace"]
}

func (t *CollectTarget) GetPodName() string {
	return t.Meta["pod"]
}

func (t *CollectTarget) GetTenant() string {
	return t.Meta["tenant"]
}
