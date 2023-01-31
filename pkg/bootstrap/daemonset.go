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
