/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	json2 "encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func loadTransformConf(name string) (*collectconfig.TransformConf, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var conf *collectconfig.TransformConf

	switch filepath.Ext(name) {
	case ".json":
		err = json2.NewDecoder(file).Decode(&conf)
	case ".yal":
		fallthrough
	case ".yaml":
		err = yaml.NewDecoder(file).Decode(&conf)
	default:
		return nil, errors.New("unsupported file ext: " + name)
	}
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func loadTransformFilter(name string) (XTransformFilter, error) {
	conf, err := loadTransformConf(name)
	if err != nil {
		return nil, err
	}
	return parseTransform(conf)
}

func TestTransform_switchcase_1(t *testing.T) {
	c, err := loadTransformFilter("transforms/switchcase_1.json")
	assert.NoError(t, err)
	assert.NotNil(t, c)

	v, err := c.Filter(&LogContext{contextValue: "aaa"})
	assert.NoError(t, err)
	assert.Equal(t, "aaa_aaa_aaa", v)

	v, err = c.Filter(&LogContext{contextValue: "bbb"})
	assert.NoError(t, err)
	assert.Equal(t, "bbb_bbb", v)

	v, err = c.Filter(&LogContext{contextValue: "ccc"})
	assert.NoError(t, err)
	assert.Equal(t, "ccc_ccc", v)
}

func TestTransform_simple_const(t *testing.T) {
	c, err := loadTransformFilter("transforms/const.json")
	assert.NoError(t, err)
	assert.NotNil(t, c)

	v, err := c.Filter(&LogContext{contextValue: "aaa"})
	assert.NoError(t, err)
	assert.Equal(t, "match case 1", v)

	v, err = c.Filter(&LogContext{contextValue: "bbb"})
	assert.NoError(t, err)
	assert.Equal(t, "match case 2", v)

	v, err = c.Filter(&LogContext{contextValue: "no exist"})
	assert.NoError(t, err)
	assert.Equal(t, "other case", v)
}

func TestTransform_mapping_1(t *testing.T) {
	c, err := loadTransformFilter("transforms/mapping1.yaml")
	assert.NoError(t, err)
	assert.NotNil(t, c)

	v, err := c.Filter(&LogContext{contextValue: "a"})
	assert.NoError(t, err)
	assert.Equal(t, "aa", v)

	v, err = c.Filter(&LogContext{contextValue: "b"})
	assert.NoError(t, err)
	assert.Equal(t, "bb", v)

	v, err = c.Filter(&LogContext{contextValue: "no exist"})
	assert.NoError(t, err)
	assert.Equal(t, "xx", v)
}

func TestTransform_substring_1(t *testing.T) {
	c, err := loadTransformFilter("transforms/substring_1.yaml")
	assert.NoError(t, err)
	assert.NotNil(t, c)

	v, err := c.Filter(&LogContext{contextValue: "holoinsight"})
	assert.NoError(t, err)
	assert.Equal(t, "insight", v)
}

func TestTransform_substring_2(t *testing.T) {
	c, err := loadTransformFilter("transforms/substring_2.yaml")
	assert.NoError(t, err)
	assert.NotNil(t, c)

	v, err := c.Filter(&LogContext{contextValue: "holoinsight"})
	assert.NoError(t, err)
	assert.Equal(t, "in", v)
}

func TestTransform_substring_3(t *testing.T) {
	c, err := loadTransformFilter("transforms/substring_3.yaml")
	assert.NoError(t, err)
	assert.NotNil(t, c)

	v, err := c.Filter(&LogContext{contextValue: "holoinsight"})
	assert.NoError(t, err)
	assert.Equal(t, "", v)
}

func TestTransform_substring_error(t *testing.T) {
	c, err := loadTransformFilter("transforms/substring_error.yaml")
	assert.NoError(t, err)
	assert.NotNil(t, c)

	_, err = c.Filter(&LogContext{contextValue: "holoinsight"})
	assert.Error(t, err)
}

func Test_FillDefaultElect(t *testing.T) {
	w := &collectconfig.Where{
		And: nil,
		Or:  nil,
		Not: nil,
		Contains: &collectconfig.MContains{
			Value:      "asd",
			Multiline:  false,
			IgnoreCase: false,
		},
		ContainsAny:   nil,
		In:            nil,
		NumberBetween: nil,
		Regexp:        nil,
		NumberOp:      nil,
	}
	fillDefaultElect(w)
	assert.Equal(t, collectconfig.EElectContext, w.Contains.Elect.Type)
}

func TestTransform_regexp1(t *testing.T) {
	c, err := loadTransformFilter("transforms/regexp1.yaml")
	assert.NoError(t, err)
	assert.NotNil(t, c)

	ret, err := c.Filter(&LogContext{contextValue: "holoinsight"})
	assert.NoError(t, err)
	assert.Equal(t, "HoloinsightXXX", ret)

	fmt.Println(regexp.MatchString("a", "aa"))
}

func TestTransform_regexp2(t *testing.T) {
	c, err := loadTransformFilter("transforms/regexp2.yaml")
	assert.NoError(t, err)
	assert.NotNil(t, c)

	ret, err := c.Filter(&LogContext{contextValue: "holoinsight"})
	assert.NoError(t, err)
	assert.Equal(t, "HoloinsightXXX", ret)
}
