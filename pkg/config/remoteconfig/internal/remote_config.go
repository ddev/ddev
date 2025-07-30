package internal

type Remote struct {
	Owner    string `json:"owner,omitempty"`
	Repo     string `json:"repo,omitempty"`
	Ref      string `json:"ref,omitempty"`
	Filepath string `json:"filepath,omitempty"`
}

type RemoteConfig struct {
	UpdateInterval int    `json:"update-interval,omitempty"`
	Remote         Remote `json:"remote,omitempty"`

	// Legacy structure for backward compatibility
	Messages Messages `json:"messages,omitempty"`

	// Direct ticker structure as it appears in the actual JSON
	Ticker Ticker `json:"ticker,omitempty"`
}
