/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package batch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestKeyBatchProcessor(t *testing.T) {
	threshold := 5
	prodNum := 10

	kc := &keyConsumer{t: t, threshold: threshold, result: make(map[string]map[int]struct{})}
	b := NewBatchProcessorWithKey(
		prodNum, kc,
		WithBatchThresholdStrategy(threshold),
		WithMaxWaitStrategy(time.Second))
	b.Run()
	defer b.Shutdown()

	// produce
	go func() {
		for i := 0; i < prodNum; i++ {
			if ok := b.TryPut("key", i); !ok {
				t.Logf("put %+v not ok", i)
			}
			time.Sleep(time.Millisecond)
		}
	}()

	time.Sleep(time.Second)
	kc.check("key", prodNum)
}

type (
	keyConsumer struct {
		t *testing.T

		threshold int
		result    map[string]map[int]struct{}
	}
)

func (kc *keyConsumer) Consume(key string, values []interface{}) {
	kc.t.Logf("[%+v] receive key: %s, len %d", time.Now(), key, len(values))
	assert.Equal(kc.t, true, kc.threshold >= len(values))

	for _, v := range values {
		vv, exist := kc.result[key]
		if !exist {
			vv = make(map[int]struct{})
			kc.result[key] = vv
		}
		vv[v.(int)] = struct{}{}
	}
}

func (kc *keyConsumer) check(key string, sum int) {
	for i := 0; i < sum; i++ {
		assert.Equal(kc.t, sum, len(kc.result[key]))

		if _, exist := kc.result[key]; !exist {
			kc.t.Errorf("key %v not exist", i)
		} else {
			if _, exist := kc.result[key][i]; !exist {
				kc.t.Errorf("data %v not exist", i)
			}
		}
	}
}
