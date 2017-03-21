package testcommon

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/drud/drud-go/utils/system"
)

type TestSite struct {
	Name string
	URL  string
	Dir  string
}

func (site *TestSite) archivePath() string {
	return filepath.Join(os.TempDir(), site.Name+".tar.gz")
}

// Prepare downloads and extracts a site codebase to a temporary directory.
func (site *TestSite) Prepare() {

	testDir, err := ioutil.TempDir("", "example")
	if err != nil {
		log.Fatalf("Could not create temporary directory %s for site %s", testDir, site.Name)
	}
	site.Dir = testDir
	fmt.Printf("Prepping test for %s.", site.Name)
	os.Setenv("DRUD_NONINTERACTIVE", "true")

	system.DownloadFile(site.archivePath(), site.URL)
	system.RunCommand("tar",
		[]string{
			"-xzf",
			site.archivePath(),
			"--strip", "1",
			"-C",
			site.Dir,
		})
}

// Chdir will change to the directory for the site specified by TestSite.
// It returns an anonymous function which will return to the original working directory when called.
func (site *TestSite) Chdir() func() {
	curDir, _ := os.Getwd()
	err := os.Chdir(site.Dir)
	if err != nil {
		log.Fatalf("Could not change to directory %s: %v\n", site.Dir, err)
	}

	return func() { os.Chdir(curDir) }
}

// Cleanup removes the archive and codebase extraction for a site after a test run has completed.
func (site *TestSite) Cleanup() {
	os.Remove(site.archivePath())
	os.RemoveAll(site.Dir)
}
