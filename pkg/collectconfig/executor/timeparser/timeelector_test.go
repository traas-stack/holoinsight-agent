/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package timeparser

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNormal(t *testing.T) {
	{
		s := "2021-01-02 03:04:05 INFO foo bar msg"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1609527845000), te.Parse(ts, s, time.Local))
	}
	{
		s := "2021-01-02 03:04:05,123 INFO foo bar msg"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1609527845123), te.Parse(ts, s, time.Local))
	}
	{
		s := "[2021-01-02 03:04:05] INFO foo bar msg"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1609527845000), te.Parse(ts, s, time.Local))
	}
	{
		s := "[[2021-01-02 03:04:05]] INFO foo bar msg"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1609527845000), te.Parse(ts, s, time.Local))
	}
	{
		s := "[[02/Jan/2021 03:04:05]] INFO foo bar msg"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1609527845000), te.Parse(ts, s, time.Local))
	}

	{
		s := "[[2021 Jan/02 03:04:05]] INFO foo bar msg"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1609527845000), te.Parse(ts, s, time.Local))
	}
	{
		s := "[[Jan 02 2021 03:04:05]] INFO foo bar msg"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1609527845000), te.Parse(ts, s, time.Local))
	}

	{
		s := "[[01/02/2021 03:04:05]] INFO foo bar msg"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1609527845000), te.Parse(ts, s, time.Local))
	}

	{
		s := "2023/06/15 16:04:01.270 +08:00"
		te, ts := ParseTimeStyle(s)
		// fmt.Println(ts)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1686816241270), te.Parse(ts, s, time.Local))
	}

	{
		s := "2023-06-15T16:24:11.123+07:00"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1686821051123), te.Parse(ts, s, time.Local))
		assert.Equal(t, ts.Layout, "2006-01-02T15:04:05.000Z07:00")
	}
	{
		s := "2023-06-15T16:24:11,123+07:00"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1686821051123), te.Parse(ts, s, time.Local))
		assert.Equal(t, ts.Layout, "2006-01-02T15:04:05.000Z07:00")
	}
	{
		s := "2023-06-15T16:24:11.123 +07:00"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1686821051123), te.Parse(ts, s, time.Local))
		assert.Equal(t, ts.Layout, "2006-01-02T15:04:05.000 Z07:00")
	}
	{
		s := "2023-06-15T16:24:11,123 +07:00"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1686821051123), te.Parse(ts, s, time.Local))
		assert.Equal(t, ts.Layout, "2006-01-02T15:04:05.000 Z07:00")
	}

	{
		s := "2023-06-15T16:24:11.123Z"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1686846251123), te.Parse(ts, s, time.Local))
		assert.Equal(t, ts.Layout, "2006-01-02T15:04:05.000Z07:00")
	}
	{
		s := "2023-06-15T16:24:11.123+00:00"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1686846251123), te.Parse(ts, s, time.Local))
		assert.Equal(t, ts.Layout, "2006-01-02T15:04:05.000Z07:00")
	}
	{
		s := "2023-06-15T16:24:11.123+07:00"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1686821051123), te.Parse(ts, s, time.Local))
		assert.Equal(t, ts.Layout, "2006-01-02T15:04:05.000Z07:00")
	}

	{
		s := "2023-06-15T16:24:11.123 Z"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1686846251123), te.Parse(ts, s, time.Local))
		assert.Equal(t, ts.Layout, "2006-01-02T15:04:05.000 Z07:00")
	}
	{
		s := "2023-06-15T16:24:11.123 +00:00"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1686846251123), te.Parse(ts, s, time.Local))
		assert.Equal(t, ts.Layout, "2006-01-02T15:04:05.000 Z07:00")
	}
	{
		s := "2023-06-15T16:24:11.123 +07:00"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1686821051123), te.Parse(ts, s, time.Local))
		assert.Equal(t, ts.Layout, "2006-01-02T15:04:05.000 Z07:00")
	}

	{
		s := "2023-06-15T16:24:11 Z"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1686846251000), te.Parse(ts, s, time.Local))
		assert.Equal(t, ts.Layout, "2006-01-02T15:04:05 Z07:00")
	}
	{
		s := "2023-06-15T16:24:11 +00:00"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1686846251000), te.Parse(ts, s, time.Local))
		assert.Equal(t, ts.Layout, "2006-01-02T15:04:05 Z07:00")
	}
	{
		s := "2023-06-15T16:24:11 +07:00"
		te, ts := ParseTimeStyle(s)
		assert.NotNil(t, te)
		assert.NotNil(t, ts)
		assert.Equal(t, int64(1686821051000), te.Parse(ts, s, time.Local))
		assert.Equal(t, ts.Layout, "2006-01-02T15:04:05 Z07:00")
	}
}
