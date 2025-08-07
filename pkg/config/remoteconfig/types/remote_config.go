package types

// RemoteConfig interface for displaying remote config data
type RemoteConfig interface {
	ShowNotifications()
	ShowTicker()
	ShowSponsorshipAppreciation()
}

// Remote config data structures (moved from internal package)

// Message represents a single message
type Message struct {
	Message    string   `json:"message"`
	Title      string   `json:"title,omitempty"`
	Conditions []string `json:"conditions,omitempty"`
	Versions   string   `json:"versions,omitempty"`
}

// Notifications represents notification messages
type Notifications struct {
	Interval int       `json:"interval"`
	Infos    []Message `json:"infos"`
	Warnings []Message `json:"warnings"`
}

// Ticker represents ticker messages
type Ticker struct {
	Interval int       `json:"interval"`
	Messages []Message `json:"messages"`
}

// Messages represents the messages configuration
type Messages struct {
	Notifications Notifications `json:"notifications"`
	Ticker        Ticker        `json:"ticker"`
}

// Remote represents the remote source configuration
type Remote struct {
	Owner    string `json:"owner,omitempty"`
	Repo     string `json:"repo,omitempty"`
	Ref      string `json:"ref,omitempty"`
	Filepath string `json:"filepath,omitempty"`
}

// RemoteConfigData represents the remote config structure
type RemoteConfigData struct {
	UpdateInterval int      `json:"update-interval,omitempty"`
	Remote         Remote   `json:"remote,omitempty"`
	Messages       Messages `json:"messages,omitempty"`
}
