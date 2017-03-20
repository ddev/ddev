package testcommon

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/drud/drud-go/utils/system"
)

type TestSite struct {
	Name string
	URL  string
	Path string
}

var archive = path.Join(os.TempDir(), "testsite.tar.gz")

// PrepareTest downloads and extracts a site codebase, and ensures the tests are run from the location the codebase is extracted to.
func PrepareTest(site *TestSite) {

	testDir, err := ioutil.TempDir("", "example")
	if err != nil {
		log.Fatalf("Could not create temporary directory %s for site %s", testDir, site.Name)
	}
	site.Path = testDir
	fmt.Printf("Prepping test for %s.", site.Name)
	os.Setenv("DRUD_NONINTERACTIVE", "true")

	system.DownloadFile(archive, site.URL)
	system.RunCommand("tar",
		[]string{
			"-xzf",
			archive,
			"--strip", "1",
			"-C",
			site.Path,
		})
}

func Chdir(dir string) func() {
	curDir, _ := os.Getwd()
	err := os.Chdir(dir)
	if err != nil {
		log.Fatalf("Could not change to directory %s: %v\n", dir, err)
	}

	return func() { os.Chdir(curDir) }
}

// CleanupTest removes the archive and codebase extraction for a site after a test run has completed.
func CleanupTest(site TestSite) {
	os.Remove(archive)
	os.RemoveAll(site.Path)
}
