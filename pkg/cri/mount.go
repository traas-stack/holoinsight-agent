/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cri

import "sort"

type (
	mountPointSortHelper []*MountPoint
)

func (m mountPointSortHelper) Len() int {
	return len(m)
}

func (m mountPointSortHelper) Less(i, j int) bool {
	// 降序排列
	return len(m[i].Source) > len(m[j].Source)
}

func (m mountPointSortHelper) Swap(i, j int) {
	t := m[i]
	m[i] = m[j]
	m[j] = t
}

func SortMountPointsByLongSourceFirst(mounts []*MountPoint) {
	sort.Sort(mountPointSortHelper(mounts))
}
