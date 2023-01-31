package httpcheckw

import (
	"encoding/json"
	"fmt"
	"github.com/TRaaSStack/holoinsight-agent/pkg/collecttask"
	"github.com/TRaaSStack/holoinsight-agent/pkg/pipeline/telegraf/providers"
	"github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input/httpcheck"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	defaultBodyLimit = 1024 * 1024
	maxBodyLimit     = 1024 * 1024
	defaultSchema    = "http"
	defaultPort      = 80
	defaultTimeout   = 3 * time.Second
)

type (
	Config struct {
		URL            string   `json:"url"`
		Schema         string   `json:"schema"`
		Port           int      `json:"port"`
		Path           string   `json:"path"`
		Timeout        int      `json:"timeout"`
		Method         string   `json:"method"`
		SuccessCodes   []int    `json:"successCodes"`
		SuccessRegexps []string `json:"successRegexps"`
		BodyLimit      int64    `json:"bodyLimit"`
		NetworkMode    string   `json:"networkMode"`
	}
	Input struct{}
)

func init() {
	providers.Register("httpcheck", Parse)
}

func Parse(task *collecttask.CollectTask) (interface{}, error) {
	config := &Config{}
	err := json.Unmarshal(task.Config.Content, config)
	if err != nil {
		return nil, err
	}

	target := task.Target

	schema := config.Schema
	if schema == "" {
		schema = defaultSchema
	}
	port := config.Port
	if port <= 0 {
		port = defaultPort
	}

	timeout := defaultTimeout
	if config.Timeout > 0 {
		timeout = time.Duration(config.Timeout) * time.Millisecond
	}

	method := config.Method
	if method == "" {
		method = http.MethodGet
	}

	bodyLimit := int64(defaultBodyLimit)
	if config.BodyLimit > 0 {
		bodyLimit = config.BodyLimit
	}
	if config.BodyLimit < maxBodyLimit {
		bodyLimit = maxBodyLimit
	}

	successRegexps := make([]*regexp.Regexp, 0, len(config.SuccessRegexps))
	for _, successRegexp := range config.SuccessRegexps {
		r, err := regexp.Compile(successRegexp)
		if err == nil {
			successRegexps = append(successRegexps, r)
		}
	}

	path := config.Path
	if path != "" && strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	url := config.URL
	if target.IsTypePod() {
		url = fmt.Sprintf("%s://%s:%d%s", schema, target.GetIP(), port, path)
	} else if target.IsTypeLocalhost() {
		url = fmt.Sprintf("%s://%s:%d%s", schema, "localhost", port, path)
	} else {
		return nil, fmt.Errorf("unsupported target type %+v", target)
	}

	return &httpcheck.Input{
		Config: &httpcheck.Config{
			URL:            url,
			Timeout:        timeout,
			Method:         method,
			BodyLimit:      bodyLimit,
			SuccessCodes:   config.SuccessCodes,
			SuccessRegexps: successRegexps,
			NetworkMode:    config.NetworkMode,
		},
	}, nil

}
