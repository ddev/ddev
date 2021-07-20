package testcommon

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

var DdevBin = "ddev"
var TestSites = []TestSite{
	{
		Name:                          "",
		SourceURL:                     "https://github.com/drud/wordpress/archive/v0.4.0.tar.gz",
		ArchiveInternalExtractionPath: "wordpress-0.4.0/",
		FilesTarballURL:               "https://github.com/drud/wordpress/releases/download/v0.4.0/files.tar.gz",
		DBTarURL:                      "https://github.com/drud/wordpress/releases/download/v0.4.0/db.tar.gz",
		Docroot:                       "htdocs",
		Type:                          nodeps.AppTypeWordPress,
		Safe200URIWithExpectation:     URIWithExpect{URI: "/readme.html", Expect: "Welcome. WordPress is a very special project to me."},
	},
}

// TestCreateTmpDir tests the ability to create a temporary directory.
func TestCreateTmpDir(t *testing.T) {
	assert := asrt.New(t)

	// Create a temporary directory and ensure it exists.
	testDir := CreateTmpDir("TestCreateTmpDir")
	dirStat, err := os.Stat(testDir)
	assert.NoError(err, "There is no error when getting directory details")
	assert.True(dirStat.IsDir(), "Temp Directory created and exists")

	// Clean up tempoary directory and ensure it no longer exists.
	CleanupDir(testDir)
	_, err = os.Stat(testDir)
	assert.Error(err, "Could not stat temporary directory")
	if err != nil {
		assert.True(os.IsNotExist(err), "Error is of type IsNotExists")
	}
}

// TestChdir tests the Chdir function and ensures it will change to a temporary directory and then properly return
// to the original directory when cleaned up.
func TestChdir(t *testing.T) {
	assert := asrt.New(t)
	// Get the current working directory.
	startingDir, err := os.Getwd()
	assert.NoError(err)

	// Create a temporary directory.
	testDir := CreateTmpDir("TestChdir")
	assert.NotEqual(startingDir, testDir, "Ensure our starting directory and temporary directory are not the same")

	// Change to the temporary directory.
	cleanupFunc := Chdir(testDir)
	currentDir, err := os.Getwd()
	assert.NoError(err)

	// On OSX this are created under /var, but /var is a symlink to /var/private, so we cannot ensure complete equality of these strings.
	assert.Contains(currentDir, testDir, "Ensure the current directory is the temporary directory we created")
	assert.True(reflect.TypeOf(cleanupFunc).Kind() == reflect.Func, "Chdir return is of type function")

	cleanupFunc()
	currentDir, err = os.Getwd()
	assert.NoError(err)
	assert.Equal(currentDir, startingDir, "Ensure we have changed back to the starting directory")

	CleanupDir(testDir)
}

// TestValidTestSite tests the TestSite struct behavior in the case of a valid configuration.
func TestValidTestSite(t *testing.T) {
	assert := asrt.New(t)
	// Get the current working directory.
	startingDir, err := os.Getwd()
	assert.NoError(err, "Could not get current directory.")

	if os.Getenv("DDEV_BINARY_FULLPATH") != "" {
		DdevBin = os.Getenv("DDEV_BINARY_FULLPATH")
	}

	// It's not ideal to copy/paste this archive around, but we don't actually care about the contents
	// of the archive for this test, only that it exists and can be extracted. This should (knock on wood)
	//not need to be updated over time.
	site := TestSites[0]

	// If running this with GOTEST_SHORT we have to create the directory, tarball etc.
	site.Name = "TestValidTestSite"
	_, _ = exec.RunCommand(DdevBin, []string{"stop", "-RO", site.Name})

	//nolint: errcheck
	defer exec.RunCommand(DdevBin, []string{"stop", "-RO", site.Name})
	//nolint: errcheck
	defer globalconfig.RemoveProjectInfo(site.Name)
	err = site.Prepare()
	require.NoError(t, err, "Prepare() failed on TestSite.Prepare() site=%s, err=%v", site.Name, err)

	assert.NotNil(site.Dir, "Directory is set.")
	docroot := filepath.Join(site.Dir, site.Docroot)
	dirStat, err := os.Stat(docroot)
	assert.NoError(err, "Docroot exists after prepare()")
	if err != nil {
		t.Fatalf("Directory did not exist after prepare(): %s", docroot)
	}
	assert.True(dirStat.IsDir(), "Docroot is a directory")

	cleanup := site.Chdir()
	defer cleanup()

	currentDir, err := os.Getwd()
	assert.NoError(err)

	// On OSX this are created under /var, but /var is a symlink to /var/private, so we cannot ensure complete equality of these strings.
	assert.Contains(currentDir, site.Dir)

	cleanup()

	currentDir, err = os.Getwd()
	assert.NoError(err)
	assert.Equal(startingDir, currentDir)

	site.Cleanup()
	_, err = os.Stat(site.Dir)
	assert.Error(err, "Could not stat temporary directory after cleanup")
}

