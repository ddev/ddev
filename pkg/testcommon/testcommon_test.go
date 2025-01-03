package testcommon

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var DdevBin = "ddev"

func init() {
	globalconfig.EnsureGlobalConfig()
}

var TestSites = []TestSite{
	{
		SourceURL:                     "https://wordpress.org/wordpress-5.8.2.tar.gz",
		ArchiveInternalExtractionPath: "wordpress/",
		FilesTarballURL:               "https://github.com/ddev/ddev_test_tarballs/releases/download/v1.1/wordpress5.8.2_files.tar.gz",
		DBTarURL:                      "https://github.com/ddev/ddev_test_tarballs/releases/download/v1.1/wordpress5.8.2_db.sql.tar.gz",
		Docroot:                       "",
		Type:                          nodeps.AppTypeWordPress,
		Safe200URIWithExpectation:     URIWithExpect{URI: "/readme.html", Expect: "Welcome. WordPress is a very special project to me."},
		DynamicURI:                    URIWithExpect{URI: "/", Expect: "this post has a photo"},
		FilesImageURI:                 "/wp-content/uploads/2021/12/DSCF0436-randy-and-nancy-with-traditional-wedding-out-fit-2048x1536.jpg",
		Name:                          "TestCmdWordpress",
		HTTPProbeURI:                  "wp-admin/setup-config.php",
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

	// Clean up temporary directory and ensure it no longer exists.
	CleanupDir(testDir)
	_, err = os.Stat(testDir)
	assert.Error(err, "Could not stat temporary directory")
	if err != nil {
		assert.True(os.IsNotExist(err), "Error is of type IsNotExists")
	}
}

// TestValidTestSite tests the TestSite struct behavior in the case of a valid configuration.
func TestValidTestSite(t *testing.T) {
	assert := asrt.New(t)

	if os.Getenv("DDEV_BINARY_FULLPATH") != "" {
		DdevBin = os.Getenv("DDEV_BINARY_FULLPATH")
	}
	origDir, _ := os.Getwd()

	// It's not ideal to copy/paste this archive around, but we don't actually care about the contents
	// of the archive for this test, only that it exists and can be extracted. This should (knock on wood)
	//not need to be updated over time.
	site := TestSites[0]

	// If running this with GOTEST_SHORT we have to create the directory, tarball etc.
	site.Name = t.Name()
	_, _ = exec.RunCommand(DdevBin, []string{"stop", "-RO", site.Name})

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_, err := exec.RunCommand(DdevBin, []string{"delete", "-Oy", site.Name})
		assert.NoError(err)
		site.Cleanup()
		_, err = os.Stat(site.Dir)
		assert.Error(err, "Could not stat temporary directory after cleanup")
	})
	err := site.Prepare()
	require.NoError(t, err, "Prepare() failed on TestSite.Prepare() site=%s, err=%v", site.Name, err)

	docroot := filepath.Join(site.Dir, site.Docroot)
	dirStat, err := os.Stat(docroot)
	assert.NoError(err, "Docroot exists after prepare()")
	if err != nil {
		t.Fatalf("Directory did not exist after prepare(): %s", docroot)
	}
	assert.True(dirStat.IsDir(), "Docroot is a directory")

	err = os.Chdir(site.Dir)
	require.NoError(t, err)

	currentDir, _ := os.Getwd()

	assert.Equal(currentDir, site.Dir)
}

