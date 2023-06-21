package globalconfig

// RemoteConfigRemote is the struct defining the source of the remote-config.
type RemoteConfigRemote struct {
	Owner    string `yaml:"owner,omitempty"`
	Repo     string `yaml:"repo,omitempty"`
	Ref      string `yaml:"ref,omitempty"`
	Filepath string `yaml:"filepath,omitempty"`
}

// RemoteConfig is the struct defining the remote-config config.
type RemoteConfig struct {
	UpdateInterval int                `yaml:"update_interval,omitempty"`
	Remote         RemoteConfigRemote `yaml:"remote,omitempty"`
}
