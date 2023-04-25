/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package strategy

import (
	"fmt"
	"testing"
)

func TestBackoffStrategy(t *testing.T) {
	reconnectStrategy := NewBackOffStrategy(0.3)

	for i := 0; i < 1000; i++ {
		fmt.Println(reconnectStrategy.GetNext())
	}

	t.Log("success")
}
