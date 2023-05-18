package internal

type Message struct {
	Message    string   `json:"message,omitempty"`
	Conditions []string `json:"conditions,omitempty"`
	Versions   string   `json:"versions,omitempty"`
}

type Ticker struct {
	Disabled bool      `json:"disabled,omitempty"`
	Interval int       `json:"interval,omitempty"`
	Last     int       `json:"last,omitempty"`
	Messages []Message `json:"messages,omitempty"`
}

type Messages struct {
	Infos    []Message `json:"infos,omitempty"`
	Warnings []Message `json:"warnings,omitempty"`
	Ticker   Ticker    `json:"ticker,omitempty"`
}
