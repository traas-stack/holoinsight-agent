package executor

import (
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"reflect"
)

type (
	xSwitchCaseV1Filter struct {
		conf          *collectconfig.TransformFilterSwitchCaseV1
		cases         []*xCase
		defaultAction XTransformFilter
	}
	xCase struct {
		case_  XWhere
		action XTransformFilter
	}
)

func (x *xSwitchCaseV1Filter) Init() error {
	x.cases = make([]*xCase, 0, len(x.conf.Cases))

	for _, case_ := range x.conf.Cases {
		fillDefaultElect(case_.Case)
		where, err := parseWhere(case_.Case)
		if err != nil {
			return err
		}
		action, err := parseTransformFilter(case_.Action)
		if err != nil {
			return err
		}
		x.cases = append(x.cases, &xCase{
			case_:  where,
			action: action,
		})
	}

	if x.conf.DefaultAction != nil {
		action, err := parseTransformFilter(x.conf.DefaultAction)
		if err != nil {
			return err
		}
		x.defaultAction = action
	}
	return nil
}

var whereType = reflect.TypeOf(collectconfig.Where{})

func fillDefaultElect(where *collectconfig.Where) {
	if where == nil {
		return
	}
	for _, sub := range where.And {
		fillDefaultElect(sub)
	}
	for _, sub := range where.Or {
		fillDefaultElect(sub)
	}
	if where.Not != nil {
		fillDefaultElect(where.Not)
	}

	elem := reflect.ValueOf(where).Elem()
	for i := 0; i < whereType.NumField(); i++ {
		fieldType := whereType.Field(i)
		switch fieldType.Name {
		case "And":
		case "Or":
		case "Not":
		default:
			fieldValue := elem.Field(i)
			if fieldValue.Kind() != reflect.Ptr || fieldValue.IsNil() {
				continue
			}
			electField := fieldValue.Elem().FieldByName("Elect")
			if electField.IsNil() {
				electField.Set(reflect.ValueOf(collectconfig.CElectContext))
			}

		}
	}
}

func (x *xSwitchCaseV1Filter) Filter(ctx *LogContext) (interface{}, error) {
	for _, case_ := range x.cases {
		b, err := case_.case_.Test(ctx)
		if err != nil {
			return nil, err
		}
		if b {
			return case_.action.Filter(ctx)
		}
	}

	if x.defaultAction != nil {
		return x.defaultAction.Filter(ctx)
	}

	// no case matched, and no defaultAction
	// returns last value
	return ctx.contextValue, nil
}
