/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAlignedTimer_1(t *testing.T) {
	{
		time.Sleep(time.Millisecond * time.Duration(1000-time.Now().UnixMilli()%1000))
	}

	start := time.Now()
	fmt.Println("start", start)
	at, _ := NewAlignedTimer(time.Second, 200*time.Millisecond, true, false)
	at.Next()

	count := 0
loop:
	for {
		select {
		case <-at.C:
			count++
			now := time.Now()
			fmt.Println(count, now)

			switch {
			case count <= 3:
				assert.Equal(t, now.Second()-start.Second()+1, count)
			case count > 4:
				assert.Equal(t, now.Second()-start.Second()-1, count)
			}

			if count == 3 {
				time.Sleep(2100 * time.Millisecond)
			}

			if count == 7 {
				at.Stop()
				break loop
			} else {
				at.Next()
			}
		}
	}
}

func TestAlignedTimer_2(t *testing.T) {
	{
		time.Sleep(time.Millisecond * time.Duration(1000-time.Now().UnixMilli()%1000+500))
	}

	start := time.Now()
	fmt.Println("start", start)
	at, _ := NewAlignedTimer(time.Second, 200*time.Millisecond, true, false)
	at.Next()

	count := 0
loop:
	for {
		select {
		case <-at.C:
			count++
			now := time.Now()
			fmt.Println(count, now)

			switch {
			case count <= 3:
				assert.Equal(t, now.Second()-start.Second(), count)
			case count > 4:
				assert.Equal(t, now.Second()-start.Second()-2, count)
			}

			if count == 3 {
				time.Sleep(2100 * time.Millisecond)
			}

			if count == 7 {
				at.Stop()
				break loop
			} else {
				at.Next()
			}
		}
	}
}

func TestAlignedTimer_3(t *testing.T) {
	{
		time.Sleep(time.Millisecond * time.Duration(1000-time.Now().UnixMilli()%1000+500))
	}

	start := time.Now()
	fmt.Println("start", start)
	at, _ := NewAlignedTimer(time.Second, 200*time.Millisecond, true, false)
	at.Next()

	count := 0
loop:
	for {
		select {
		case <-at.C:
			count++
			now := time.Now()
			fmt.Println(count, now)

			switch {
			case count <= 3:
				assert.Equal(t, now.Second()-start.Second(), count)
			case count > 4:
				assert.Equal(t, now.Second()-start.Second()-1, count)
			}

			if count == 3 {
				time.Sleep(1800 * time.Millisecond)
			}

			if count == 7 {
				at.Stop()
				break loop
			} else {
				at.Next()
			}
		}
	}
}
