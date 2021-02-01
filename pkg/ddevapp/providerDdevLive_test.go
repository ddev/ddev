package ddevapp_test

import (
	"bufio"
	"fmt"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/output"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
)

/**
 * These tests rely on an external test account managed by DRUD. To run them, you'll
 * need to set an environment variable called "DDEV_DDEVLIVE_API_TOKEN" with credentials for
 * this account. If no such environment variable is present, these tests will be skipped.
 *
 * A valid site (with backups) must be present which matches the test site and environment name
 * defined in the constants below.
 */
const ddevliveTestSite = "ddev-live-test-no-delete"
const ddevLiveOrgName = "ddltest"

// TestDdevLiveConfigCommand tests the interactive config options.
func TestDdevLiveConfigCommand(t *testing.T) {
	if os.Getenv("DDEV_DDEVLIVE_API_TOKEN") == "" {
		t.Skipf("No DDEV_DDEVLIVE_API_TOKEN env var has been set. Skipping %v", t.Name())
	}
	_ = os.Setenv("DDEV_LIVE_NO_ANALYTICS", "true")

	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir(t.Name())

	// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	docroot := "web"

	// Create the docroot.
	err := os.Mkdir(filepath.Join(testDir, docroot), 0755)
	if err != nil {
		t.Errorf("Could not create %s directory under %s", docroot, testDir)
	}

	// Create the ddevapp we'll use for testing.
	app, err := NewApp(testDir, true, nodeps.ProviderDdevLive)
	assert.NoError(err)

	/**
	 * Do a full interactive configuration for a ddev-live environment.
	 *
	 * 1. Provide a valid site name. Ensure there is no error.
	 * 2. Provide a valid docroot (already tested elsewhere)
	 * 3. Provide a valid project type (drupal8)
	 **/
	input := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n", ddevliveTestSite, docroot, "drupal8", ddevLiveOrgName, ddevliveTestSite)
	scanner := bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)

	restoreOutput := util.CaptureUserOut()
	err = app.PromptForConfig()
	assert.NoError(err, t)
	out := restoreOutput()

	// Get the provider interface and ensure it validates.
	provider, err := app.GetProvider("")
	assert.NoError(err)
	err = provider.Validate()
	assert.NoError(err)

	// Ensure we have expected string values in output.
	assert.Contains(out, testDir)

	// Ensure values were properly set on the app struct.
	assert.Equal(ddevliveTestSite, app.Name)
	assert.Equal(nodeps.AppTypeDrupal8, app.Type)
	assert.Equal(docroot, app.Docroot)
	require.Equal(t, "*ddevapp.DdevLiveProvider", fmt.Sprintf("%T", provider))
	realProvider := provider.(*DdevLiveProvider)
	assert.Equal(ddevliveTestSite, realProvider.SiteName)
	assert.Equal(ddevLiveOrgName, realProvider.OrgName)
	err = PrepDdevDirectory(testDir)
	assert.NoError(err)
	output.UserOut.Print("")
}

// TestDdevLivePull ensures we can pull backups from ddev-live .
func TestDdevLivePull(t *testing.T) {
	if os.Getenv("DDEV_DDEVLIVE_API_TOKEN") == "" {
		t.Skipf("No DDEV_DDEVLIVE_API_TOKEN env var has been set. Skipping %v", t.Name())
	}
	_ = os.Setenv("DDEV_LIVE_NO_ANALYTICS", "true")

	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir(t.Name())

	// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()
	// Move into the properly named ddev-live site (must match ddev-live sitename)
	siteDir := filepath.Join(testDir, ddevliveTestSite)
	err := os.MkdirAll(filepath.Join(siteDir, "web", "sites/default"), 0777)
	assert.NoError(err)
	err = os.Chdir(siteDir)
	assert.NoError(err)

	app, err := NewApp(siteDir, true, nodeps.ProviderDdevLive)
	assert.NoError(err)

	// nolint: errcheck
	defer app.Stop(true, false)

	app.Name = t.Name()
	app.Type = nodeps.AppTypeDrupal8
	app.Docroot = "web"
	_ = os.MkdirAll(filepath.Join(app.AppRoot, app.Docroot, "sites/default/files"), 0755)

	app.Hooks = map[string][]YAMLTask{"post-pull": {{"exec-host": "touch hello-post-pull-" + app.Name}}, "pre-pull": {{"exec-host": "touch hello-pre-pull-" + app.Name}}}

	err = app.WriteConfig()
	assert.NoError(err)

	testcommon.ClearDockerEnv()

	provider := DdevLiveProvider{}
	err = provider.Init(app)
	require.NoError(t, err)

	provider.SiteName = ddevliveTestSite
	provider.OrgName = ddevLiveOrgName
	err = provider.Write(app.GetConfigPath("import.yaml"))
	require.NoError(t, err)
	err = app.WriteConfig()
	require.NoError(t, err)
	// Ensure we can do a pull on the configured site.
	app, err = GetActiveApp("")
	assert.NoError(err)
	err = app.Pull(&provider, &PullOptions{})
	require.NoError(t, err)

	// Verify that we got the special file created in this site.
	assert.FileExists(filepath.Join(app.AppRoot, "web/sites/default/files/i-exist-in-ddev-pull.txt"))

	// Make sure that we have the actual database from the site
	stdout, _, err := app.Exec(&ExecOpts{
		Service: "db",
		Cmd:     "mysql -N -e 'select name from users_field_data where uid=2;' | cat",
	})
	assert.NoError(err)
	assert.Equal("test-account-for-ddev-tests", strings.Trim(stdout, "\n"))

	assert.FileExists("hello-pre-pull-" + app.Name)
	assert.FileExists("hello-post-pull-" + app.Name)
	err = os.Remove("hello-pre-pull-" + app.Name)
	assert.NoError(err)
	err = os.Remove("hello-post-pull-" + app.Name)
	assert.NoError(err)

	app.Hooks = nil
	_ = app.WriteConfig()
	err = app.Stop(true, false)
	assert.NoError(err)
	output.UserOut.Print("")
}
