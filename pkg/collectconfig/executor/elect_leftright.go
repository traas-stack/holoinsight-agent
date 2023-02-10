package executor

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/spf13/cast"
	"strings"
)

type (
	xLeftRight struct {
		rule *collectconfig.LeftRight
	}
)

func (x *xLeftRight) Elect(rc *LogContext) (interface{}, error) {
	return x.ElectString(rc)
}

func (x *xLeftRight) ElectString(rc *LogContext) (string, error) {
	// TODO 需要有一个左起右至的cache来加速
	// 缺点, 这加速用的cache可能要挂在rc上, 这样rc就感知了实现细节...
	// 可以想办法让rc可以挂任意属性, 然后每个模块自己强转, 不知道性能如何

	s := rc.GetLine()

	for i := 0; i <= x.rule.LeftIndex; i++ {
		r := strings.Index(s, x.rule.Left)
		if r < 0 {
			//if x.rule.DefaultValue != nil {
			//	return *x.rule.DefaultValue, nil
			//}
			// return "", errors.New("no left")
			return "-", nil
		}
		s = s[r+len(x.rule.Left):]
	}
	var r int
	if x.rule.Right == "" {
		// 匹配到完
		r = len(s)
	} else {
		r = strings.Index(s, x.rule.Right)
	}
	if r < 0 {
		r = len(s)
	}
	// 没有找到right
	// 默认匹配到行尾
	if r == 0 {
		return "-", nil
	}
	return s[:r], nil
}

func (x *xLeftRight) ElectNumber(ctx *LogContext) (float64, error) {
	s, err := x.ElectString(ctx)
	if err != nil {
		return 0, err
	}
	return cast.ToFloat64E(s)
}

func (x *xLeftRight) Init() {
}

func parseShortLeftRightRule(s []interface{}) (*collectconfig.LeftRight, error) {
	if len(s) != 3 {
		return nil, errors.New("LeftRightS")
	}
	leftIndex, err := cast.ToIntE(s[0])
	if err != nil {
		return nil, err
	}
	left, err := cast.ToStringE(s[1])
	if err != nil {
		return nil, err
	}
	right, err := cast.ToStringE(s[2])
	if err != nil {
		return nil, err
	}
	return &collectconfig.LeftRight{
		LeftIndex: leftIndex,
		Left:      left,
		Right:     right,
	}, nil
}
