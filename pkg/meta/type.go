package meta

type (
	Data struct {
	}

	Syncer interface {
		Start() error

		Stop() error

		AddEventHandler(EventHandler)
	}

	EventHandler interface {
		OnUpdate(oldObj, newObj []*Data)
	}
)
