package batch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
)

func TestBatchProcessor(t *testing.T) {
	logger.TestMode()
	threshold := 5
	prodNum := 100

	pc := &producerConsumer{t: t, threshold: threshold, result: make(map[int]struct{})}
	b := NewBatchProcessor(prodNum, pc, WithBatchThresholdStrategy(threshold), WithMaxWaitStrategy(time.Second))
	b.Run()
	defer b.Shutdown()
	pc.bp = b

	// produce
	go func() {
		for i := 0; i < prodNum; i++ {
			if ok := pc.bp.TryPut(i); !ok {
				t.Logf("put %+v not ok", i)
			}
			time.Sleep(time.Millisecond)
		}
	}()

	time.Sleep(time.Second)
	pc.check(prodNum)
}

func TestBatchProcessor_Wait(t *testing.T) {
	logger.TestMode()
	threshold := 5
	prodNum := 100

	pc := &producerConsumer{t: t, threshold: threshold, result: make(map[int]struct{})}
	b := NewBatchProcessor(prodNum, pc, WithBatchThresholdStrategy(threshold), WithMaxWaitStrategy(time.Second))
	b.Run()
	defer b.Shutdown()
	pc.bp = b

	// produce
	go func() {
		for i := 0; i < prodNum+1; i++ {
			if ok := pc.bp.TryPut(i); !ok {
				t.Logf("put %+v not ok", i)
			}
			time.Sleep(time.Millisecond)
		}
	}()

	time.Sleep(time.Second * 2)
	pc.check(prodNum + 1)
}

func TestBatchProcessor_Shutdown(t *testing.T) {
	logger.TestMode()
	threshold := 5
	prodNum := 100

	pc := &producerConsumer{t: t, threshold: threshold, result: make(map[int]struct{})}
	b := NewBatchProcessor(prodNum, pc, WithBatchThresholdStrategy(threshold), WithMaxWaitStrategy(time.Second))
	b.Run()
	pc.bp = b

	b.Shutdown()

	if ok := pc.bp.TryPut(0); !ok {
		t.Logf("put not ok")
	}

	time.Sleep(time.Second)
	pc.check(1)
}

func TestBatchProcessor_Weight(t *testing.T) {
	logger.TestMode()
	threshold := 4
	prodNum := 100

	pc := &producerConsumer{t: t, threshold: threshold, result: make(map[int]struct{})}
	b := NewBatchProcessor(prodNum, pc,
		WithBatchThresholdStrategy(threshold),
		WithItemsWeightStrategy(func(i interface{}) int {
			return i.(int)
		}, 10))
	b.Run()
	pc.bp = b
	defer b.Shutdown()

	// produce
	go func() {
		for i := 0; i < prodNum; i++ {
			if ok := pc.bp.TryPut(i); !ok {
				t.Logf("put %+v not ok", i)
			}
			time.Sleep(time.Millisecond)
		}
	}()

	time.Sleep(time.Second)
	pc.check(prodNum)
}

type (
	producerConsumer struct {
		bp Processor
		t  *testing.T

		threshold int
		result    map[int]struct{}
	}
)

func (pc *producerConsumer) Consume(values []interface{}) {
	pc.t.Logf("[%+v] receive len %d", time.Now(), len(values))
	assert.Equal(pc.t, true, pc.threshold >= len(values))

	for _, v := range values {
		pc.result[v.(int)] = struct{}{}
	}
}

func (pc *producerConsumer) check(sum int) {
	assert.Equal(pc.t, sum, len(pc.result))

	for i := 0; i < sum; i++ {
		if _, exist := pc.result[i]; !exist {
			pc.t.Errorf("%v not exist", i)
		}
	}
}
