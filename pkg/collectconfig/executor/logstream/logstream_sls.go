/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package logstream

import (
	"encoding/gob"
	"errors"
	"fmt"
	aliyunsls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	slsSourceKey  = "__source__"
	slsShardKey   = "__shard__"
	slsContentKey = "content"
)

type (
	SlsConfig struct {
		Endpoint string
		AK       string
		SK       string
		Project  string
		Logstore string
		Shard    int
	}
	SlsSecretProvider func(SlsConfig) (string, string)
	slsStateObj       struct {
		Cursor       int64
		Cache        map[int64]*ReadResponse
		RemoteCursor string
	}
	slsSubLogStream struct {
		g            *GLogStream
		config       SlsConfig
		client       aliyunsls.ClientInterface
		remoteCursor string
	}
)

var (
	SlsSecretProviders []SlsSecretProvider
	dnsCacheHelper     *util.DnsCacheHelper
	slsHttpclient      *http.Client
)

func init() {
	SlsSecretProviders = append(SlsSecretProviders, envSlsSecretProvider)
	gob.Register(&slsStateObj{})

	dnsCacheHelper = util.NewDnsCacheHelper()
	dnsCacheHelper.Start()
	slsHttpclient = dnsCacheHelper.NewHttpClient()
}

func envSlsSecretProvider(config SlsConfig) (string, string) {
	ss := strings.Split(strings.TrimSpace(os.Getenv("SLS_SECRET")), ",")
	if len(ss) == 2 {
		return ss[0], ss[1]
	}

	// ak, sk
	return "", ""
}

func (sc *SlsConfig) BuildKey() string {
	return fmt.Sprintf("%s/%s/%d", sc.Project, sc.Logstore, sc.Shard)
}

func NewSlsLogStream(config SlsConfig) *GLogStream {
	ak := config.AK
	sk := config.SK

	if ak == "" || sk == "" {
		for _, provider := range SlsSecretProviders {
			if ak, sk = provider(config); ak != "" && sk != "" {
				break
			}
		}
	}

	g := &GLogStream{
		Key: config.BuildKey(),
	}

	client := &aliyunsls.Client{
		Endpoint:        config.Endpoint,
		AccessKeyID:     ak,
		AccessKeySecret: sk,
		RequestTimeOut:  5 * time.Second,
		RetryTimeOut:    5 * time.Second,
		HTTPClient:      slsHttpclient,
	}
	sub := &slsSubLogStream{
		g:      g,
		config: config,
		client: client,
	}
	g.sub = sub
	return g
}

func IsSlsLogStream(ls LogStream) bool {
	switch x := ls.(type) {
	case *GLogStream:
		_, ok := x.sub.(*slsSubLogStream)
		return ok
	default:
		return false
	}
}

func (s *slsSubLogStream) SaveState() (interface{}, error) {
	state := &slsStateObj{
		Cursor:       s.g.Cursor,
		RemoteCursor: s.remoteCursor,
		Cache:        make(map[int64]*ReadResponse),
	}
	s.g.Cache.Range(func(key, value any) bool {
		index := key.(int64)
		resp := value.(*cachedRead).resp
		state.Cache[index] = resp
		return true
	})
	return state, nil
}

func (s *slsSubLogStream) LoadState(i interface{}) error {
	if i == nil {
		return nil
	}
	state, ok := i.(*slsStateObj)
	if !ok {
		return errors.New("not sls state")
	}

	s.g.Cursor = state.Cursor
	s.remoteCursor = state.RemoteCursor
	for index, resp := range state.Cache {
		s.g.Cache.Store(index, &cachedRead{
			pendingReads: 0,
			resp:         resp,
		})
		s.g.UpdatePending(resp, true)
	}
	return nil
}

func (s *slsSubLogStream) Start() {
}

func (s *slsSubLogStream) Stop() {
	s.client.Close()
}

func (s *slsSubLogStream) Read(resp *ReadResponse) error {
	if s.remoteCursor == "" {
		remoteCursor, err := s.client.GetCursor(s.config.Project, s.config.Logstore, s.config.Shard, aliyunsls.OffsetNewest)
		if err != nil {
			return err
		}
		s.remoteCursor = remoteCursor
	}

	slgs, remoteNextCursor, err := s.client.PullLogs(s.config.Project, s.config.Logstore, s.config.Shard, s.remoteCursor, "", 1000)
	if err != nil {
		return err
	}

	if false {
		begin, _ := s.client.GetCursorTime(s.config.Project, s.config.Logstore, s.config.Shard, s.remoteCursor)
		end, _ := s.client.GetCursorTime(s.config.Project, s.config.Logstore, s.config.Shard, remoteNextCursor)
		//		resp.Range = fmt.Sprintf("%s:%s", f.remoteCursor, remoteNextCursor)
		resp.Range = fmt.Sprintf("%s-%s", begin.Format(time.RFC3339), end.Format(time.RFC3339))
	} else {
		resp.Range = fmt.Sprintf("%s:%s", s.remoteCursor, remoteNextCursor)
	}
	if s.remoteCursor == remoteNextCursor {
		// empty
		return nil
	}

	for _, slg := range slgs.LogGroups {
		lg := &LogGroup{
			Tags: make(map[string]string, len(slg.LogTags)+1),
			Logs: make([]*Log, 0, len(slg.Logs)),
		}
		lg.Tags[slsSourceKey] = slg.GetSource()
		lg.Tags[slsShardKey] = cast.ToString(s.config.Shard)
		for _, tag := range slg.LogTags {
			lg.Tags[tag.GetKey()] = tag.GetValue()
		}
		resp.Count += len(slg.Logs)
		for _, slog := range slg.Logs {
			log := &Log{
				Time:     int64(slog.GetTime()) * 1000,
				Contents: make(map[string]string, len(slog.Contents)),
			}
			for _, c := range slog.Contents {
				key := c.GetKey()
				log.Contents[key] = c.GetValue()
				if key == slsContentKey {
					resp.Lines = append(resp.Lines, c.GetValue())
				}
			}
			lg.Logs = append(lg.Logs, log)
		}
		resp.LogGroups = append(resp.LogGroups, lg)
	}

	resp.HasMore = true
	s.remoteCursor = remoteNextCursor

	return nil
}

func (s *slsSubLogStream) CreateResponse(cursor int64) *ReadResponse {
	return &ReadResponse{
		Cursor:      cursor,
		IOStartTime: time.Now(),
		Path:        fmt.Sprintf("%s/%s/%d", s.config.Project, s.config.Logstore, s.config.Shard),
	}
}
