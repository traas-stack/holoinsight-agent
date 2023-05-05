/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"regexp"
	"strings"
)

const (
	tokenTypeStr tokenType = iota
	tokenTypeRef
)

type (
	tokenType            uint8
	xRegexpReplaceFilter struct {
		regexp      *regexp.Regexp
		conf        *collectconfig.TransformFilterRegexpReplaceV1
		strReplacer *strReplacer
	}
	token struct {
		tokenType tokenType
		str       string
		index     int
	}
	strReplacer struct {
		tokens []*token
	}
)

var (
	// refPattern is the regexp pattern for '$1' '${1}'
	refPattern = regexp.MustCompile("\\$(\\{\\d+}|\\d+)")
)

func (r *strReplacer) replace(groups []string) string {
	sb := strings.Builder{}

	for _, t := range r.tokens {
		if t.tokenType == tokenTypeStr {
			sb.WriteString(t.str)
		} else {
			if t.index < len(groups) {
				sb.WriteString(groups[t.index])
			} else {
				sb.WriteString(t.str)
			}
		}
	}

	return sb.String()
}

func (x *xRegexpReplaceFilter) Init() error {
	if x.conf.Expression != "" {
		r, err := regexp.Compile(x.conf.Expression)
		if err != nil {
			return err
		}
		x.regexp = r
	}

	strReplacer, err := parseStrReplacer(x.conf.Replacement)
	if err != nil {
		return err
	}
	x.strReplacer = strReplacer

	return nil
}

func parseStrReplacer(str string) (*strReplacer, error) {
	sr := &strReplacer{}
	submatches := refPattern.FindAllStringSubmatchIndex(str, -1)
	visitedOffset := 0
	for _, submatchIndex := range submatches {
		matched := str[submatchIndex[0]:submatchIndex[1]]
		group1 := str[submatchIndex[2]:submatchIndex[3]]
		if group1[0] == '{' {
			group1 = group1[1 : len(group1)-1]
		}
		index, err := cast.ToIntE(group1)
		if err != nil {
			return nil, err
		}
		if visitedOffset < submatchIndex[0] {
			sr.tokens = append(sr.tokens, &token{
				tokenType: tokenTypeStr,
				str:       str[visitedOffset:submatchIndex[0]],
			})
		}
		sr.tokens = append(sr.tokens, &token{
			tokenType: tokenTypeRef,
			str:       matched,
			index:     index,
		})
		visitedOffset = submatchIndex[1]
	}
	if visitedOffset < len(str) {
		sr.tokens = append(sr.tokens, &token{
			tokenType: tokenTypeStr,
			str:       str[visitedOffset:],
		})
	}
	return sr, nil
}

func (x *xRegexpReplaceFilter) Filter(ctx *LogContext) (interface{}, error) {
	if x.regexp != nil {
		str := cast.ToString(ctx.contextValue)
		groups := x.regexp.FindStringSubmatch(str)
		if len(groups) == 0 {
			return str, nil
		}

		if x.strReplacer != nil {
			ret := x.strReplacer.replace(groups)
			//return "o:" + str + " n:" + ret, nil
			return ret, nil
		}
		return str, nil
	} else {
		return x.strReplacer.replace(ctx.columns), nil
	}
}
