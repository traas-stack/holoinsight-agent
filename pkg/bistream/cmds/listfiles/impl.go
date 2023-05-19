/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package listfiles

import (
	"errors"
	commonpb "github.com/traas-stack/holoinsight-agent/pkg/server/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util/fs2"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultMaxDepth   = 10
	defaultMaxVisited = 4096
)

func getFileNode(path string) (*commonpb.FileNode, error) {
	stat, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	n := &commonpb.FileNode{
		Name: stat.Name(),
	}

	if stat.Mode()&os.ModeSymlink == os.ModeSymlink {
		n.Symbol = true

		stat, path, err = fs2.Stat2(path, 3)
		if err != nil {
			return n, err
		}
	}

	if stat.IsDir() {
		n.Dir = true
	} else {
		n.Stat = &commonpb.FileInfo{ //
			Size:    stat.Size(),                //
			ModTime: stat.ModTime().UnixMilli(), //
			Mode:    int32(stat.Mode()),         //
		}
	}

	return n, nil
}

func dfsGetFileNode(stat os.FileInfo, path string, depth, maxDepth int32, visited *int, exts map[string]struct{}) (*commonpb.FileNode, error) {
	*visited++
	n := &commonpb.FileNode{
		Name: stat.Name(),
	}

	// respect symbol
	if stat.Mode()&os.ModeSymlink == os.ModeSymlink {
		n.Symbol = true

		var err error
		var link string
		link, err = os.Readlink(path)
		if err != nil {
			return n, err
		}
		stat, err = os.Stat(link)
		if err != nil {
			return n, nil
		}
		path = link
	}

	if stat.IsDir() {
		n.Dir = true
		entries, err := os.ReadDir(path)
		if err != nil {
			return n, nil
		}
		if depth < maxDepth {
			for _, entry := range entries {

				// limit visited file nodes
				if *visited >= defaultMaxVisited {
					break
				}

				if strings.HasPrefix(entry.Name(), ".") {
					continue
				}
				if !entry.IsDir() && len(exts) > 0 {
					ext := filepath.Ext(entry.Name())
					if strings.HasPrefix(ext, ".") {
						ext = ext[1:]
					}
					if _, ok := exts[ext]; !ok {
						continue
					}
				}
				info, err := entry.Info()
				if err != nil {
					continue
				}
				child, err := dfsGetFileNode(info, filepath.Join(path, entry.Name()), depth+1, maxDepth, visited, exts)
				if err == nil && child != nil {
					n.Children = append(n.Children, child)
				}
			}
		}
	} else {
		n.Stat = &commonpb.FileInfo{
			Size:    stat.Size(),
			ModTime: stat.ModTime().UnixMilli(),
			Mode:    int32(stat.Mode()),
		}
	}
	return n, nil
}

func ListFiles(req *pb.ListFilesRequest, resp *pb.ListFilesResponse) error {
	path := filepath.Clean(req.Name)

	if !filepath.IsAbs(path) {
		return errors.New("must be a absolute dir path")
	}

	exts := make(map[string]struct{})
	for _, s := range req.IncludeExts {
		exts[s] = struct{}{}
	}

	firstStat, err := os.Lstat(path)
	if err != nil {
		return err
	}

	maxDepth := req.MaxDepth
	if maxDepth <= 0 || maxDepth > defaultMaxDepth {
		maxDepth = defaultMaxDepth
	}

	var visited int
	n, err := dfsGetFileNode(firstStat, path, 0, req.MaxDepth, &visited, exts)
	if err != nil {
		return err
	}

	if req.GetIncludeParents() {
		root := n
		for path != "/" && len(path) > 0 {
			dir := filepath.Dir(path)
			if dir == "/" {
				break
			}
			parent, err := getFileNode(dir)
			if err != nil {
				return err
			}
			parent.Children = []*commonpb.FileNode{root}
			root = parent
			path = dir
		}
		resp.Nodes = []*commonpb.FileNode{
			root,
		}
	} else {
		resp.Nodes = []*commonpb.FileNode{
			n,
		}
	}
	return nil
}

// Rebase nodes from one dir to another
// /Users/xzchaoo/logs/a.log -> /home/admin/logs/a.log
func Rebase(root []*commonpb.FileNode, from, to string) (*commonpb.FileNode, error) {
	children, err := removePrefixNodes(root[0], from)
	if err != nil {
		return nil, err
	}

	return appendPrefixNodes(children, to)
}

// appendPrefixNodes append prefix nodes to existing nodes
// /Users/xzchaoo/logs/a.log + /a/b -> /a/b/Users/xzchaoo/logs/a.log
func appendPrefixNodes(root []*commonpb.FileNode, dir string) (*commonpb.FileNode, error) {
	newRoot, last, err := makeDirTree(dir)
	if err != nil {
		return nil, err
	}
	last.Children = root
	return newRoot, nil
}

// removePrefixNodes append remove nodes from existing nodes
// /Users/xzchaoo/logs/a.log - /Usersb -> /xzchaoo/logs/a.log
func removePrefixNodes(root *commonpb.FileNode, dir string) ([]*commonpb.FileNode, error) {
	dir = filepath.Clean(dir)
	if !strings.HasPrefix(dir, string(os.PathSeparator)) {
		return nil, errors.New("must be a absolute path")
	}
	ss := strings.Split(dir, string(os.PathSeparator))
	for i, segment := range ss {
		if segment == "" {
			continue
		}
		if root.Name != segment {
			return nil, errors.New("nodes is not prefix with dir")
		}
		if i+1 < len(ss) {
			if len(root.Children) != 1 {
				return nil, errors.New("nodes is not prefix with dir")
			}
			root = root.Children[0]
		}
	}
	return root.Children, nil
}

func makeDirTree(dir string) (*commonpb.FileNode, *commonpb.FileNode, error) {
	dir = filepath.Clean(dir)

	if !strings.HasPrefix(dir, string(os.PathSeparator)) {
		return nil, nil, errors.New("must be a absolute path")
	}

	ss := strings.Split(dir, string(os.PathSeparator))
	var newRoot *commonpb.FileNode
	var current *commonpb.FileNode
	for _, segment := range ss {
		if segment == "" {
			continue
		}
		node := &commonpb.FileNode{
			Name:  segment,
			Dir:   true,
			Exist: true,
		}
		if newRoot == nil {
			newRoot = node
			current = node
		} else {
			current.Children = append(current.Children, node)
			current = node
		}
	}

	return newRoot, current, nil
}
