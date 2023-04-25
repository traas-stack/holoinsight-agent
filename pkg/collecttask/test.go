/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package collecttask

type (
	One struct {
		One *CollectTask
	}
)

func (f *One) GetAll() []*CollectTask {
	return []*CollectTask{f.One}
}

func (f *One) Listen(listener ChangeListener) {
}

func (f *One) RemoveListen(listener ChangeListener) {
}
