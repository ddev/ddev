package ddevapp_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/**
 * A valid site (with backups) must be present which matches the test site and environment name
 * defined in the constants below.
 */
const acquiaPullTestEnvironment = "ddevdemo.dev"
const acquiaPushTestEnvironment = "ddevdemo.test"

const acquiaPullSiteURL = "http://ddevdemodev.prod.acquia-sites.com/"
const acquiaSiteExpectation = "Super easy vegetarian pasta"

// Note that these tests won't run with GitHub actions on a forked PR.
// This is a security feature, but means that PRs intended to test this
// must be done in the DDEV repo.

// TestAcquiaPull ensures we can pull backups from Acquia
func TestAcquiaPull(t *testing.T) {
	acquiaKey := ""
	acquiaSecret := ""
	sshkey := ""
	if acquiaKey = os.Getenv("DDEV_ACQUIA_API_KEY"); acquiaKey == "" {
		t.Skipf("No DDEV_ACQUIA_API_KEY env var has been set. Skipping %v", t.Name())
	}
	if acquiaSecret = os.Getenv("DDEV_ACQUIA_API_SECRET"); acquiaSecret == "" {
		t.Skipf("No DDEV_ACQUIA_SECRET env var has been set. Skipping %v", t.Name())
	}
	if sshkey = os.Getenv("DDEV_ACQUIA_SSH_KEY"); sshkey == "" {
		t.Skipf("No DDEV_ACQUIA_SSH_KEY env var has been set. Skipping %v", t.Name())
	}

	require.True(t, isPullSiteValid(acquiaPullSiteURL, acquiaSiteExpectation), "acquiaPullSiteURL %s isn't working right", acquiaPullSiteURL)
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	origDir, _ := os.Getwd()

	webEnvSave := globalconfig.DdevGlobalConfig.WebEnvironment

	globalconfig.DdevGlobalConfig.WebEnvironment = []string{"ACQUIA_API_KEY=" + acquiaKey, "ACQUIA_API_SECRET=" + acquiaSecret, "ACQUIA_ENVIRONMENT_ID=" + acquiaPullTestEnvironment}
	err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	assert.NoError(err)

	// Use a Drupal 10 codebase (test CMS 12)
	drupalCode := FullTestSites[12]
	drupalCode.Name = t.Name()
	err = globalconfig.RemoveProjectInfo(t.Name())
	require.NoError(t, err)
	err = drupalCode.Prepare()
	require.NoError(t, err)
	app, err := ddevapp.NewApp(drupalCode.Dir, false)
	require.NoError(t, err)
	_ = app.Stop(true, false)
	err = os.Chdir(drupalCode.Dir)
	require.NoError(t, err)
	// acli really wants the project to look like the target project
	app.Docroot = "docroot"
	app.Database = ddevapp.DatabaseDesc{
		Type:    nodeps.MySQL,
		Version: nodeps.MySQL57,
	}

	err = setupSSHKey(t, sshkey, filepath.Join(origDir, "testdata", t.Name()))
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)

		globalconfig.DdevGlobalConfig.WebEnvironment = webEnvSave
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		assert.NoError(err)

		_ = os.Chdir(origDir)
	})

	app.Name = t.Name()
	app.Type = nodeps.AppTypeDrupal

	_ = app.Stop(true, false)
	err = app.WriteConfig()
	assert.NoError(err)

	testcommon.ClearDockerEnv()

	err = ddevapp.PopulateExamplesCommandsHomeadditions(app.Name)
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	err = app.WriteConfig()
	require.NoError(t, err)
	err = app.MutagenSyncFlush()
	require.NoError(t, err)

	provider, err := app.GetProvider("acquia")
	require.NoError(t, err)
	err = app.Pull(provider, false, false, false)
	require.NoError(t, err)

	assert.FileExists(filepath.Join(app.GetHostUploadDirFullPath(), "chocolate-brownie-umami.jpg"))
	out, err := exec.RunCommand("bash", []string{"-c", fmt.Sprintf(`echo 'select COUNT(*) from users_field_data where mail="randy@example.com";' | %s mysql -B --skip-column-names `, DdevBin)})
	assert.NoError(err)
	assert.True(strings.HasSuffix(out, "\n1\n"), "out is unexpected '%s'", out)
}

