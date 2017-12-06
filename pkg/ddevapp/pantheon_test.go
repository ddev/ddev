package ddevapp_test

import (
	"bufio"
	"fmt"
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

// TestConfigCommand tests the interactive config options.
func TestPantheonConfigCommand(t *testing.T) {
	if os.Getenv("DDEV_PANTHEON_API_TOKEN") == "" {
		t.Skip("No DDEV_PANTHEON_API_TOKEN env var has been set. Skipping Pantheon specific test.")
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
	config, err := NewConfig(testDir, "pantheon")
	assert.NoError(err)

	// Randomize some values to use for Stdin during testing.
	invalidName := strings.ToLower(util.RandString(16))
	docroot := "docroot"
	invalidEnvironment := strings.ToLower(util.RandString(8))

	/**
	 * Do a full interactive configuration for a pantheon environment.
	 *
	 * 1. Provide an invalid site name, ensure there is an error.
	 * 2. Provide a valid site name. Ensure there is no error.
	 * 3. Provide a valid docroot (already tested elsewhere)
	 * 4. Provide a valid app type (drupal8)
	 * 5. Provide an invalid pantheon environment name, ensure an error is triggered.
	 * 6. Provide a valid environment name.
	 **/
	input := fmt.Sprintf("%s\n%s\n%s\ndocroot\ndrupal8\n%s\n%s", invalidName, pantheonTestSiteName, docroot, invalidEnvironment, pantheonTestEnvName)
	scanner := bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)

	restoreOutput := testcommon.CaptureUserOut()
	err = config.PromptForConfig()
	assert.NoError(err, t)
	out := restoreOutput()

	// Get the provider interface and ensure it validates.
	provider, err := config.GetProvider()
	assert.NoError(err)
	err = provider.Validate()
	assert.NoError(err)

	// Ensure we have expected string values in output.
	assert.Contains(out, testDir)
	assert.Contains(out, fmt.Sprintf("could not find a pantheon site named %s", invalidName))
	assert.Contains(out, fmt.Sprintf("could not find an environment named '%s'", invalidEnvironment))

	// Ensure values were properly set on the config struct.
	assert.Equal(pantheonTestSiteName, config.Name)
	assert.Equal("drupal8", config.AppType)
	assert.Equal("docroot", config.Docroot)
	err = PrepDdevDirectory(testDir)
	assert.NoError(err)
}

// TestPantheonBackupLinks ensures we can get backups from pantheon for a configured environment.
func TestPantheonBackupLinks(t *testing.T) {
	if os.Getenv("DDEV_PANTHEON_API_TOKEN") == "" {
		t.Skip("No DDEV_PANTHEON_API_TOKEN env var has been set. Skipping Pantheon specific test.")
	}

	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir("TestPantheonBackupLinks")

	// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	config, err := NewConfig(testDir, "pantheon")
	assert.NoError(err)
	config.Name = pantheonTestSiteName

	provider := PantheonProvider{}
	err = provider.Init(config)
	assert.NoError(err)

	provider.Sitename = pantheonTestSiteName
	provider.EnvironmentName = pantheonTestEnvName

	// Ensure GetBackup triggers an error for unknown backup types.
	_, _, err = provider.GetBackup(util.RandString(8))
	assert.Error(err)

	// Ensure we can get a
	backupLink, importPath, err := provider.GetBackup("database")

	assert.Equal(importPath, "")
	assert.Contains(backupLink, "database.sql.gz")
	assert.NoError(err)
}

// TestPantheonPull ensures we can pull backups from pantheon for a configured environment.
func TestPantheonPull(t *testing.T) {
	if os.Getenv("DDEV_PANTHEON_API_TOKEN") == "" {
		t.Skip("No DDEV_PANTHEON_API_TOKEN env var has been set. Skipping Pantheon specific test.")
	}

	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir("TestPantheonPull")

	// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	// Move into the properly named pantheon site (must match pantheon sitename)
	siteDir := testDir + "/" + pantheonTestSiteName
	err := os.MkdirAll(siteDir+"/sites/default", 0777)
	assert.NoError(err)
	err = os.Chdir(siteDir)
	assert.NoError(err)

	config, err := NewConfig(siteDir, "pantheon")
	assert.NoError(err)
	config.Name = pantheonTestSiteName
	config.AppType = "drupal8"
	err = config.Write()
	assert.NoError(err)

	testcommon.ClearDockerEnv()

	provider := PantheonProvider{}
	err = provider.Init(config)
	assert.NoError(err)

	provider.Sitename = pantheonTestSiteName
	provider.EnvironmentName = pantheonTestEnvName
	err = provider.Write(config.GetPath("import.yaml"))
	assert.NoError(err)

	// Ensure we can do a pull on the configured site.
	app, err := GetActiveApp("")
	assert.NoError(err)
	err = app.Import()
	assert.NoError(err)
	err = app.Down(true)
	assert.NoError(err)
}
