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
