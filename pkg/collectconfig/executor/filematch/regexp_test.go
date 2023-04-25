/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package filematch

import (
	"fmt"
	"github.com/vjeantet/grok"
	"testing"
)

func TestRegexp(t *testing.T) {
	// golang的命名捕获组

	// /Users/xzchaoo/logs/{file}.log
	// /Users/xzchaoo/logs/(?P<file>[^/]+).log

	//r := regexp.MustCompile("^/Users/xzchaoo/logs/(?P<name>[^/]+)\\.log$")
	//fmt.Println(len(r.SubexpNames()))

	g, err := grok.NewWithConfig(&grok.Config{NamedCapturesOnly: true})
	if err != nil {
		panic(err)
	}

	g.AddPattern("f", "/Users/xzchaoo/logs/%{DATA:file}.log")
	fmt.Println(g.Parse("/Users/xzchaoo/logs/%{DATA:file}.log", "/Users/xzchaoo/logs/1.log"))

	// "%{TIMESTAMP_ISO8601:time} %{DATA:level}%{SPACE}\\[%{DATA:thread}\\] c\\.a\\.c\\.g\\.c\\.t\\.PressureGenerator - result=%{NUMBER:cost}"
}
