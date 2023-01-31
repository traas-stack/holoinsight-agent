package core

import "os"

var hostfs = "/"

func init() {
	// TODO 这个类可以进一步抽象
	if s := os.Getenv("HOSTFS"); s != "" {
		hostfs = s
	}
}

func GetHostfs() string {
	return hostfs
}
