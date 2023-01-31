package util

import (
	"os"
	"runtime"
)

var isLinux = runtime.GOOS == "linux"

func IsLinux() bool {
	return isLinux
}

func GetEnvOrDefault(name, defaultValue string) string {
	s := os.Getenv(name)
	if s == "" {
		s = defaultValue
	}
	return s
}
