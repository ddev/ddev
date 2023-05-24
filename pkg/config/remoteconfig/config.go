package remoteconfig

import (
	"path/filepath"

	"github.com/ddev/ddev/internal/build"
)

// LocalSource is the struct defining the local source.
type LocalSource struct {
	Path string
}

// RemoteSource is the struct defining the remote source.
type RemoteSource struct {
	Owner    string
	Repo     string
	Ref      string
	Filepath string
}

// Config is the struct defining the RemoteConfig config.
type Config struct {
	LocalSource  LocalSource
	RemoteSource RemoteSource

	UpdateInterval int
	TickerDisabled bool
	TickerInterval int
}

// getLocalSourceFileName returns the filename of the local storage.
func (c *Config) getLocalSourceFileName() string {
	return filepath.Join(c.LocalSource.Path, localFileName)
}

// getRemoteSourceOwner returns the owner to be used for the remote
// config download from Github, the global config overwrites the default.
func (c *Config) getRemoteSourceOwner() string {
	if c.RemoteSource.Owner != "" {
		return c.RemoteSource.Owner
	}

	return build.RemoteConfigRemoteSourceOwner
}

// getRemoteSourceRepo returns the repo to be used for the remote
// config download from Github, the global config overwrites the default.
func (c *Config) getRemoteSourceRepo() string {
	if c.RemoteSource.Repo != "" {
		return c.RemoteSource.Repo
	}

	return build.RemoteConfigRemoteSourceRepo
}

// getRemoteSourceRef returns the ref to be used for the remote
// config download from Github, the global config overwrites the default.
func (c *Config) getRemoteSourceRef() string {
	if c.RemoteSource.Ref != "" {
		return c.RemoteSource.Ref
	}

	return build.RemoteConfigRemoteSourceRef
}

// getRemoteSourceFilepath returns the filepath to be used for the remote
// config download from Github, the global config overwrites the default.
func (c *Config) getRemoteSourceFilepath() string {
	if c.RemoteSource.Filepath != "" {
		return c.RemoteSource.Filepath
	}

	return build.RemoteConfigRemoteSourceFilepath
}
