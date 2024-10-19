package ddevapp_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/**
 * These tests rely on an external test account.
 */

const lagoonProjectName = "amazeeio-ddev"

// TODO: Change to use the  "pull" environment when we have one
const lagoonPullTestSiteEnvironment = "pull"
const lagoonPushTestSiteEnvironment = "push"

// TODO: Change this to the actual dedicated pull environment
const lagoonPullSiteURL = "https://nginx.pull.amazeeio-ddev.us2.amazee.io/"
const lagoonSiteExpectation = "Super easy vegetarian pasta"

// These tests won't run with GitHub actions on a forked PR.

func lagoonSetupSSHKey(t *testing.T) string {
	sshkey := ""
	if sshkey = os.Getenv("DDEV_LAGOON_SSH_KEY"); sshkey == "" {
		t.Skipf("No DDEV_LAGOON_SSH_KEY env var has been set. Skipping %v", t.Name())
	}
	return sshkey + "\n"
}

// TestLagoonPull ensures we can pull from lagoon
func TestLagoonPull(t *testing.T) {
	assert := asrt.New(t)
	var err error

	sshKey := lagoonSetupSSHKey(t)

	require.True(t, isPullSiteValid(lagoonPullSiteURL, lagoonSiteExpectation), "lagoonPullSiteURL %s isn't working right", lagoonPullSiteURL)

	origDir, _ := os.Getwd()

	siteDir := testcommon.CreateTmpDir(t.Name())

	err = globalconfig.RemoveProjectInfo(t.Name())
	require.NoError(t, err)

	err = os.Chdir(siteDir)
	assert.NoError(err)
	app, err := ddevapp.NewApp(siteDir, true)
	assert.NoError(err)
	app.Name = t.Name()
	app.Type = nodeps.AppTypeDrupal
	err = app.Stop(true, false)
	require.NoError(t, err)
	err = app.WriteConfig()
	require.NoError(t, err)

	testcommon.ClearDockerEnv()

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)

		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(siteDir)
	})

	err = ddevapp.PopulateExamplesCommandsHomeadditions(app.Name)
	require.NoError(t, err)

	app.Docroot = "web"
	app.Database = ddevapp.DatabaseDesc{
		Type:    nodeps.MySQL,
		Version: nodeps.MySQL57,
	}

	err = app.WriteConfig()
	require.NoError(t, err)

	provider, err := app.GetProvider("lagoon")
	require.NoError(t, err)

	app.WebEnvironment = []string{"LAGOON_PROJECT=" + lagoonProjectName, "LAGOON_ENVIRONMENT=" + lagoonPullTestSiteEnvironment}

	err = setupSSHKey(t, sshKey, filepath.Join(origDir, "testdata", t.Name()))
	require.NoError(t, err)

	err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), ".lagoon.yml"), filepath.Join(app.AppRoot, ".lagoon.yml"))
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)
	err = app.Pull(provider, false, false, false)
	require.NoError(t, err)

	assert.FileExists(filepath.Join(app.GetHostUploadDirFullPath(), "victoria-sponge-umami.jpg"))
	out, err := exec.RunHostCommand("bash", "-c", fmt.Sprintf(`echo 'select COUNT(*) from users_field_data where mail="margaret.hopper@example.com";' | %s mysql -N`, DdevBin))
	assert.NoError(err)
	assert.True(strings.HasSuffix(out, "\n1\n"))
}

// TestLagoonPush ensures we can push to lagoon for a configured environment.
func TestLagoonPush(t *testing.T) {
	assert := asrt.New(t)
	origDir, _ := os.Getwd()

	sshKey := lagoonSetupSSHKey(t)

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

		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(siteDir)
	})

	app.Name = t.Name()
	app.Type = nodeps.AppTypeDrupal
	_ = app.Stop(true, false)

	app.Docroot = "web"
	app.Database = ddevapp.DatabaseDesc{
		Type:    nodeps.MySQL,
		Version: nodeps.MySQL57,
	}

	err = app.WriteConfig()
	require.NoError(t, err)

	testcommon.ClearDockerEnv()

	err = ddevapp.PopulateExamplesCommandsHomeadditions(app.Name)
	require.NoError(t, err)

	provider, err := app.GetProvider("lagoon")
	require.NoError(t, err)

	provider.EnvironmentVariables["LAGOON_PROJECT"] = lagoonProjectName
	provider.EnvironmentVariables["LAGOON_ENVIRONMENT"] = lagoonPushTestSiteEnvironment

	err = setupSSHKey(t, sshKey, filepath.Join(origDir, "testdata", t.Name()))
	require.NoError(t, err)

	err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), ".lagoon.yml"), filepath.Join(app.AppRoot, ".lagoon.yml"))
	require.NoError(t, err)

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
	c := fmt.Sprintf(`echo 'SELECT title FROM %s WHERE title="%s";' | lagoon ssh --strict-host-key-checking no -p %s -e %s -C 'mysql --host=$MARIADB_HOST --user=$MARIADB_USERNAME --password=$MARIADB_PASSWORD --database=$MARIADB_DATABASE'`, t.Name(), tval, lagoonProjectName, lagoonPushTestSiteEnvironment)
	//t.Logf("attempting command '%s'", c)
	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: c,
	})
	assert.NoError(err)
	assert.Contains(out, tval)

	// Test that the file arrived there
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: fmt.Sprintf(`lagoon ssh --strict-host-key-checking no -p %s -e %s -C 'ls -l /app/web/sites/default/files/%s'`, lagoonProjectName, lagoonPushTestSiteEnvironment, fName),
	})
	assert.NoError(err)
	assert.Contains(out, tval)

	err = app.MutagenSyncFlush()
	assert.NoError(err)
}
