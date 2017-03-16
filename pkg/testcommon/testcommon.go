package testcommon

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/drud/drud-go/utils/system"
)

var archive = path.Join(os.TempDir(), "testsite.tar.gz")

// PrepareTest downloads and extracts a site codebase, and ensures the tests are run from the location the codebase is extracted to.
func PrepareTest(site []string) {
	siteName := site[0]
	archiveURL := site[1]
	archiveExtPath := site[2]

	testDir := path.Join(os.TempDir(), archiveExtPath)

	fmt.Printf("Prepping test for %s.", siteName)
	os.Setenv("DRUD_NONINTERACTIVE", "true")
	os.Mkdir(testDir, 0755)

	system.DownloadFile(archive, archiveURL)

	system.RunCommand("tar",
		[]string{
			"-xzf",
			archive,
			"-C",
			os.TempDir(),
		})

	err := os.Chdir(testDir)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("running from %s\n", testDir)
}

// CleanupTest removes the archive and codebase extraction for a site after a test run has completed.
func CleanupTest(site []string) {
	siteName := site[0]
	archiveExtPath := site[2]
	testDir := path.Join(os.TempDir(), archiveExtPath)
	fmt.Printf("Cleaning up from %s tests.", siteName)
	os.Remove(archive)
	os.RemoveAll(testDir)
}
