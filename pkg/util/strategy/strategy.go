package strategy

type RetryStrategy interface {
	GetNext() int
	Reset()
}
