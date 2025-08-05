package globalconfig

// RemoteConfig is the struct defining the remote-config config.
type RemoteConfig struct {
	UpdateInterval     int    `yaml:"update_interval,omitempty"`
	RemoteConfigURL    string `yaml:"remote_config_url,omitempty"`
	SponsorshipDataURL string `yaml:"sponsorship_data_url,omitempty"`
}
