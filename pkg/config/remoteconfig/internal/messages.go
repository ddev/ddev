package internal

type Message struct {
	Message    string   `json:"message"`
	Title      string   `json:"title,omitempty"`
	Conditions []string `json:"conditions,omitempty"`
	Versions   string   `json:"versions,omitempty"`
}

type Notifications struct {
	Interval int       `json:"interval"`
	Infos    []Message `json:"infos"`
	Warnings []Message `json:"warnings"`
}

type Ticker struct {
	Interval int       `json:"interval"`
	Messages []Message `json:"messages"`
}

type Messages struct {
	Notifications Notifications `json:"notifications"`
	Ticker        Ticker        `json:"ticker"`
}
