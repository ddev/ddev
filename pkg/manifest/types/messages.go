package types

type MessageType int64

const (
	Info MessageType = iota
	Warning
)

type Message struct {
	Message  string `json:"message,omitempty"`
	Versions string `json:"versions,omitempty"`
}

type Messages struct {
	Infos    []Message `json:"infos,omitempty"`
	Warnings []Message `json:"warnings,omitempty"`
}
