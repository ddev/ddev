package testcommon

import (
	"bytes"
	log "github.com/Sirupsen/logrus"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/drud/drud-go/utils/system"
)

// TestSite describes a site for testing, with name, URL of tarball, and optional dir.
type TestSite struct {
	// Name is the generic name of the site, and is used as the default dir.
	Name string
	// DownloadURL is the URL of the tarball to be used for building the site.
	DownloadURL string
	Dir         string
}

func (site *TestSite) archivePath() string {
	dir := CreateTmpDir(site.Name + "download")
	return filepath.Join(dir, site.Name+".tar.gz")
}

// Prepare downloads and extracts a site codebase to a temporary directory.
func (site *TestSite) Prepare() error {
	testDir := CreateTmpDir(site.Name)
	site.Dir = testDir
	log.Debugf("Prepping test for %s.\n", site.Name)
	os.Setenv("DRUD_NONINTERACTIVE", "true")

	log.Debugln("Downloading file:", site.DownloadURL)
	tarballPath := site.archivePath()
	err := system.DownloadFile(tarballPath, site.DownloadURL)

	if err != nil {
		site.Cleanup()
		return err
	}
	log.Debugln("File downloaded:", tarballPath)

	_, err = system.RunCommand("tar",
		[]string{
			"-xzf",
			tarballPath,
			"--strip", "1",
			"-C",
			site.Dir,
		})
	if err != nil {
		log.Errorf("Tar extraction failed err=%v\n", err)
		// If we had an error extracting the archive, we should go ahead and clean up the temporary directory, since this
		// testsite is useless.
		site.Cleanup()
	}

	return err
}

// Chdir will change to the directory for the site specified by TestSite.
func (site *TestSite) Chdir() func() {
	return Chdir(site.Dir)
}

// Cleanup removes the archive and codebase extraction for a site after a test run has completed.
func (site *TestSite) Cleanup() {
	os.Remove(site.archivePath())
	CleanupDir(site.Dir)
}

// CleanupDir removes a directory specified by string.
func CleanupDir(dir string) error {
	err := os.RemoveAll(dir)
	return err
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

	return func() { os.Chdir(curDir) }
}

var letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// RandString returns a random string of given length n.
func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func init() {
	rand.Seed(time.Now().UnixNano())
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
			io.Copy(&buf, r)
			outC <- buf.String()
		}()

		// back to normal state
		w.Close()
		os.Stdout = old // restoring the real stdout
		out := <-outC
		return out
	}

}

func setLetterBytes(lb string) {
	letterBytes = lb
}
