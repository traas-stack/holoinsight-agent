package timeparser

import (
	"strconv"
	"strings"
	"time"
)

// findTimeStart 寻找年份的开始位置
func findTimeStart(log string) int {
	yearInt := time.Now().Year()
	yearStr := strconv.Itoa(yearInt)
	yearStart := strings.Index(log, yearStr)
	if yearStart == -1 {
		lastYear := yearInt - 1
		lastYearStr := strconv.Itoa(lastYear) //试试去年
		yearStart = strings.Index(log, lastYearStr)
	}
	return yearStart
}

// FillYearLeft 设置时间位置描述 这是一个猜测
func fillYearLeft(lt *TimeStyle, log string, timeStart int) {
	if timeStart == 0 {
		lt.YearLeftIndex = lineStart //行首
		return
	}
	delim := ""
	c := log[timeStart-1]
	if timeStart-2 >= 0 && log[timeStart-2] == c {
		cStr := string(c)
		if timeStart-3 >= 0 && log[timeStart-2] == c {
			delim = cStr + cStr + cStr //三个重复的分隔符
		} else {
			delim = cStr + cStr //两个重复的分隔符
		}
	} else {
		delim = string(c)
	}
	lt.YearLeft = delim
	tmpIndex := -1
	start := 0
	for start < timeStart {
		index := strings.Index(log[start:], lt.YearLeft)
		if index == -1 {
			break
		} else {
			tmpIndex++
			start = start + index + len(lt.YearLeft)
		}
	}
	lt.YearLeftIndex = tmpIndex
}
