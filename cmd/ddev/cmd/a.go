package cmd

import (
	"os"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
)

// This file is a.go because global config must be loaded before anybody else
// runs their init(), as they might overwrite global_config.yaml with
// uninitialized data

func init() {
	globalconfig.EnsureGlobalConfig()
	_ = os.Setenv("DOCKER_CLI_HINTS", "false")
	// GetMutagenDataDirectory() sets MUTAGEN_DATA_DIRECTORY
	_ = globalconfig.GetMutagenDataDirectory()
	// GetDockerClient should be called early to get DOCKER_HOST set
	_, _ = dockerutil.GetDockerClient()
}
