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
		assert.Equal(t, int64(1609527845000), te.Parse(ts, s, time.Local))
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
}
