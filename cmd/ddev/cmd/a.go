package cmd

import (
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"os"
)

// This file is a.go because global config must be loaded before anybody else
// runs their init(), as they might overwrite global_config.yaml with
// uninitialized data

func init() {
	// Prevent running as root for most cases
	// We really don't want ~/.ddev to have root ownership, breaks things.
	if os.Geteuid() == 0 && len(os.Args) > 1 && os.Args[1] != "hostname" {
		output.UserOut.Fatal("ddev is not designed to be run with root privileges, please run as normal user and without sudo")
	}

	err := globalconfig.ReadGlobalConfig()
	if err != nil {
		util.Failed("unable to read global config: %v", err)
	}
	globalconfig.GetCAROOT()
}
