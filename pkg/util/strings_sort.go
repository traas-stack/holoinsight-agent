package util

type (
	SortStringsByLength []string
)

func (s SortStringsByLength) Len() int {
	return len(s)
}

func (s SortStringsByLength) Less(i, j int) bool {
	return len(s[i]) < len(s[j])
}

func (s SortStringsByLength) Swap(i, j int) {
	t := s[i]
	s[i] = s[j]
	s[j] = t
}
