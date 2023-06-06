/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package transfer

type (
	// StateStore stores states for transformation.
	// States must use simple data structures, and expose field names, to ensure they can be successfully serialized and deserialized by gob.
	StateStore interface {
		// Get returns state associated by key
		Get(key string) (interface{}, error)
		// Put associates state with a key
		Put(key string, state interface{})
		// Sub returns a sub StateStore with key prefixed with the given parameter
		Sub(prefix string) StateStore
	}

	StatefulComponent interface {
		StopAndSaveState(StateStore) error
		LoadState(StateStore) error
	}

	// StatefulInput api.Input that need to be lossless when restart or redeploy must impl this interface.
	// TODO Supports states with different version are incompatible
	StatefulInput interface {
		SaveState() (interface{}, error)
		LoadState(interface{}) error
	}
)
