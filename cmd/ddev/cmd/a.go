package cmd

import (
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/util"
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
	// GetDockerClient should be called early to get DOCKER_HOST set
	_ = dockerutil.GetDockerClient()
}
