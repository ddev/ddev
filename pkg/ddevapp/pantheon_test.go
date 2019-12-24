package ddevapp_test

import (
	"bufio"
	"fmt"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/output"
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
 * need to set an environment variable called "DDEV_PANTHEON_API_TOKEN" with credentials for
 * this account. If no such environment variable is present, these tests will be skipped.
 *
 * A valid site (with backups) must be present which matches the test site and environment name
 * defined in the constants below.
 */
const pantheonTestSiteName = "ddev-test-site-do-not-delete"
const pantheonTestEnvName = "bbowman"

// TestPantheonConfigCommand tests the interactive config options.
func TestPantheonConfigCommand(t *testing.T) {
	if os.Getenv("DDEV_PANTHEON_API_TOKEN") == "" {
		t.Skipf("No DDEV_PANTHEON_API_TOKEN env var has been set. Skipping %v", t.Name())
	}

	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir("TestPantheonConfigCommand")

	// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	// Create a docroot folder.
	err := os.Mkdir(filepath.Join(testDir, "docroot"), 0644)
	if err != nil {
		t.Errorf("Could not create docroot directory under %s", testDir)
	}

	// Create the ddevapp we'll use for testing.
	app, err := NewApp(testDir, true, nodeps.ProviderPantheon)
	assert.NoError(err)

	docroot := "docroot"

	/**
	 * Do a full interactive configuration for a pantheon environment.
	 *
	 * 1. Provide a valid site name. Ensure there is no error.
	 * 2. Provide a valid docroot (already tested elsewhere)
	 * 3. Provide a valid app type (drupal8)
	 * 4. Provide a valid environment name.
	 **/
	input := fmt.Sprintf("%s\n%s\ndocroot\ndrupal8\n%s", pantheonTestSiteName, docroot, pantheonTestEnvName)
	scanner := bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)

	restoreOutput := util.CaptureUserOut()
	err = app.PromptForConfig()
	assert.NoError(err, t)
	out := restoreOutput()

	// Get the provider interface and ensure it validates.
	provider, err := app.GetProvider()
	assert.NoError(err)
	err = provider.Validate()
	assert.NoError(err)

	// Ensure we have expected string values in output.
	assert.Contains(out, testDir)

	// Ensure values were properly set on the app struct.
	assert.Equal(pantheonTestSiteName, app.Name)
	assert.Equal(nodeps.AppTypeDrupal8, app.Type)
	assert.Equal("docroot", app.Docroot)
	err = PrepDdevDirectory(testDir)
	assert.NoError(err)
	output.UserOut.Print("")
}

// TestPantheonBackupLinks ensures we can get backups from pantheon for a configured environment.
func TestPantheonBackupLinks(t *testing.T) {
	if os.Getenv("DDEV_PANTHEON_API_TOKEN") == "" {
		t.Skipf("No DDEV_PANTHEON_API_TOKEN env var has been set. Skipping %v", t.Name())
	}

	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir("TestPantheonBackupLinks")

	// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	app, err := NewApp(testDir, true, nodeps.ProviderPantheon)
	assert.NoError(err)

	app.Name = pantheonTestSiteName

	provider := PantheonProvider{}
	err = provider.Init(app)
	assert.NoError(err)

	provider.Sitename = pantheonTestSiteName
	provider.EnvironmentName = pantheonTestEnvName

	// Ensure GetBackup triggers an error for unknown backup types.
	_, _, err = provider.GetBackup(util.RandString(8), "")
	assert.Error(err)

	// Ensure we can get a backupLink
	backupLink, importPath, err := provider.GetBackup("database", "")
	assert.NoError(err)

	assert.Equal(importPath, "")
	assert.Contains(backupLink, "database.sql.gz")
	output.UserOut.Print("")
}

// TestPantheonPull ensures we can pull backups from pantheon for a configured environment.
func TestPantheonPull(t *testing.T) {
	if os.Getenv("DDEV_PANTHEON_API_TOKEN") == "" {
		t.Skipf("No DDEV_PANTHEON_API_TOKEN env var has been set. Skipping %v", t.Name())
	}

	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir("TestPantheonPull")

	// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()
	// Move into the properly named pantheon site (must match pantheon sitename)
	siteDir := filepath.Join(testDir, pantheonTestSiteName)
	err := os.MkdirAll(filepath.Join(siteDir, "sites/default"), 0777)
	assert.NoError(err)
	err = os.Chdir(siteDir)
	assert.NoError(err)

	app, err := NewApp(siteDir, true, nodeps.ProviderPantheon)
	assert.NoError(err)

	// nolint: errcheck
	defer app.Stop(true, false)

	app.Name = pantheonTestSiteName
	app.Type = nodeps.AppTypeDrupal8
	app.Hooks = map[string][]YAMLTask{"post-pull": {{"exec-host": "touch hello-post-pull-" + app.Name}}, "pre-pull": {{"exec-host": "touch hello-pre-pull-" + app.Name}}}

	err = app.WriteConfig()
	assert.NoError(err)

	testcommon.ClearDockerEnv()

	provider := PantheonProvider{}
	err = provider.Init(app)
	assert.NoError(err)

	provider.Sitename = pantheonTestSiteName
	provider.EnvironmentName = pantheonTestEnvName
	err = provider.Write(app.GetConfigPath("import.yaml"))
	assert.NoError(err)

	// Ensure we can do a pull on the configured site.
	app, err = GetActiveApp("")
	assert.NoError(err)
	err = app.Pull(&provider, &PullOptions{})
	assert.NoError(err)

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
