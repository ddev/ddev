package internal

type Message struct {
	Message  string `json:"message,omitempty"`
	Versions string `json:"versions,omitempty"`
}

type Tips struct {
	Messages []string `json:"messages,omitempty"`
	Last     int      `json:"last,omitempty"`
}

type Messages struct {
	Infos    []Message `json:"infos,omitempty"`
	Warnings []Message `json:"warnings,omitempty"`
	Tips     Tips      `json:"tips,omitempty"`
}
