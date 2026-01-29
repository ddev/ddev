package ddevapp_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

/**
 * These tests rely on an external test account. To run them, you'll
 * need to set an environment variable called "DDEV_PLATFORM_API_TOKEN" with credentials for
 * this account. If no such environment variable is present, these tests will be skipped.
 *
 * A valid site must be present which matches the test site and environment name
 * defined in the constants below.
 */

const platformTestSiteID = "uwok34bf5555a"
const platformPullTestSiteEnvironment = "platform-pull"
const platformPushTestSiteEnvironment = "platform-push"

const platformPullSiteURL = "https://platform-pull-7tsp6cq-uwok34bf5555a.ca-1.platformsh.site/"
const platformSiteExpectation = "Super easy vegetarian pasta"

// Note that these tests won't run with GitHub actions on a forked PR.
// This is a security feature, but means that PRs intended to test this
// must be done in the DDEV repo.

// TestPlatformPull ensures we can pull database/files from Upsun Fixed for a configured environment.
func TestPlatformPull(t *testing.T) {
	util.SkipIfEmbargoed(t)

	var token string
	if token = os.Getenv("DDEV_PLATFORM_API_TOKEN"); token == "" {
		t.Skipf("No DDEV_PLATFORM_API_TOKEN env var has been set. Skipping %v", t.Name())
	}
	var err error

	require.True(t, isPullSiteValid(platformPullSiteURL, platformSiteExpectation), "platformPullSiteURL %s isn't working right", platformPullSiteURL)

	origDir, _ := os.Getwd()

	app, provider, err := setupPlatformProject(t, platformPullTestSiteEnvironment)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		require.NoError(t, err)
		_ = os.Chdir(origDir)
		_ = os.RemoveAll(app.AppRoot)
	})

	// variant using .platform/local/project.yaml
	t.Run("local-config", func(t *testing.T) {
		err = writePlatformLocalConfig(t, app, token)
		require.NoError(t, err)
		startAndCheckPlatformPull(t, app, provider)
	})

	// variant using environment variables
	t.Run("environment-var-config", func(t *testing.T) {
		provider.EnvironmentVariables["PLATFORM_PROJECT"] = platformTestSiteID
		provider.EnvironmentVariables["PLATFORM_ENVIRONMENT"] = platformPullTestSiteEnvironment
		provider.EnvironmentVariables["PLATFORMSH_CLI_TOKEN"] = token
		_ = os.RemoveAll(filepath.Join(app.AppRoot, ".platform/local/"))
		startAndCheckPlatformPull(t, app, provider)
	})
}

// TestPlatformPush ensures we can push to Upsun Fixed for a configured environment.
func TestPlatformPush(t *testing.T) {
	util.SkipIfEmbargoed(t)

	var token string
	if token = os.Getenv("DDEV_PLATFORM_API_TOKEN"); token == "" {
		t.Skipf("No DDEV_PLATFORM_API_TOKEN env var has been set. Skipping %v", t.Name())
	}

	origDir, _ := os.Getwd()

	app, provider, err := setupPlatformProject(t, platformPushTestSiteEnvironment)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		require.NoError(t, err)
		_ = os.Chdir(origDir)
		_ = os.RemoveAll(app.AppRoot)
	})

	t.Run("local-config", func(t *testing.T) {
		err = writePlatformLocalConfig(t, app, token)
		require.NoError(t, err)
		startAndCheckPlatformPush(t, app, provider, token)
	})

	t.Run("environment-based-config", func(t *testing.T) {
		provider.EnvironmentVariables["PLATFORM_PROJECT"] = platformTestSiteID
		provider.EnvironmentVariables["PLATFORM_ENVIRONMENT"] = platformPushTestSiteEnvironment
		provider.EnvironmentVariables["PLATFORMSH_CLI_TOKEN"] = token
		_ = os.RemoveAll(filepath.Join(app.AppRoot, ".platform/local/"))
		startAndCheckPlatformPush(t, app, provider, token)
	})
}

// writePlatformLocalConfig writes the .platform/local directory with a project.yaml file
// And sets the required environment variables for the provider.
func writePlatformLocalConfig(t *testing.T, app *ddevapp.DdevApp, token string) error {
	err := os.MkdirAll(filepath.Join(app.AppRoot, ".platform/local"), 0755)
	require.NoError(t, err)
	// Provide a project.yaml to
	err = os.WriteFile(filepath.Join(app.AppRoot, ".platform/local/project.yaml"), []byte(fmt.Sprintf("id: %s\nhost: api.platform.sh", platformTestSiteID)), 0644)
	require.NoError(t, err)

	app.ProviderInstance.EnvironmentVariables["PLATFORMSH_CLI_TOKEN"] = token
	return nil
}