// TestGetLocalHTTPResponse() brings up a project and hits a URL to get the response
func TestGetLocalHTTPResponse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows as we always seem to have port conflicts")
	}
	// We have to get globalconfig read so CA is known and installed.
	err := globalconfig.ReadGlobalConfig()
	require.NoError(t, err)

	assert := asrt.New(t)

	dockerutil.EnsureDdevNetwork()

	if os.Getenv("DDEV_BINARY_FULLPATH") != "" {
		DdevBin = os.Getenv("DDEV_BINARY_FULLPATH")
	}

	// It's not ideal to copy/paste this archive around, but we don't actually care about the contents
	// of the archive for this test, only that it exists and can be extracted. This should (knock on wood)
	//not need to be updated over time.
	site := TestSites[0]
	site.Name = t.Name()

	_, _ = exec.RunCommand(DdevBin, []string{"stop", "-RO", site.Name})
	//nolint: errcheck
	defer exec.RunCommand(DdevBin, []string{"stop", "-RO", site.Name})
	//nolint: errcheck
	defer globalconfig.RemoveProjectInfo(site.Name)

	err = site.Prepare()
	require.NoError(t, err, "Prepare() failed on TestSite.Prepare() site=%s, err=%v", site.Name, err)

	cleanup := site.Chdir()
	defer cleanup()

	app := &ddevapp.DdevApp{}
	err = app.Init(site.Dir)
	assert.NoError(err)
	// nolint: errcheck
	defer app.Stop(true, false)

	for _, pair := range []PortPair{{"80", "443"}, {"8080", "8443"}} {
		ClearDockerEnv()
		app.RouterHTTPPort = pair.HTTPPort
		app.RouterHTTPSPort = pair.HTTPSPort
		err = app.WriteConfig()
		assert.NoError(err)

		startErr := app.StartAndWait(5)
		assert.NoError(startErr, "app.StartAndWait failed for port pair %v", pair)
		if startErr != nil {
			logs, _ := ddevapp.GetErrLogsFromApp(app, startErr)
			t.Fatalf("logs from broken container:\n=======\n%s\n========\n", logs)
		}

		safeURL := app.GetHTTPURL() + site.Safe200URIWithExpectation.URI
		out, _, err := GetLocalHTTPResponse(t, safeURL)
		assert.NoError(err)
		assert.Contains(out, site.Safe200URIWithExpectation.Expect)

		safeURL = app.GetHTTPSURL() + site.Safe200URIWithExpectation.URI
		out, _, err = GetLocalHTTPResponse(t, safeURL)
		assert.NoError(err)
		assert.Contains(out, site.Safe200URIWithExpectation.Expect)

		// This does the same thing as previous, but worth exercising it here.
		_, _ = EnsureLocalHTTPContent(t, safeURL, site.Safe200URIWithExpectation.Expect)
	}
	// Set the ports back to the default was so we don't break any following tests.
	app.RouterHTTPSPort = "443"
	app.RouterHTTPPort = "80"
	err = app.WriteConfig()
	assert.NoError(err)

	err = app.Stop(true, false)
	assert.NoError(err)

	cleanup()

	site.Cleanup()
}

// TestGetCachedArchive tests download and extraction of archives for test sites
// to testcache directory.
func TestGetCachedArchive(t *testing.T) {
	assert := asrt.New(t)

	sourceURL := "https://raw.githubusercontent.com/drud/ddev/master/.gitignore"
	exPath, archPath, err := GetCachedArchive("TestInvalidArchive", "test", "", sourceURL)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), fmt.Sprintf("archive extraction of %s failed", archPath))
	}

	err = os.RemoveAll(filepath.Dir(exPath))
	assert.NoError(err)

	sourceURL = "http://invalid_domain/somefilethatdoesnotexists"
	exPath, archPath, err = GetCachedArchive("TestInvalidDownloadURL", "test", "", sourceURL)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), fmt.Sprintf("Failed to download url=%s into %s", sourceURL, archPath))
	}

	err = os.RemoveAll(filepath.Dir(exPath))
	assert.NoError(err)
}
