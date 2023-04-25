/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

type (
	multilineAccumulator struct {
		pendingLog *LogGroup
		multiline  *xMultiline
	}
)

func newMultilineAccumulator(multiline *xMultiline) *multilineAccumulator {
	return &multilineAccumulator{
		multiline: multiline,
	}
}

func (a *multilineAccumulator) add(ctx *LogContext) (ret *LogGroup, err error) {
	b, err := a.multiline.where.Test(ctx)
	if err != nil {
		return nil, err
	}

	if b {
		if a.multiline.what == multilineWhatPrevious {
			// 该行匹配 where, 纳入 previous 组

			// 当前组必须要存在
			if a.pendingLog != nil {
				a.pendingLog.Add(ctx.GetLine())
				if len(a.pendingLog.Lines) >= a.multiline.maxLines {
					// force commit
					ret = a.pendingLog
					a.pendingLog = nil
				}
			} else {
				// warn?
			}

		} else {
			// 该行匹配 where, 纳入当前组
			if a.pendingLog == nil {
				a.pendingLog = &LogGroup{Line: ctx.GetLine(), Lines: []string{ctx.GetLine()}}
			} else {
				a.pendingLog.Add(ctx.GetLine())
				if len(a.pendingLog.Lines) >= a.multiline.maxLines {
					// force commit
					ret = a.pendingLog
					a.pendingLog = nil
				}
			}
		}
	} else {
		if a.multiline.what == multilineWhatPrevious {
			// 该行不匹配 where, 因此它中断 pendingLog, 并且自立一个新分组, 它成为行首
			ret = a.pendingLog
			a.pendingLog = &LogGroup{Line: ctx.GetLine(), Lines: []string{ctx.GetLine()}}
		} else {
			// 该行不匹配 where, 因此它中断 pendingLog, 它是pending的最后一行
			a.pendingLog.Add(ctx.GetLine())
			ret = a.pendingLog
			a.pendingLog = nil
		}
	}
	return
}

func (a *multilineAccumulator) getAndClearPending() *LogGroup {
	pending := a.pendingLog
	a.pendingLog = nil
	return pending
}
