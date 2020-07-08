package main

import (
	"os"
	"path/filepath"

	"github.com/drud/ddev/cmd/ddev/cmd"
	"github.com/drud/ddev/pkg/util"
)

var targetDir = ".gotmp/bin"

func main() {
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		err = os.MkdirAll(targetDir, 0755)
		util.CheckErr(err)
	}
	err := cmd.RootCmd.GenBashCompletionFile(filepath.Join(targetDir, "ddev_bash_completion.sh"))
	if err != nil {
		util.Failed("could not generate ddev_bash_completion.sh: %v", err)
	}
	err = cmd.RootCmd.GenZshCompletionFile(filepath.Join(targetDir, "ddev_zsh_completion.sh"))
	if err != nil {
		util.Failed("could not generate ddev_zsh_completion.sh: %v", err)
	}
	err = cmd.RootCmd.GenFishCompletionFile(filepath.Join(targetDir, "ddev_fish_completion.sh"), true)
	if err != nil {
		util.Failed("could not generate ddev_fish_completion.sh: %v", err)
	}
	err = cmd.RootCmd.GenPowerShellCompletionFile(filepath.Join(targetDir, "ddev_powershell_completion.ps1"))
	if err != nil {
		util.Failed("could not generate ddev_powershell_completion.ps1: %v", err)
	}
}
