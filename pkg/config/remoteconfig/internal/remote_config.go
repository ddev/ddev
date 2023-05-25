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

	Messages Messages `json:"messages,omitempty"`
}
