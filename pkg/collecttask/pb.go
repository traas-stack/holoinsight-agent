package collecttask

import (
	"fmt"
	"github.com/TRaaSStack/holoinsight-agent/pkg/logger"
	"github.com/TRaaSStack/holoinsight-agent/pkg/server/registry/pb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type (
	CommonResources struct {
		Configs map[string]*CollectConfig
		Targets map[string]*CollectTarget
	}
)

// 转成我们的业务模型
func toBuckets(resp *pb.GetCollectTasksResponse) (map[string]*BucketInfo, error) {
	// 公共部分
	configs := map[string]*CollectConfig{}
	targets := map[string]*CollectTarget{}
	for k, pbConfig := range resp.GetCollectConfigs() {
		configs[k] = toCollectConfig(pbConfig)
	}
	for k, pbTarget := range resp.GetCollectTargets() {
		targets[k] = toCollectTarget(pbTarget)
	}

	buckets := make(map[string]*BucketInfo, len(resp.GetBuckets()))
	for bucketKey, pbBucket := range resp.GetBuckets() {
		bi := &BucketInfo{
			key:   bucketKey,
			state: pbBucket.State,
			tasks: make(map[string]*CollectTask),
		}
		buckets[bucketKey] = bi
		for _, pbTask := range pbBucket.CollectTasks {
			refConfig, ok := configs[pbTask.GetCollectConfigKey()]
			if !ok {
				logger.Configz("miss config", zap.String("key", pbTask.GetKey()))
				continue
			}
			refTarget, ok := targets[pbTask.GetCollectTargetKey()]
			if !ok {
				logger.Configz("miss dim", zap.String("key", pbTask.GetKey()))
				continue
			}
			task := &CollectTask{
				Key:     pbTask.GetKey(),
				Version: fmt.Sprintf("%s/%s", refConfig.Version, refTarget.Version),
				Config:  refConfig,
				Target:  refTarget,
			}
			bi.tasks[task.Key] = task
		}
	}
	return buckets, nil
}

func toCollectConfig(config *pb.CollectConfig) *CollectConfig {
	return &CollectConfig{
		Key:     config.GetKey(),
		Type:    config.GetType(),
		Version: config.GetVersion(),
		Content: config.GetContent(),
	}
}

func toCollectTarget(target *pb.CollectTarget) *CollectTarget {
	return &CollectTarget{
		Key:     target.GetKey(),
		Type:    target.GetType(),
		Version: target.GetVersion(),
		Meta:    target.GetMeta(),
	}
}

func toPbCollectTargetBytes(target *CollectTarget) []byte {
	b, _ := proto.Marshal(&pb.CollectTarget{
		Key:     target.Key,
		Type:    target.Type,
		Version: target.Version,
		Meta:    target.Meta,
	})
	return b
}

func toPbCollectConfigBytes(target *CollectConfig) []byte {
	b, _ := proto.Marshal(&pb.CollectConfig{
		Key:     target.Key,
		Type:    target.Type,
		Version: target.Version,
		Content: target.Content,
	})
	return b
}
