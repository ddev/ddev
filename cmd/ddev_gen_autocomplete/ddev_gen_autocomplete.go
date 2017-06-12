package main

import (
	"github.com/drud/ddev/cmd/ddev/cmd"
	"github.com/drud/ddev/pkg/util"
	"os"
	"path/filepath"
)

var targetDir string = "bin"

func main() {
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		err = os.MkdirAll(targetDir, 0755)
		util.CheckErr(err)
	}
	err := cmd.RootCmd.GenBashCompletionFile(filepath.Join(targetDir, "ddev_bash_completion.sh"))
	util.CheckErr(err)
}
