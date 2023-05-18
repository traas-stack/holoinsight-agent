/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"reflect"
	"unsafe"
)

// zero cost string conversion
// this may cause unexpected behavior, use this carefully
func String(b []byte) (s string) {
	if len(b) == 0 {
		return ""
	}
	return *(*string)(unsafe.Pointer(&b))
	//pbytes := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	//pstring := (*reflect.StringHeader)(unsafe.Pointer(&s))
	//pstring.Data = pbytes.Data
	//pstring.Len = pbytes.Len
	//return
}

// DeepCopyString deep copy string
func DeepCopyString(s string) string {
	if len(s) == 0 {
		return ""
	}
	var b []byte
	pbytes := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	pstring := (*reflect.StringHeader)(unsafe.Pointer(&s))
	pbytes.Data = pstring.Data
	pbytes.Len = pstring.Len
	pbytes.Cap = pstring.Len
	return string(b)
}

func DeepCopyStringSlice(a []string) []string {
	b := make([]string, len(a))
	for i := range a {
		b[i] = DeepCopyString(a[i])
	}
	return b
}

func ReverseStringSlice(a []string) {
	size := len(a)
	end := size >> 1
	for i := 0; i < end; i++ {
		t := a[i]
		a[i] = a[size-1-i]
		a[size-1-i] = t
	}
}

func StringSliceContains(elems []string, elem string) bool {
	for _, e := range elems {
		if elem == e {
			return true
		}
	}
	return false
}

func StringSliceFind(elems []string, elem string) int {
	for i, e := range elems {
		if elem == e {
			return i
		}
	}
	return -1
}

func ConvertStringSliceToHashSet(a []string) map[string]struct{} {
	if a == nil {
		return nil
	}
	m := make(map[string]struct{}, len(a))
	for _, s := range a {
		m[s] = struct{}{}
	}
	return m
}

func SubstringMax(s string, n int) string {
	if n < len(s) {
		return s[:n]
	} else {
		return s
	}
}

func TransformStringSlice(strs []string, transformer func(string) string) []string {
	ret := make([]string, len(strs))
	for i := range strs {
		ret[i] = transformer(strs[i])
	}
	return ret
}

// 返回第一个非空的字符串, 如果全部为空则返回空
func FirstNotEmpty(strs ...string) string {
	for _, str := range strs {
		if str != "" {
			return str
		}
	}
	return ""
}

func CopyStringSlice(src []string) []string {
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

// SubBytesMax returns a sub slice with max length limit
func SubBytesMax(bs []byte, max int) []byte {
	if len(bs) <= max {
		return bs
	}
	return bs[:max]
}
