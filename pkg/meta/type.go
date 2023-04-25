/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

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
