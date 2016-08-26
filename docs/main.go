package main

import (
	drudcmd "github.com/drud/bootstrap/cli/cmd"

	"github.com/spf13/cobra/doc"
)

func main() {
	cmd := drudcmd.RootCmd
	doc.GenMarkdownTree(cmd, "./")
}
