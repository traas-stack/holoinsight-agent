/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

// Package appconfig 应用程序级别的全局配置, 它在初始化顺序里是第一顺位, 不要依赖其他的业务包
package appconfig

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
)

var agentVersion string
var agentBuildTime string
var gitcommit string

const (
	defaultWorkspace = "default"
	defaultCluster   = "default"
)

var std = struct {
	// 是否处于dev模式
	dev bool
	wd  string
}{
	dev: false,
}

var (
	StdAgentConfig = AgentConfig{}
	k8sMode        bool
)

type (
	AgentConfig struct {
		App      string         `json:"app" yaml:"app" toml:"app"`
		ApiKey   string         `json:"apikey" yaml:"apikey" toml:"apikey"`
		Mode     core.AgentMode `json:"mode" yaml:"mode" toml:"mode"`
		Registry RegistryConfig `json:"registry" yaml:"registry" toml:"registry"`
		Gateway  GatewayConfig  `json:"gateway" yaml:"gateway" toml:"gateway"`
		Version  string         `json:"version" yaml:"version" toml:"version"`
		// 该配置存在的原因: toml 根表无法后置, 因此不太好在shell里进行配置 " echo "a=1" >> xxx.toml " 于是单独给它分配一个节
		Basic       *BasicConfig      `json:"basic" yaml:"basic" toml:"basic"`
		Central     CentralConfig     `json:"central" yaml:"central" toml:"central"`
		K8s         K8sConfig         `json:"k8s" yaml:"k8s" toml:"k8s"`
		Workspace   string            `json:"workspace" yaml:"workspace" toml:"workspace"`
		Cluster     string            `json:"cluster" yaml:"cluster" toml:"cluster"`
		Data        DataConfig        `json:"data" yaml:"data" toml:"data"`
		Daemonagent DaemonagentConfig `json:"daemonagent" yaml:"daemonagent" toml:"daemonagent"`
	}
	BasicConfig struct {
		App       string         `json:"app" yaml:"app" toml:"app"`
		ApiKey    string         `json:"apikey" yaml:"apikey" toml:"apikey"`
		Mode      core.AgentMode `json:"mode" yaml:"mode" toml:"mode"`
		Workspace string         `json:"workspace" yaml:"workspace" toml:"workspace"`
	}
	RegistryConfig struct {
		Addr        string `json:"addr" yaml:"addr" toml:"addr"`
		Secure      bool   `json:"secure" yaml:"secure" toml:"secure"`
		ServiceName string `json:"serviceName,omitempty" yaml:"serviceName" toml:"serviceName"`
		CaCert      string `json:"caCert,omitempty" yaml:"caCert" toml:"caCert"`
	}
	GatewayConfig struct {
		Addr        string `json:"addr" yaml:"addr" toml:"addr"`
		Secure      bool   `json:"secure" yaml:"secure" toml:"secure"`
		ServiceName string `json:"serviceName,omitempty" yaml:"serviceName" toml:"serviceName"`
		CaCert      string `json:"caCert,omitempty" yaml:"caCert" toml:"caCert"`
	}
	CentralConfig struct {
		Name                       string `json:"name" yaml:"name" toml:"name"`
		AlibabacloudMetricEndpoint string `json:"aliyunMetricEndpoint" yaml:"aliyunMetricEndpoint" toml:"aliyunMetricEndpoint"`
	}
	K8sConfig struct {
		Meta    K8sMetaConfig    `json:"meta,omitempty" yaml:"meta" toml:"meta"`
		Cri     K8sConfig_Cri    `json:"cri,omitempty" yaml:"cri" toml:"cri"`
		Metrics K8sMetricsConfig `json:"metrics" yaml:"metrics" toml:"metrics"`
	}
	K8sConfig_Cri struct {
		// docker pouch
		Type  string            `json:"type,omitempty" yaml:"type" toml:"type"`
		Pouch K8sCriPouchConfig `json:"pouch,omitempty" yaml:"pouch" toml:"pouch"`
	}
	K8sMetaConfig struct {
		AppRef          string                `json:"appRef,omitempty" yaml:"appRef" toml:"appRef"`
		HostnameRef     string                `json:"hostnameRef,omitempty" yaml:"hostnameRef" toml:"hostnameRef"`
		NodeHostnameRef string                `json:"nodeHostnameRef,omitempty" yaml:"nodeHostnameRef" toml:"nodeHostnameRef"`
		SidecarCheck    string                `json:"sidecarCheck,omitempty" yaml:"sidecarCheck" toml:"sidecarCheck"`
		Sandbox         K8sMetaConfig_Sandbox `json:"sandbox,omitempty" yaml:"sandbox" toml:"sandbox"`
	}
	K8sMetaConfig_Sandbox struct {
		Labels map[string]string `json:"labels,omitempty" yaml:"labels" toml:"labels"`
	}
	PrometheusConfig struct {
		AutoScapeK8s bool `json:"autoScapeK8s,omitempty" yaml:"autoScapeK8s" toml:"autoScapeK8s"`
	}
	DataConfig struct {
		Metric DataConfig_Metric `json:"metric" yaml:"metric" toml:"metric"`
	}
	DataConfig_Metric struct {
		RefLabels DataConfig_Metric_RefLabels `json:"refLabels" yaml:"refLabels" toml:"refLabels"`
		// SuppressedTags remove common tags whose keys are in this slice
		SuppressedTags []string `json:"suppressedTags,omitempty" yaml:"suppressedTags" toml:"suppressedTags"`
	}
	DataConfig_Metric_RefLabels struct {
		Items []*DataConfig_Metric_RefLabels_Item `json:"items" yaml:"items" toml:"items"`
	}
	// Ref a label as a tag
	DataConfig_Metric_RefLabels_Item struct {
		// Tag key
		Key string `json:"key" yaml:"key" toml:"key" `
		// Find first non-empty label
		Labels []string `json:"labels" yaml:"labels" toml:"labels" `
		// If all label values are empty, this DefaultValue will be used
		DefaultValue string `json:"defaultValue" yaml:"defaultValue" toml:"defaultValue" `
	}
	K8sMetricsConfig struct {
		FiveSecondsEnabled bool `json:"fiveSecondsEnabled,omitempty" yaml:"fiveSecondsEnabled" toml:"fiveSecondsEnabled"`
	}
	K8sCriPouchConfig struct {
		CpWorkaroundEnabled bool `json:"cpWorkaroundEnabled,omitempty" yaml:"cpWorkaroundEnabled"`
	}
	// DaemonagentConfig daemonagent config
	DaemonagentConfig struct {
		ClusterAgentEnabled bool `json:"clusterAgentEnabled" yaml:"clusterAgentEnabled"`
	}
)

