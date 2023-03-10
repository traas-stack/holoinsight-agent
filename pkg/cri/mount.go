package cri

import "sort"

type (
	mountPointSortHelper []*MountPoint
)

func (m mountPointSortHelper) Len() int {
	return len(m)
}

func (m mountPointSortHelper) Less(i, j int) bool {
	// ιεΊζε
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
