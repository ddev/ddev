package globalconfig

const (
	// DefaultRemoteConfigURL is the default URL for remote config data
	DefaultRemoteConfigURL = "https://raw.githubusercontent.com/ddev/remote-config/main/remote-config.jsonc"
	// DefaultSponsorshipDataURL is the default URL for sponsorship data
	DefaultSponsorshipDataURL = "https://ddev.com/s/sponsorship-data.json"
)

// RemoteConfig is the struct defining the remote-config config.
type RemoteConfig struct {
	UpdateInterval     int    `yaml:"update_interval,omitempty"`
	RemoteConfigURL    string `yaml:"remote_config_url,omitempty"`
	SponsorshipDataURL string `yaml:"sponsorship_data_url,omitempty"`
}
