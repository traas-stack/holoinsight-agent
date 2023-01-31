package model

type (
	Handler func(action string, resp *Resp) error
)
