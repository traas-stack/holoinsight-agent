package executor

import (
	"fmt"
	"regexp"
	"testing"
)

func TestRegexp2(t *testing.T) {
	r := regexp.MustCompile("^\\d{4}-\\d{2}-\\d{2}")
	fmt.Println(r.MatchString("2021-11-11 11:11:11"))
}

func TestRegexp(t *testing.T) {
	r := regexp.MustCompile("(ab)(c\\d)")
	// 找到第一个匹配
	fmt.Println(r.FindString("abc1abc2"))
	// 找到第一个匹配及其捕获组
	fmt.Println(r.FindStringSubmatch("abc1abc2"))
	// 找到所有匹配
	fmt.Println(r.FindAllString("abc1abc2", -1))
	// 找到所有匹配及其捕获组
	fmt.Println(r.FindAllStringSubmatch("abc1abc2", -1))
	//判断是否匹配正则表达式
	fmt.Println(r.MatchString("eabc1e"))
}
