package executor

import (
	"strings"
	"testing"
)

func TestSplit(t *testing.T) {
	str := "2022-08-29 16:37:34,747|||0a374dd916617622522722575e006d|INFO |DubboServerHandler-10.55.77.217:20880-thread-197|DUBBO-MONITOR-PROVIDER-DIGEST-LOG|superlink|PROVIDER|-|gloandecision|10.55.78.187:38588 -> 10.55.77.217:20880|com.antbank.superlink.facade.MessageSendService|send|Y|2475|-|{\"com.antbank.fc.common.lang.event.EventContext\":\"{\\\"tntInstId\\\":\\\"XHBKC1MO\\\"}\"}"
	println(len(strings.Split(str, "|")))
}
