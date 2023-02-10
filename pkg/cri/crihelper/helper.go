package crihelper

import (
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
)

type (
	StatelessHelper interface {
		// 获取容器内部一个 pid 的信息
		GetProcessInfo(c *cri.Container, pid int) (*ProcessInfo, error)
		// 在容器内部做 glob
		Glob(c *cri.Container, pattern string) (interface{}, error)
		// 列出文件目录
		ListFiles(c *cri.Container, request *pb.ListFilesRequest) (*pb.ListFilesResponse, error)
	}

	StatefulHelper interface {
		// 打开一个文件用于后续读操作, 要返回一个句柄, 并且对端(必须长期运行)必须要持有文件句柄不关闭
		FileOpen(c *cri.Container, path string) (interface{}, error)
		FileSeek(c *cri.Container, fd interface{}) (interface{}, error)
		FileRead(c *cri.Container, fd interface{}) (interface{}, error)
		FileRead2(c *cri.Container, fd interface{}, offset, limit int) (interface{}, error)
		FileStat(c *cri.Container, fd interface{}) (interface{}, error)
		FileClose(c *cri.Container, fd interface{}) (interface{}, error)
	}
)
