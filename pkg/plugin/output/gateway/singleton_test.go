package gateway

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSingleton(t *testing.T) {
	t.SkipNow()

	gs, err := defaultGatewaySingleton.acquire()
	if err != nil {
		panic(err)
	}

	assert.Equal(t, 1, defaultGatewaySingleton.refCount)

	fmt.Println(gs.Ping(context.Background()))
}
