package cmd

import (
	"os"

	"github.com/ddev/ddev/pkg/globalconfig"
)

// setupGlobalConfig loads global config; it must run before any other
// setup that might read or overwrite global_config.yaml with uninitialized
// data, so Setup() calls it first.
func setupGlobalConfig() {
	globalconfig.EnsureGlobalConfig()
	_ = os.Setenv("DOCKER_CLI_HINTS", "false")
	// GetMutagenDataDirectory() sets MUTAGEN_DATA_DIRECTORY
	_ = globalconfig.GetMutagenDataDirectory()
}
