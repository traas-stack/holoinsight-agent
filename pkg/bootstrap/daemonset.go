/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package bootstrap

type (
	Runnable func()
)

var daemonsetHooks []Runnable

func RegisterDaemonsetHook(r Runnable) {
	daemonsetHooks = append(daemonsetHooks, r)
}

func TriggerDaemonsetHooks() {
	for _, r := range daemonsetHooks {
		r()
	}
}
