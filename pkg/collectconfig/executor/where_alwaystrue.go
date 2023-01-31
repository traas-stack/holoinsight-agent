package executor

type (
	xWhereAlwaysTrue struct{}
)

func (x xWhereAlwaysTrue) Test(ctx *LogContext) (ret bool, _ error) {
	we := ctx.whereEvent
	if we != nil {
		we.Name = "alwaysTrue"
		defer func() {
			we.Result = ret
		}()
	}

	return true, nil
}
