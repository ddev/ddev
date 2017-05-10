package cmd

import (
	"log"
	"path/filepath"
	"testing"

	"github.com/drud/drud-go/utils/system"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/assert"
)

// TestImportTilde tests passing paths to import-files that use ~ to represent home dir.
func TestImportTilde(t *testing.T) {
	assert := assert.New(t)

	for _, site := range DevTestSites {
		cleanup := site.Chdir()

		usr, err := homedir.Dir()
		assert.NoError(err)
		err = system.DownloadFile(filepath.Join(usr, "files.tar.gz"), site.FileURL)
		assert.NoError(err)

		// this ~ should be expanded by shell
		args := []string{"import-files", "--src", "~/files.tar.gz"}
		out, err := system.RunCommand(DdevBin, args)
		if err != nil {
			log.Println("Error Output from ddev import-files:", out, site)
		}
		assert.NoError(err)
		assert.Contains(string(out), "Successfully imported files")

		// this ~ is not expanded by shell, ddev should convert it to a valid path
		args = []string{"import-files", "--src=~/files.tar.gz"}
		out, err = system.RunCommand(DdevBin, args)
		if err != nil {
			log.Println("Error Output from ddev import-files:", out, site)
		}
		assert.NoError(err)
		assert.Contains(string(out), "Successfully imported files")

		cleanup()
	}

	assert.NoError(nil)
}
