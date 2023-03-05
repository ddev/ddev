package main

import (
	"github.com/ddev/ddev/cmd/ddev/cmd"
	"github.com/ddev/ddev/pkg/util"
	"os"
	"path/filepath"
)

var targetDir = ".gotmp/bin/completions"

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
	err = genFigSpecCompletionFile(filepath.Join(targetDir, "ddev_fig_spec.ts"))
	if err != nil {
		util.Failed("could not generate ddev_fig_spec.ts: %v", err)
	}
}
