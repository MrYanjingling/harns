package message

type msgType byte

const (
	msgTypeUndefined msgType = iota
	msgTypeSimple
	msgTypeTemplate
)
