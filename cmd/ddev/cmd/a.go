package cmd

import (
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/versionconstants"
	"os"
)

// This file is a.go because global config must be loaded before anybody else
// runs their init(), as they might overwrite global_config.yaml with
// uninitialized data

func init() {
	err := globalconfig.ReadGlobalConfig()
	if err != nil {
		util.Failed("unable to read global config: %v", err)
	}
	globalconfig.GetCAROOT()
	_ = os.Setenv("MUTAGEN_DATA_DIRECTORY", globalconfig.GetMutagenDataDirectory())
	// GetDockerClient should be called early to get DOCKER_HOST set
	_ = dockerutil.GetDockerClient()

	// This should be in versionconstants, but it's here because initialization
	// of this type is consolidated here.
	if globalconfig.DdevGlobalConfig.UseTraefik {
		versionconstants.RouterImage = "traefik:v2.8"
	}

}
