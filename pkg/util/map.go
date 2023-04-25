/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import "sync"

func CopyStringMap(m map[string]string) map[string]string {
	newMap := make(map[string]string, len(m))
	for k, v := range m {
		newMap[k] = v
	}
	return newMap
}

func CopyStringMapCap(m map[string]string, cap int) map[string]string {
	newMap := make(map[string]string, cap)
	for k, v := range m {
		newMap[k] = v
	}
	return newMap
}

func SyncMapSize(m *sync.Map) int {
	size := 0
	m.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return size
}
