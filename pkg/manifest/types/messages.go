package types

type MessageType int64

const (
	Info    MessageType = iota
	Warning MessageType = iota
)
