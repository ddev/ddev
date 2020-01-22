package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/util"
)

func init() {
	err := globalconfig.ReadGlobalConfig()
	if err != nil {
		util.Failed("unable to read global config: %v")
	}
	ddevapp.GetCAROOT()
}
