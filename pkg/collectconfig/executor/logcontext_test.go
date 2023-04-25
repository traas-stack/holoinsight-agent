/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"fmt"
	"regexp"
	"testing"
)

func TestLogContextRegex(t *testing.T) {
	// 	r := regexp.MustCompile("^(?P<time>.*?),\\w* (?P<level>\\w+)\\s+\\[(?P<thread>\\w+)\\].*result=(?P<cost>\\w+)$")
	r := regexp.MustCompile("^(?P<time>.*?),.*? (?P<level>\\w+)\\s+\\[(?P<thread>.*?)]\\s+c\\.a\\.c\\.g\\.c\\.t\\.PressureGenerator - result=(?P<cost>.*)$")
	ss := r.FindStringSubmatch("2022-03-22 17:43:48,131 INFO  [common-scheduler-4] c.a.c.g.c.t.PressureGenerator - result=683.7297745504324")
	for _, s := range ss {
		fmt.Println(s)
	}
}
