package logger

import "testing"

func TestLog(t *testing.T) {
	TestMode()
	Infof("AAAA")
	Errorf("BBBB")
}
