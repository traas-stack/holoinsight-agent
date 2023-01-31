package executor

type (
	LogInput interface {
		Start()
		Stop()
		Pull(*PullRequest) (*PullResponse, error)
		Finished() bool
	}
)
