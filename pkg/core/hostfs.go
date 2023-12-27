/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package core

import (
	"os"
	"path/filepath"
	"strings"
)

var hostfs = "/hostfs"

func init() {
	if s := os.Getenv("HOSTFS"); s != "" {
		hostfs = s
	}
}

func GetHostfs() string {
	return hostfs
}

func FindFirstSockInHostfs(env string, defaultAddrs ...string) string {
	var addrs []string
	if s := os.Getenv(env); s != "" {
		separator := ","
		switch {
		case strings.Contains(s, ","):
			separator = ","
		case strings.Contains(s, ":"):
			separator = ":"
		}

		for _, addr := range strings.Split(s, separator) {
			if addr = strings.TrimSpace(addr); addr != "" {
				addrs = append(addrs, addr)
			}
		}
	}
	addrs = append(addrs, defaultAddrs...)

	for _, path := range addrs {
		addr := filepath.Join(GetHostfs(), path)
		if st, err := os.Stat(addr); err == nil && st.Mode()&os.ModeSocket == os.ModeSocket {
			return addr
		}
	}
	return ""
}
