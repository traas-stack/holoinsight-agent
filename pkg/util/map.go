/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"sort"
	"strings"
	"sync"
)

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

// MergeStringMapTo merges one string map to another
func MergeStringMapTo(a map[string]string, to map[string]string, override bool) map[string]string {
	if !override {
		for k, v := range a {
			if _, exist := to[k]; !exist {
				to[k] = v
			}
		}
	} else {
		for k, v := range a {
			to[k] = v
		}
	}
	return to
}

// BuildTagsKey build string from tags
// "key1=value1 key2=value2 key3=value3"
// key1 ... keyn is sorted
func BuildTagsKey(tags map[string]string) string {
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	sb := strings.Builder{}
	for _, key := range keys {
		sb.WriteString(key)
		sb.WriteByte('=')
		sb.WriteString(tags[key])
		sb.WriteByte(' ')
	}
	return sb.String()
}