// TestAcquiaPush ensures we can push to acquia for a configured environment.
func TestAcquiaPush(t *testing.T) {
	acquiaKey := ""
	acquiaSecret := ""
	sshkey := ""
	if os.Getenv("DDEV_ALLOW_ACQUIA_PUSH") != "true" {
		t.Skip("TestAcquiaPush is currently embargoed by DDEV_ALLOW_ACQUIA_PUSH not set to true")
	}
	if acquiaKey = os.Getenv("DDEV_ACQUIA_API_KEY"); acquiaKey == "" {
		t.Skipf("No DDEV_ACQUIA_API_KEY env var has been set. Skipping %v", t.Name())
	}
	if acquiaSecret = os.Getenv("DDEV_ACQUIA_API_SECRET"); acquiaSecret == "" {
		t.Skipf("No DDEV_ACQUIA_API_SECRET env var has been set. Skipping %v", t.Name())
	}
	if sshkey = os.Getenv("DDEV_ACQUIA_SSH_KEY"); sshkey == "" {
		t.Skipf("No DDEV_ACQUIA_SSH_KEY env var has been set. Skipping %v", t.Name())
	}

	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	origDir, _ := os.Getwd()

	webEnvSave := globalconfig.DdevGlobalConfig.WebEnvironment

	globalconfig.DdevGlobalConfig.WebEnvironment = []string{"ACQUIA_API_KEY=" + acquiaKey, "ACQUIA_API_SECRET=" + acquiaSecret, "ACQUIA_ENVIRONMENT_ID=" + acquiaPushTestEnvironment}
	err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	assert.NoError(err)

	// Use a Drupal 10 codebase (test CMS 12)
	drupalCode := FullTestSites[12]
	drupalCode.Name = t.Name()
	err = globalconfig.RemoveProjectInfo(t.Name())
	require.NoError(t, err)
	err = drupalCode.Prepare()
	require.NoError(t, err)
	app, err := ddevapp.NewApp(drupalCode.Dir, false)
	require.NoError(t, err)
	_ = app.Stop(true, false)
	err = os.Chdir(drupalCode.Dir)
	require.NoError(t, err)
	// acli really wants the project to look like the target project
	app.Docroot = "docroot"
	app.Database = ddevapp.DatabaseDesc{
		Type:    nodeps.MySQL,
		Version: nodeps.MySQL57,
	}

	err = setupSSHKey(t, sshkey, filepath.Join(origDir, "testdata", t.Name()))
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)

		globalconfig.DdevGlobalConfig.WebEnvironment = webEnvSave
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		assert.NoError(err)

		_ = os.Chdir(origDir)
		_ = os.RemoveAll(drupalCode.Dir)
	})

	app.Hooks = map[string][]ddevapp.YAMLTask{"post-push": {{"exec-host": "touch hello-post-push-" + app.Name}}, "pre-push": {{"exec-host": "touch hello-pre-push-" + app.Name}}}
	_ = app.Stop(true, false)

	err = app.WriteConfig()
	require.NoError(t, err)

	testcommon.ClearDockerEnv()

	err = ddevapp.PopulateExamplesCommandsHomeadditions(app.Name)
	require.NoError(t, err)

	// Create the uploaddir and a file; it won't have existed in our download
	tval := nodeps.RandomString(10)
	fName := tval + ".txt"
	fContent := []byte(tval)
	err = os.MkdirAll(filepath.Join(app.AppRoot, app.Docroot, "sites/default/files"), 0777)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(app.AppRoot, app.Docroot, "sites/default/files", fName), fContent, 0644)
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	// Since allow-plugins isn't there and you can't even set it with Composer
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: `composer config --no-plugins allow-plugins true`,
	})
	require.NoError(t, err)

	// Create database and files entries that we can verify after push
	writeQuery := fmt.Sprintf(`mysql -e 'CREATE TABLE IF NOT EXISTS %s ( title VARCHAR(255) NOT NULL ); INSERT INTO %s VALUES("%s");'`, t.Name(), t.Name(), tval)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: writeQuery,
	})
	require.NoError(t, err)

	// Make sure that the file we created exists in the container
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: fmt.Sprintf("ls %s", path.Join("/var/www/html", app.Docroot, "sites/default/files", fName)),
	})
	require.NoError(t, err)

	err = app.WriteConfig()
	require.NoError(t, err)

	provider, err := app.GetProvider("acquia")
	require.NoError(t, err)

	err = app.Push(provider, false, false)
	require.NoError(t, err)

	// Test that the database row was added
	readQuery := fmt.Sprintf(`acli -n ssh %s 'echo "SELECT title FROM %s WHERE title=\"%s\";" | docroot/vendor/bin/drush  sql-cli --extra=-N'`, acquiaPushTestEnvironment, t.Name(), tval)
	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: readQuery,
	})
	require.NoError(t, err)
	assert.Contains(out, tval)

	// Remove the file we pushed to we know it's gone
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: fmt.Sprintf(`rm docroot/sites/default/files/%s`, fName),
	})
	// Pull the files back using acli
	require.NoError(t, err)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: fmt.Sprintf(`acli pull:files %s`, acquiaPushTestEnvironment),
	})
	require.NoError(t, err)

	// Test that the file arrived back with the pull
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: fmt.Sprintf("ls docroot/sites/default/files/%s", fName),
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

// isPullSiteValid checks to make sure the site we're testing against is OK
func isPullSiteValid(siteURL string, siteExpectation string) bool {
	resp, err := http.Get(siteURL)
	if err != nil {
		return false
	}
	//nolint: errcheck
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	return strings.Contains(string(body), siteExpectation)
}
