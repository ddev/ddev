package ddevapp

import (
	"os"

	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"
)

// genericPostConfigAction is the default post-config, and will currently
// remove an existing nginx-site.conf
func genericPostConfigAction(app *DdevApp) error {
	// If an nginx-site.conf exists that we generated, it should be removed by
	// default
	filePath := app.GetConfigPath("nginx-site.conf")

	if fileutil.FileExists(filePath) {
		signatureFound, err := fileutil.FgrepStringInFile(filePath, DdevFileSignature)
		util.CheckErr(err) // Really can't happen as we already checked for the file existence

		if signatureFound {
			util.Warning("Removing ddev-generated %s", filePath)
			err = os.Remove(filePath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