func init() {
	k8sMode = os.Getenv("POD_NAMESPACE") != ""
}

func IsDev() bool {
	return std.dev
}

func SetDev(enabled bool) {
	std.dev = enabled
}

func SetupDevAppConfig(apikey string) error {
	StdAgentConfig.ApiKey = apikey
	return nil
}

func SetupAppConfig() error {

	// 默认值
	StdAgentConfig.Registry.Secure = true
	StdAgentConfig.Gateway.Secure = true

	// load from config file
	{
		fileBytes, err := os.ReadFile("agent.yaml")
		if os.IsNotExist(err) {
			fileBytes, err = os.ReadFile("conf/agent.yaml")
			if err == nil {
				fmt.Println("read conf/agent.yaml")
			}
		} else {
			fmt.Println("read agent.yaml")
		}
		if err == nil {
			err = yaml.Unmarshal(fileBytes, &StdAgentConfig)
			if err != nil {
				fmt.Fprintf(os.Stderr, "fail to parse agent.yaml\n%s\n", string(fileBytes))
			}
		}
	}
	{
		fileBytes, err := os.ReadFile("agent.toml")
		if os.IsNotExist(err) {
			fileBytes, err = os.ReadFile("conf/agent.toml")
		}
		if err == nil {
			err = toml.Unmarshal(fileBytes, &StdAgentConfig)
			if err != nil {
				fmt.Fprintf(os.Stderr, "fail to parse agent.toml\n%s\n", string(fileBytes))
			}
			if StdAgentConfig.Basic != nil {
				StdAgentConfig.ApiKey = StdAgentConfig.Basic.ApiKey
				StdAgentConfig.App = StdAgentConfig.Basic.App
				StdAgentConfig.Mode = StdAgentConfig.Basic.Mode
				StdAgentConfig.Workspace = StdAgentConfig.Basic.Workspace
			}
		}
	}

	if StdAgentConfig.Workspace == "" {
		StdAgentConfig.Workspace = "default"
	}

	// 确保一些目录存在
	os.MkdirAll("data", 0755)
	os.MkdirAll("logs", 0755)

	// load from env
	if s := os.Getenv("HI_APP"); s != "" {
		StdAgentConfig.App = s
	}
	if s := os.Getenv("HI_APIKEY"); s != "" {
		StdAgentConfig.ApiKey = s
	}
	if s := os.Getenv("HI_REGISTRY_ADDR"); s != "" {
		StdAgentConfig.Registry.Addr = s
	}

	if s := os.Getenv("HI_REGISTRY_SECURE"); s != "" {
		StdAgentConfig.Registry.Secure = cast.ToBool(s)
	}

	if s := os.Getenv("HI_GATEWAY_ADDR"); s != "" {
		StdAgentConfig.Gateway.Addr = s
	}
	if s := os.Getenv("HI_GATEWAY_SECURE"); s != "" {
		StdAgentConfig.Gateway.Secure = cast.ToBool(s)
	}
	if s := os.Getenv("HI_AGENT_MODE"); s != "" {
		mode, err := core.ParseAgentMode(s)
		if err != nil {
			return err
		}
		StdAgentConfig.Mode = mode
	}
	if s := os.Getenv("HI_CENTRAL_NAME"); s != "" {
		StdAgentConfig.Central.Name = s
	}
	if s := os.Getenv("HI_CENTRAL_ALIBABACLOUD_METRIC_ENDPOINT"); s != "" {
		StdAgentConfig.Central.AlibabacloudMetricEndpoint = s
	}

	if s := os.Getenv("HI_K8S_META_APPREFS"); s != "" {
		StdAgentConfig.K8s.Meta.AppRef = s
	}
	if s := os.Getenv("HI_K8S_META_EXT_APP"); s != "" {
		StdAgentConfig.K8s.Meta.AppRef = s
	}
	if s := os.Getenv("HI_K8S_META_EXT_HOSTNAME"); s != "" {
		StdAgentConfig.K8s.Meta.HostnameRef = s
	}
	if s := os.Getenv("HI_K8S_META_EXT_NODE_HOSTNAME"); s != "" {
		StdAgentConfig.K8s.Meta.NodeHostnameRef = s
	}
	if s := os.Getenv("HI_K8S_META_SIDECAR_CHECK"); s != "" {
		StdAgentConfig.K8s.Meta.SidecarCheck = s
	}

	if s := os.Getenv("HI_WORKSPACE"); s != "" {
		StdAgentConfig.Workspace = s
	}

	if s := os.Getenv("HI_CLUSTER"); s != "" {
		StdAgentConfig.Cluster = s
	}

	if StdAgentConfig.Workspace == "" {
		StdAgentConfig.Workspace = defaultWorkspace
	}
	if StdAgentConfig.Cluster == "" {
		StdAgentConfig.Cluster = defaultCluster
	}

	if StdAgentConfig.Mode == "" {
		// defaults to sidecar
		StdAgentConfig.Mode = core.AgentModeSidecar
	}
	StdAgentConfig.Version = agentVersion
	return nil
}

func ensureExist(dir string) error {
	if _, err := os.Stat(dir); err == nil {
		return nil
		// exist
	} else if os.IsNotExist(err) {
		// no exist
		return os.MkdirAll(dir, fs.ModePerm)
	} else {
		return err
	}
}

func IsInteractive() bool {
	return false
}

func (c *AgentConfig) IsDaemonset() bool {
	return c.Mode == core.AgentModeDaemonset
}

func IsK8sMode() bool {
	return k8sMode
}