// TestGetLocalHTTPResponse() brings up a project and hits a URL to get the response
func TestGetLocalHTTPResponse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows as it always seems to fail starting 2023-04")
	}
	if dockerutil.IsColima() || dockerutil.IsLima() {
		t.Skip("Skipping on Lima/Colima")
	}
	// We have to get globalconfig read so CA is known and installed.
	err := globalconfig.ReadGlobalConfig()
	require.NoError(t, err)

	assert := asrt.New(t)

	origDir, _ := os.Getwd()

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

	err = site.Prepare()
	require.NoError(t, err, "Prepare() failed on TestSite.Prepare() site=%s, err=%v", site.Name, err)

	app := &ddevapp.DdevApp{}
	err = app.Init(site.Dir)
	assert.NoError(err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)

		err = app.Stop(true, false)
		assert.NoError(err)

		app.RouterHTTPSPort = ""
		app.RouterHTTPPort = ""
		err = app.WriteConfig()
		assert.NoError(err)

		site.Cleanup()
	})

	for _, pair := range []PortPair{{"8000", "8043"}, {"8080", "8443"}} {
		ClearDockerEnv()
		app.RouterHTTPPort = pair.HTTPPort
		app.RouterHTTPSPort = pair.HTTPSPort
		err = app.WriteConfig()
		assert.NoError(err)

		startErr := app.StartAndWait(5)
		assert.NoError(startErr, "app.StartAndWait failed for port pair %v", pair)
		if startErr != nil {
			logs, health, _ := ddevapp.GetErrLogsFromApp(app, startErr)
			t.Fatalf("healthcheck:\n%s\n\nlogs from broken container:\n=======\n%s\n========\n", health, logs)
		}

		safeURL := app.GetHTTPURL() + site.Safe200URIWithExpectation.URI

		// Extra dummy GetLocalHTTPResponse is for mac M1 to try to prime it.
		_, _, _ = GetLocalHTTPResponse(t, safeURL, 60)
		out, _, err := GetLocalHTTPResponse(t, safeURL, 60)
		assert.NoError(err)
		assert.Contains(out, site.Safe200URIWithExpectation.Expect)

		// Skip the https version if we don't have mkcert working
		if globalconfig.GetCAROOT() != "" {
			safeURL = app.GetHTTPSURL() + site.Safe200URIWithExpectation.URI
			out, _, err = GetLocalHTTPResponse(t, safeURL, 60)
			assert.NoError(err)
			assert.Contains(out, site.Safe200URIWithExpectation.Expect)
			// This does the same thing as previous, but worth exercising it here.
			_, _ = EnsureLocalHTTPContent(t, safeURL, site.Safe200URIWithExpectation.Expect)
		}
	}

}

// TestGetCachedArchive tests download and extraction of archives for test sites
// to testcache directory.
func TestGetCachedArchive(t *testing.T) {
	sourceURL := "https://raw.githubusercontent.com/ddev/ddev/main/.gitignore"
	exPath, archPath, err := GetCachedArchive("TestInvalidArchive", "test", "", sourceURL)
	require.Error(t, err)
	if err != nil {
		require.Contains(t, err.Error(), fmt.Sprintf("archive extraction of %s failed", archPath))
	}

	err = os.RemoveAll(exPath)
	require.NoError(t, err)

	err = os.RemoveAll(archPath)
	require.NoError(t, err)

	sourceURL = "http://invalid_domain/somefilethatdoesnotexists"
	exPath, archPath, err = GetCachedArchive("TestInvalidDownloadURL", "test", "", sourceURL)
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("failed to download url=%s into %s", sourceURL, archPath))

	err = os.RemoveAll(exPath)
	require.NoError(t, err)

	err = os.RemoveAll(archPath)
	require.NoError(t, err)
}

// TestPretestAndEnv tests that the testsite PretestCmd works along with WebEvironment
func TestPretestAndEnv(t *testing.T) {
	assert := asrt.New(t)

	origDir, err := os.Getwd()
	require.NoError(t, err)
	site := TestSites[0]
	site.Name = t.Name()

	_, _ = exec.RunCommand(DdevBin, []string{"delete", "-Oy", site.Name})

	site.WebEnvironment = []string{"SOMEVAR=somevar"}
	site.PretestCmd = fmt.Sprintf("%s exec 'touch /var/tmp/%s'", DdevBin, t.Name())
	err = site.Prepare()
	require.NoError(t, err, "Prepare() failed on TestSite.Prepare() site=%s, err=%v", site.Name, err)

	err = os.Chdir(site.Dir)
	require.NoError(t, err)

	app := &ddevapp.DdevApp{}
	err = app.Init(site.Dir)
	assert.NoError(err)

	t.Cleanup(func() {
		site.WebEnvironment = []string{}
		site.PretestCmd = ""

		err = app.Stop(true, false)
		assert.NoError(err)
		_ = globalconfig.RemoveProjectInfo(site.Name)
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	if runtime.GOOS == "windows" && app.IsMutagenEnabled() {
		// We can't replace mutagen.exe on windows if anything has been using it
		ddevapp.PowerOff()
		err = ddevapp.MutagenReset(app)
		require.NoError(t, err)
		time.Sleep(1000 * time.Millisecond)
	}

	err = app.Start()
	require.NoError(t, err)
	somevar, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "printf ${SOMEVAR}",
	})
	require.NoError(t, err)
	require.Equal(t, "somevar", somevar, "did not find env var SOMEVAR=somevar; output=%s", somevar)

	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: fmt.Sprintf("ls -l /var/tmp/%s", t.Name()),
	})
	require.NoError(t, err, "Error testing for existence of /var/tmp/%s; output=%s", t.Name(), out)
}
