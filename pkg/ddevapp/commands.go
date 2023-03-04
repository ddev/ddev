package ddevapp

import (
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
	copy2 "github.com/otiai10/copy"
	"os"
	"path/filepath"
)

// PopulateCustomCommandFiles sets up the custom command files in the project
// directories where they need to go.
func PopulateCustomCommandFiles(app *DdevApp) error {

	sourceGlobalCommandPath := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands")
	err := os.MkdirAll(sourceGlobalCommandPath, 0755)
	if err != nil {
		return nil
	}

	projectCommandPath := app.GetConfigPath("commands")
	// Make sure our target global command directory is empty
	copiedGlobalCommandPath := app.GetConfigPath(".global_commands")
	err = os.MkdirAll(copiedGlobalCommandPath, 0755)
	if err != nil {
		util.Error("Unable to create directory %s: %v", copiedGlobalCommandPath, err)
		return nil
	}

	// Make sure it's empty
	err = fileutil.PurgeDirectory(copiedGlobalCommandPath)
	if err != nil {
		util.Error("Unable to remove %s: %v", copiedGlobalCommandPath, err)
		return nil
	}

	err = copy2.Copy(sourceGlobalCommandPath, copiedGlobalCommandPath)
	if err != nil {
		return err
	}

	if !fileutil.FileExists(projectCommandPath) || !fileutil.IsDirectory(projectCommandPath) {
		return nil
	}
	return nil
}
