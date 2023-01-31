package loganalysis

import (
	"bytes"
	"net"
	"strings"
	"unicode"
)

type (
	LAPart struct {
		Content       string `json:"content"`
		latterContent *string
		Source        bool `json:"source"`
		Important     bool `json:"important"`
		Count         int  `json:"count"`
	}
)

const (
	similarPartsFactor = 0.8
	ignoreTypeLength   = 5
)

var (
	// one bytes
	simpleCutors = map[byte]bool{
		')': true,
		'!': true,
		';': true,
		'@': true,
		'[': true,
		'(': true,
		',': true,
		'?': true,
		'}': true,
		']': true,
		':': true,
		'^': true,
		'&': true,
		'=': true,
		'{': true,
		'*': true,
	}

	// 3bytes
	threeCharCutors = map[string]bool{
		"，":   true,
		"【":   true,
		"。":   true,
		"（":   true,
		"）":   true,
		"】":   true,
		"！":   true,
		"；":   true,
		"：":   true,
		" - ": true,
	}

	twoCharCutors = [2]byte{
		'-', '>',
	}
)

func (p *LAPart) getLatterContent(reuse *bytes.Buffer) string {
	if p.latterContent == nil {
		p.makeLatterContent(reuse)
	}
	return *p.latterContent
}

func isImportant(in string) bool {

	//太短了
	if len(in) <= 10 {
		return false
	}
	letterC := 0
	connectorC := 0

	for _, c := range in {
		if c == '@' {
			return false
		}

		if c == '.' || c == '_' || c == '-' {
			connectorC++
			if connectorC >= 2 {
				return false
			}
			continue
		}

		if unicode.IsLetter(c) || c == ' ' {
			letterC++
			continue
		}
	}

	return float64(letterC)/(float64(len(in))) > 0.9
}

func isSource(in string) bool {
	//太短了
	if len(in) <= 5 {
		return false
	}

	if strings.Index(in, "com.") == 0 || strings.Index(in, "java.") == 0 ||
		strings.Index(in, "org.") == 0 {
		return true
	}

	if strings.HasSuffix(in, ".com") {
		return true
	}

	if fastContainIp(in) {
		return true
	}

	// xflush有一些用户自定义的，通过文件导入先不管

	return false
}

func fastContainIp(log string) bool {
	in := []byte(log)

	indexs := make([]int, 0, 3)

	// 找到所有的.
	for i, c := range in {
		if c == '.' {
			indexs = append(indexs, i)
		}
	}

	if len(indexs) < 3 {
		return false
	}

	for currntIndex := 0; currntIndex+3 <= len(indexs); currntIndex++ {
		if isIp(indexs[currntIndex:currntIndex+3], in) {
			return true
		}
	}

	return false
}

func (p *LAPart) makeLatterContent(reuse *bytes.Buffer) {
	if reuse == nil {
		reuse = bytes.NewBuffer(nil)
	} else {
		reuse.Reset()
	}
	for _, c := range p.Content {
		if unicode.IsLetter(c) {
			reuse.WriteRune(c)
		}
	}
	str := reuse.String()
	p.latterContent = &str
}

func isCut(before string, c byte) (bool, int) {
	// latin
	if _, ok := simpleCutors[c]; ok {
		return true, 0
	}

	size := len(before)
	if size >= 1 {
		last := before[size-1]
		if twoCharCutors[0] == last && twoCharCutors[1] == byte(c) {
			return true, 1
		}
	}

	if size >= 2 {
		t := make([]byte, 3, 3)
		t[0] = before[size-2]
		t[1] = before[size-1]
		t[2] = c
		if _, ok := threeCharCutors[string(t)]; ok {
			return true, 2
		}
	}

	return false, 0
}

func skipWhenSimilar(f *LAPart) bool {
	if f.Source {
		return true
	}
	if len(f.Content) < ignoreTypeLength {
		return true
	}
	return false
}

func isSimilar(fs, ts []*LAPart) bool {
	small := 0
	big := 0

	if len(fs) > len(ts) {
		big = len(fs)
		small = len(ts)
	} else {
		big = len(ts)
		small = len(fs)
	}

	if float64(small)/float64(big) < similarPartsFactor {
		return false
	}

	similar := 0
	total := 0

	var reuse bytes.Buffer
	for _, f := range fs {
		if skipWhenSimilar(f) {
			continue
		}

		found := false

		for _, t := range ts {
			// TODO t也要check skipWhenSimilar 吗?
			if isSimilarPart(&reuse, f, t) {
				found = true
				break
			}
		}
		partSize := len(f.Content)

		sAdd, tAdd := 0, partSize
		if found {
			sAdd = partSize
		}

		if f.Important {
			sAdd *= 2
		} else {
			tAdd /= 2
		}

		similar += sAdd
		total += tAdd
	}

	//奇葩的日志，没错误类型
	if total == 0 {
		return true
	}

	return (float64(similar) / float64(total)) >= similarPartsFactor
}

func isSimilarPart(reuse *bytes.Buffer, t, f *LAPart) bool {
	return t.getLatterContent(reuse) == f.getLatterContent(reuse)
}

func dissembleParts(input string) []*LAPart {

	// 去掉时间戳
	// 2015-02-27 14:39:05,565
	if len(input) > 23 {
		if input[4] == '-' && input[7] == '-' {
			input = input[23:]
		}
	}

	ret := make([]*LAPart, 0)

	var current string
	nextStartLeft := 0
	nextStartRight := 0

	for i := 0; i < len(input); i++ {
		char := input[i]
		ok, backCount := isCut(current, char)
		if ok {
			last := len(current) - backCount
			content := strings.TrimSpace(current[:last])
			if len(content) > 0 {
				part := &LAPart{
					Content:   content,
					Source:    isSource(content),
					Important: isImportant(content),
					Count:     1}
				ret = append(ret, part)
			}
			current = ""

			nextStartLeft = i + 1
			nextStartRight = nextStartLeft
		} else {
			nextStartRight++
			if nextStartRight <= len(input) {
				current = input[nextStartLeft:nextStartRight]
			}
		}
	}

	if len(current) > 0 {
		current = strings.TrimSpace(current)
		part := &LAPart{
			Content:   current,
			Source:    isSource(current),
			Important: isImportant(current),
			Count:     1}
		ret = append(ret, part)
	}

	return ret
}

func isIp(index []int, in []byte) bool {
	// 11.222.33.33 2个点之间差太远了
	if index[1]-index[0] > 4 || index[2]-index[1] > 4 {
		return false
	}

	leftMost := index[0]

	for leftShitTime := 0; leftShitTime < 3; leftShitTime++ {
		leftMost--

		if leftMost < 0 {
			leftMost = 0
			break
		}

		// x.11.22.33
		if !isDigit(in[leftMost]) {
			// 左1不是数字,直接失败
			if leftShitTime == 0 {
				return false
			}

			// 左边就到这里为止了
			// left
			// |
			// 11.11.11.11
			leftMost++
			break
		}
	}

	rightMost := index[2]
	for rightShitTime := 0; rightShitTime < 3; rightShitTime++ {
		rightMost++

		if rightMost >= len(in)-1 {
			rightMost = len(in) - 1
			break
		}

		if !isDigit(in[rightMost]) {
			// 11.11.22.x
			if rightShitTime == 0 {
				return false
			}

			// 左边就到这里为止了
			//           right
			//           |
			// 11.11.11.11
			rightMost = rightMost - 1
			break
		}
	}

	return net.ParseIP(string(in[leftMost:rightMost+1])) != nil
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
