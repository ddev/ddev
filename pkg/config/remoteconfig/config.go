package remoteconfig

import (
	"path/filepath"
)

// Local is the struct defining the local source.
type Local struct {
	Path string
}

// Config is the struct defining the RemoteConfig config.
type Config struct {
	Local Local
	URL   string

	UpdateInterval int
	TickerInterval int
}

// getLocalSourceFileName returns the filename of the local storage.
func (c *Config) getLocalSourceFileName() string {
	return filepath.Join(c.Local.Path, localFileName)
}
