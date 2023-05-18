package internal

type RemoteConfig struct {
	UpdateInterval int `json:"update-interval,omitempty"`

	Messages Messages `json:"messages,omitempty"`
}
