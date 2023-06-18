package main

import (
	"github.com/ddev/ddev/cmd/ddev/cmd"
	"github.com/withfig/autocomplete-tools/integrations/cobra"
	"os"
)

// Generate a Fig spec
func genFigSpecCompletionFile(filename string) error {
	var spec = cobracompletefig.GenerateCompletionSpec(cmd.RootCmd)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	_, err = f.WriteString(spec.ToTypescript())
	if err != nil {
		return err
	}
	err = f.Close()

	return err
}
