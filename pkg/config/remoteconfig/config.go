package remoteconfig

import (
	"path/filepath"

	"github.com/ddev/ddev/pkg/build"
	"github.com/ddev/ddev/pkg/config/remoteconfig/internal"
)

// Local is the struct defining the local source.
type Local struct {
	Path string
}

// Remote is the struct defining the remote source.
type Remote struct {
	Owner    string
	Repo     string
	Ref      string
	Filepath string
}

// Config is the struct defining the RemoteConfig config.
type Config struct {
	Local  Local
	Remote Remote

	UpdateInterval int
	TickerInterval int
}

// getLocalSourceFileName returns the filename of the local storage.
func (c *Config) getLocalSourceFileName() string {
	return filepath.Join(c.Local.Path, localFileName)
}

// getRemoteSourceOwner returns the owner to be used for the remote
// config download from Github, the global config overwrites the default.
func (c *Config) getRemoteSourceOwner(remoteConfig *internal.RemoteConfig) string {
	if c.Remote.Owner != "" {
		return c.Remote.Owner
	}

	if remoteConfig.Remote.Owner != "" {
		return remoteConfig.Remote.Owner
	}

	return build.RemoteConfigRemoteSourceOwner
}

// getRemoteSourceRepo returns the repo to be used for the remote
// config download from Github, the global config overwrites the default.
func (c *Config) getRemoteSourceRepo(remoteConfig *internal.RemoteConfig) string {
	if c.Remote.Repo != "" {
		return c.Remote.Repo
	}

	if remoteConfig.Remote.Repo != "" {
		return remoteConfig.Remote.Repo
	}

	return build.RemoteConfigRemoteSourceRepo
}

// getRemoteSourceRef returns the ref to be used for the remote
// config download from Github, the global config overwrites the default.
func (c *Config) getRemoteSourceRef(remoteConfig *internal.RemoteConfig) string {
	if c.Remote.Ref != "" {
		return c.Remote.Ref
	}

	if remoteConfig.Remote.Ref != "" {
		return remoteConfig.Remote.Ref
	}

	return build.RemoteConfigRemoteSourceRef
}

// getRemoteSourceFilepath returns the filepath to be used for the remote
// config download from Github, the global config overwrites the default.
func (c *Config) getRemoteSourceFilepath(remoteConfig *internal.RemoteConfig) string {
	if c.Remote.Filepath != "" {
		return c.Remote.Filepath
	}

	if remoteConfig.Remote.Filepath != "" {
		return remoteConfig.Remote.Filepath
	}

	return build.RemoteConfigRemoteSourceFilepath
}
