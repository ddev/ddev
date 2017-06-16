package testcommon

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/ddev/pkg/util"
	"github.com/pkg/errors"
)

// TestSite describes a site for testing, with name, URL of tarball, and optional dir.
type TestSite struct {
	// Name is the generic name of the site, and is used as the default dir.
	Name        string
	ArchivePath string
	// SourceURL is the URL of the source code tarball to be used for building the site.
	SourceURL string
	// ArchiveExtractionPath is the relative path within the tarball which should be extracted, ending with /
	ArchiveInternalExtractionPath string
	// FullSiteTarballURL is the URL of the tarball of a full site archive used for testing import.
	FullSiteTarballURL string
	// FilesTarballURL is the URL of the tarball of file uploads used for testing file import.
	FilesTarballURL string
	// FilesZipballURL is the URL of the zipball of file uploads used for testing file import.
	FilesZipballURL string
	// DBTarURL is the URL of the database dump tarball used for testing database import.
	DBTarURL string
	// DBZipURL is the URL of an optional zip-style db dump.
	DBZipURL string
	// Dir is the rooted full path of the test site
	Dir string
}

func (site *TestSite) createArchivePath() string {
	dir := CreateTmpDir(site.Name + "download")
	return filepath.Join(dir, site.Name+".tar.gz")
}

// Prepare downloads and extracts a site codebase to a temporary directory.
func (site *TestSite) Prepare() error {
	testDir := CreateTmpDir(site.Name)
	site.Dir = testDir
	log.Debugf("Prepping test for %s.\n", site.Name)
	err := os.Setenv("DRUD_NONINTERACTIVE", "true")
	util.CheckErr(err)

	log.Debugln("Downloading file:", site.SourceURL)
	site.ArchivePath = site.createArchivePath()
	err = util.DownloadFile(site.ArchivePath, site.SourceURL)

	if err != nil {
		site.Cleanup()
		return err
	}
	log.Debugln("File downloaded:", site.ArchivePath)

	err = archive.Untar(site.ArchivePath, site.Dir, site.ArchiveInternalExtractionPath)
	if err != nil {
		log.Errorf("Tar extraction failed err=%v\n", err)
		// If we had an error extracting the archive, we should go ahead and clean up the temporary directory, since this
		// testsite is useless.
		site.Cleanup()
	}

	// If our test site has a Name: then update the config file to reflect that.
	if site.Name != "" {
		config, err := ddevapp.NewConfig(site.Dir)
		if err != nil {
			return errors.Errorf("Failed to read site config for site %s, dir %s, err:%v", site.Name, site.Dir, err)
		}
		err = config.Read()
		if err != nil {
			return errors.Errorf("Failed to read site config for site %s, dir %s, err: %v", site.Name, site.Dir, err)
		}
		config.Name = site.Name
		err = config.Write()
		if err != nil {
			return errors.Errorf("Failed to write site config for site %s, dir %s, err: %v", site.Name, site.Dir, err)
		}
	}

	return err
}

// Chdir will change to the directory for the site specified by TestSite.
func (site *TestSite) Chdir() func() {
	return Chdir(site.Dir)
}

// Cleanup removes the archive and codebase extraction for a site after a test run has completed.
func (site *TestSite) Cleanup() {
	app, err := platform.GetActiveApp("")
	util.CheckErr(err)
	err = app.Down()
	util.CheckErr(err)
	err = os.Remove(site.ArchivePath)
	util.CheckErr(err)
	// CleanupDir checks its own errors.
	CleanupDir(site.Dir)
}

// CleanupDir removes a directory specified by string.
func CleanupDir(dir string) {
	err := os.RemoveAll(dir)
	util.CheckErr(err)
}

// OsTempDir gets os.TempDir() (usually provided by $TMPDIR) but expands any symlinks found within it.
// This wrapper function can prevent problems with docker-for-mac trying to use /var/..., which is not typically
// shared/mounted. It will be expanded via the /var symlink to /private/var/...
func OsTempDir() (string, error) {
	dirName := os.TempDir()
	tmpDir, err := filepath.EvalSymlinks(dirName)
	if err != nil {
		return "", err
	}
	tmpDir = filepath.Clean(tmpDir)
	return tmpDir, nil
}

// CreateTmpDir creates a temporary directory and returns its path as a string.
func CreateTmpDir(prefix string) string {
	systemTempDir, err := OsTempDir()
	if err != nil {
		log.Fatalln("Failed getting system temp dir", err)
	}
	fullPath, err := ioutil.TempDir(systemTempDir, prefix)
	if err != nil {
		log.Fatalln("Failed to create temp directory", err)
	}
	return fullPath
}

// Chdir will change to the directory for the site specified by TestSite.
// It returns an anonymous function which will return to the original working directory when called.
func Chdir(path string) func() {
	curDir, _ := os.Getwd()
	err := os.Chdir(path)
	if err != nil {
		log.Fatalf("Could not change to directory %s: %v\n", path, err)
	}

	return func() {
		err := os.Chdir(curDir)
		if err != nil {
			log.Fatalf("Failed to change directory to original dir=%s, err=%v", curDir, err)
		}
	}
}

// CaptureStdOut captures Stdout to a string. Capturing starts when it is called. It returns an anonymous function that when called, will return a string
// containing the output during capture, and revert once again to the original value of os.StdOut.
func CaptureStdOut() func() string {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	return func() string {
		outC := make(chan string)
		// copy the output in a separate goroutine so printing can't block indefinitely
		go func() {
			var buf bytes.Buffer
			_, err := io.Copy(&buf, r)
			util.CheckErr(err)
			outC <- buf.String()
		}()

		// back to normal state
		util.CheckClose(w)
		os.Stdout = old // restoring the real stdout
		out := <-outC
		return out
	}

}

// ClearDockerEnv unsets env vars set in platform DockerEnv() so that
// they can be set by another test run.
func ClearDockerEnv() {
	envVars := []string{
		"COMPOSE_PROJECT_NAME",
		"DDEV_SITENAME",
		"DDEV_DBIMAGE",
		"DDEV_WEBIMAGE",
		"DDEV_APPROOT",
		"DDEV_DOCROOT",
		"DDEV_URL",
		"DDEV_HOSTNAME",
	}
	for _, env := range envVars {
		err := os.Unsetenv(env)
		if err != nil {
			log.Printf("failed to unset %s: %v\n", env, err)
		}
	}
}

// ContainerCheck determines if a given container name exists and matches a given state
func ContainerCheck(checkName string, checkState string) (bool, error) {
	// ensure we have docker network
	client := dockerutil.GetDockerClient()
	err := dockerutil.EnsureNetwork(client, dockerutil.NetName)
	if err != nil {
		log.Fatal(err)
	}

	containers, err := dockerutil.GetDockerContainers(true)
	if err != nil {
		log.Fatal(err)
	}

	for _, container := range containers {
		name := dockerutil.ContainerName(container)
		if name == checkName {
			if container.State == checkState {
				return true, nil
			}
			return false, errors.New("container " + name + " returned " + container.State)
		}
	}

	return false, errors.New("unable to find container " + checkName)
}

// TimeTrack determines the amount of time a function takes to return. Timing starts when it is called.
// It returns an anonymous function that, when called, will print the elapsed run time.
func TimeTrack(start time.Time, name string) func() {
	return func() {
		elapsed := time.Since(start)
		log.Printf("PERF: %s took %s", name, elapsed)
	}
}
