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
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/**
 * These tests rely on an external test account. To run them, you'll
 * need to set an environment variable called "DDEV_UPSUN_API_TOKEN" with credentials for
 * this account. If no such environment variable is present, these tests will be skipped.
 *
 * A valid site must be present which matches the test site and environment name
 * defined in the constants below.
 */

const upsunTestSiteID = "nqorjgl7ohua4"
const upsunPullTestSiteEnvironment = "pull"
const upsunPushTestSiteEnvironment = "push"

const upsunPullSiteURL = "https://pull-yx2voha-nqorjgl7ohua4.ca-1.platformsh.site/"
const upsunSiteExpectation = "Super easy vegetarian pasta"

// Note that these tests won't run with GitHub actions on a forked PR.
// This is a security feature, but means that PRs intended to test this
// must be done in the DDEV repo.

// TestUpsunPull ensures we can pull backups from Upsun for a configured environment.
func TestUpsunPull(t *testing.T) {
	var token string
	if token = os.Getenv("DDEV_UPSUN_API_TOKEN"); token == "" {
		t.Skipf("No DDEV_UPSUN_API_TOKEN env var has been set. Skipping %v", t.Name())
	}
	assert := asrt.New(t)
	var err error

	require.True(t, isPullSiteValid(upsunPullSiteURL, upsunSiteExpectation), "upsunPullSiteURL %s isn't working right", upsunPullSiteURL)

	origDir, _ := os.Getwd()

	siteDir := testcommon.CreateTmpDir(t.Name())

	err = globalconfig.RemoveProjectInfo(t.Name())
	require.NoError(t, err)

	err = os.Chdir(siteDir)
	assert.NoError(err)
	app, err := ddevapp.NewApp(siteDir, true)
	assert.NoError(err)
	app.Name = t.Name()
	app.Type = nodeps.AppTypeDrupal10
	err = app.Stop(true, false)
	require.NoError(t, err)
	err = app.WriteConfig()
	require.NoError(t, err)

	testcommon.ClearDockerEnv()

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)

		_ = os.Chdir(origDir)
		err = os.RemoveAll(siteDir)
		assert.NoError(err)
	})

	err = ddevapp.PopulateExamplesCommandsHomeadditions(app.Name)
	require.NoError(t, err)

	app.Docroot = "web"
	err = app.WriteConfig()
	require.NoError(t, err)

	provider, err := app.GetProvider("upsun")
	require.NoError(t, err)

	provider.EnvironmentVariables["UPSUN_PROJECT"] = upsunTestSiteID
	provider.EnvironmentVariables["UPSUN_ENVIRONMENT"] = upsunPullTestSiteEnvironment
	provider.EnvironmentVariables["UPSUN_CLI_TOKEN"] = token

	err = app.Start()
	require.NoError(t, err)
	err = app.Pull(provider, false, false, false)
	require.NoError(t, err)

	assert.FileExists(filepath.Join(app.GetHostUploadDirFullPath(), "victoria-sponge-umami.jpg"))
	out, err := exec.RunHostCommand("bash", "-c", fmt.Sprintf(`echo 'select COUNT(*) from users_field_data where mail="margaret.hopper@example.com";' | %s mysql -N`, DdevBin))
	assert.NoError(err)
	assert.True(strings.HasPrefix(out, "1\n"))
}

// TestUpsunPush ensures we can push to Upsun for a configured environment.
func TestUpsunPush(t *testing.T) {
	var token string
	if token = os.Getenv("DDEV_UPSUN_API_TOKEN"); token == "" {
		t.Skipf("No DDEV_UPSUN_API_TOKEN env var has been set. Skipping %v", t.Name())
	}

	assert := asrt.New(t)
	origDir, _ := os.Getwd()

	siteDir := testcommon.CreateTmpDir(t.Name())

	err := os.Chdir(siteDir)
	require.NoError(t, err)

	err = globalconfig.RemoveProjectInfo(t.Name())
	require.NoError(t, err)

	app, err := ddevapp.NewApp(siteDir, true)
	assert.NoError(err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)

		_ = os.Chdir(origDir)
		err = os.RemoveAll(siteDir)
		assert.NoError(err)
	})

	app.Name = t.Name()
	app.Type = nodeps.AppTypeDrupal10
	app.Hooks = map[string][]ddevapp.YAMLTask{"post-push": {{"exec-host": "touch hello-post-push-" + app.Name}}, "pre-push": {{"exec-host": "touch hello-pre-push-" + app.Name}}}
	_ = app.Stop(true, false)

	app.Docroot = "web"

	err = app.WriteConfig()
	require.NoError(t, err)

	testcommon.ClearDockerEnv()

	err = ddevapp.PopulateExamplesCommandsHomeadditions(app.Name)
	require.NoError(t, err)

	provider, err := app.GetProvider("upsun")
	require.NoError(t, err)

	provider.EnvironmentVariables["UPSUN_PROJECT"] = upsunTestSiteID
	provider.EnvironmentVariables["UPSUN_ENVIRONMENT"] = upsunPushTestSiteEnvironment
	provider.EnvironmentVariables["UPSUN_CLI_TOKEN"] = token

	err = app.Start()
	require.NoError(t, err)

	// Create database and files entries that we can verify after push
	tval := nodeps.RandomString(10)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: fmt.Sprintf(`mysql -e 'CREATE TABLE IF NOT EXISTS %s ( title VARCHAR(255) NOT NULL ); INSERT INTO %s VALUES("%s");'`, t.Name(), t.Name(), tval),
	})
	require.NoError(t, err)
	fName := tval + ".txt"
	fContent := []byte(tval)
	err = os.WriteFile(filepath.Join(siteDir, "web/sites/default/files", fName), fContent, 0644)
	assert.NoError(err)

	err = app.Push(provider, false, false)
	require.NoError(t, err)

	// Test that the database row was added
	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: fmt.Sprintf(`echo 'SELECT title FROM %s WHERE title="%s";' | UPSUN_CLI_TOKEN=%s upsun db:sql --project="%s" --environment="%s"`, t.Name(), tval, token, upsunTestSiteID, upsunPushTestSiteEnvironment),
	})
	require.NoError(t, err)
	assert.Contains(out, tval)

	// Test that the file arrived there (by rsyncing it back)
	tmpRsyncDir := filepath.Join("/tmp", t.Name()+util.RandString(5))
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: fmt.Sprintf(`UPSUN_CLI_TOKEN=%s upsun mount:download --yes --quiet --project="%s" --environment="%s" --mount=web/sites/default/files --target=%s && cat %s/%s && rm -rf %s`, token, upsunTestSiteID, upsunPushTestSiteEnvironment, tmpRsyncDir, tmpRsyncDir, fName, tmpRsyncDir),
	})
	require.NoError(t, err)
	assert.Contains(out, tval)

	err = app.MutagenSyncFlush()
	assert.NoError(err)

	assert.FileExists("hello-pre-push-" + app.Name)
	assert.FileExists("hello-post-push-" + app.Name)
	err = os.Remove("hello-pre-push-" + app.Name)
	assert.NoError(err)
	err = os.Remove("hello-post-push-" + app.Name)
	assert.NoError(err)
}