// setupPlatformProject does basic setup, creating project, etc.
func setupPlatformProject(t *testing.T, environment string) (*ddevapp.DdevApp, *ddevapp.Provider, error) {
	testName := strings.Split(t.Name(), "/")[0]
	siteDir := testcommon.CreateTmpDir(testName)

	err := globalconfig.RemoveProjectInfo(testName)
	require.NoError(t, err)

	err = os.Chdir(siteDir)
	require.NoError(t, err)

	// Initialize a git repository and create the branch needed for the test.
	// This runs in the temporary siteDir because we've already chdir'ed into it.
	out, err := exec.RunHostCommand("bash", "-c", fmt.Sprintf("git init && git checkout -b %s", environment))
	require.NoError(t, err, "failed to initialize git repository and create branch '%s': output='%s'", environment, out)

	app, err := ddevapp.NewApp(siteDir, true)
	require.NoError(t, err)
	app.Name = testName
	app.Type = nodeps.AppTypeDrupal11
	err = app.Stop(true, false)
	require.NoError(t, err)
	err = app.WriteConfig()
	require.NoError(t, err)

	testcommon.ClearDockerEnv()

	err = ddevapp.PopulateExamplesCommandsHomeadditions(app.Name)
	require.NoError(t, err)

	app.Docroot = "web"
	err = app.WriteConfig()
	require.NoError(t, err)

	provider, err := app.GetProvider("platform")
	require.NoError(t, err)

	return app, provider, nil
}

// startAndCheckPlatformPull starts the app and does platform pull, then checks for the expected file and database entry.
func startAndCheckPlatformPull(t *testing.T, app *ddevapp.DdevApp, provider *ddevapp.Provider) {
	err := app.Start()
	require.NoError(t, err)
	err = app.Pull(provider, false, false, false)
	require.NoError(t, err)

	require.FileExists(t, filepath.Join(app.GetHostUploadDirFullPath(), "victoria-sponge-umami.jpg"))
	out, err := exec.RunHostCommand("bash", "-c", fmt.Sprintf(`echo 'select COUNT(*) from users_field_data where mail="margaret.hopper@example.com";' | %s mysql -N`, DdevBin))
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(out, "1\n"))
}

// startAndCheckPlatformPush starts the app and does platform push, then checks for the expected file and database entry.
func startAndCheckPlatformPush(t *testing.T, app *ddevapp.DdevApp, provider *ddevapp.Provider, token string) {
	err := app.Start()
	require.NoError(t, err)

	testName := strings.Split(t.Name(), "/")[0]

	// Create database and files entries that we can verify after push
	tval := nodeps.RandomString(10)
	tableName := testName
	c := fmt.Sprintf(`%s -e 'CREATE TABLE IF NOT EXISTS %s ( title VARCHAR(255) NOT NULL ); INSERT INTO %s VALUES("%s");'`, app.GetDBClientCommand(), tableName, tableName, tval)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: c,
	})
	require.NoError(t, err)
	fName := tval + ".txt"
	fContent := []byte(tval)
	err = os.WriteFile(filepath.Join(app.AppRoot, "web/sites/default/files", fName), fContent, 0644)
	require.NoError(t, err)

	err = app.Push(provider, false, false)
	require.NoError(t, err)

	// Test that the database row was added in the upstream platform project
	c = fmt.Sprintf(`echo 'SELECT title FROM %s WHERE title="%s";' | PLATFORMSH_CLI_TOKEN=%s platform db:sql --project="%s" --environment="%s"`, tableName, tval, token, platformTestSiteID, platformPushTestSiteEnvironment)
	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: c,
	})
	require.NoError(t, err)
	require.Contains(t, out, tval)

	// Test that the file arrived there (by rsyncing it back)

	tmpRsyncDir := filepath.Join("/tmp", testName+util.RandString(5))
	c = fmt.Sprintf(`PLATFORMSH_CLI_TOKEN=%s platform mount:download --yes --quiet --project="%s" --environment="%s" --mount=web/sites/default/files --target=%s && cat %s/%s && rm -rf %s`, token, platformTestSiteID, platformPushTestSiteEnvironment, tmpRsyncDir, tmpRsyncDir, fName, tmpRsyncDir)
	out, stderr, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: c,
	})
	require.NoError(t, err, "output='%s', stderr='%s'", out, stderr)
	require.Contains(t, out, tval)
}
