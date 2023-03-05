package ddevapp_test

import (
	"fmt"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
)

/**
 * A valid site (with backups) must be present which matches the test site and environment name
 * defined in the constants below.
 */
const acquiaPullTestSite = "ddevdemo.dev"
const acquiaPullDatabase = "ddevdemo"
const acquiaPushTestSite = "ddevdemo.test"

const acquiaPullSiteURL = "http://ddevdemodev.prod.acquia-sites.com/"
const acquiaSiteExpectation = "Super easy vegetarian pasta"

// Note that these tests won't run with GitHub actions on a forked PR.
// Thie is a security feature, but means that PRs intended to test this
// must be done in the ddev repo.

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
	sshkey = strings.Replace(sshkey, "<SPLIT>", "\n", -1)

	require.True(t, isPullSiteValid(acquiaPullSiteURL, acquiaSiteExpectation), "acquiaPullSiteURL %s isn't working right", acquiaPullSiteURL)
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	origDir, _ := os.Getwd()

	webEnvSave := globalconfig.DdevGlobalConfig.WebEnvironment

	globalconfig.DdevGlobalConfig.WebEnvironment = []string{"ACQUIA_API_KEY=" + acquiaKey, "ACQUIA_API_SECRET=" + acquiaSecret}
	err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	assert.NoError(err)

	siteDir := testcommon.CreateTmpDir(t.Name())
	err = os.MkdirAll(filepath.Join(siteDir, "docroot/sites/default"), 0777)
	assert.NoError(err)
	err = os.Chdir(siteDir)
	assert.NoError(err)

	err = setupSSHKey(t, sshkey, filepath.Join(origDir, "testdata", t.Name()))
	require.NoError(t, err)

	app, err := NewApp(siteDir, true)
	assert.NoError(err)
	app.PHPVersion = "8.0"

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)

		globalconfig.DdevGlobalConfig.WebEnvironment = webEnvSave
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		assert.NoError(err)

		_ = os.Chdir(origDir)
		err = os.RemoveAll(siteDir)
		assert.NoError(err)
	})

	app.Name = t.Name()
	app.Type = nodeps.AppTypeDrupal9

	_ = app.Stop(true, false)
	err = app.WriteConfig()
	assert.NoError(err)

	testcommon.ClearDockerEnv()

	err = PopulateExamplesCommandsHomeadditions(app.Name)
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	// Make sure we have drush
	_, _, err = app.Exec(&ExecOpts{
		Cmd: "composer require --no-interaction drush/drush symfony/http-kernel>/dev/null 2>/dev/null",
	})
	require.NoError(t, err)

	// Build our acquia.yaml from the example file
	s, err := os.ReadFile(app.GetConfigPath("providers/acquia.yaml.example"))
	require.NoError(t, err)

	// Replace the project_id and database_name
	x := strings.Replace(string(s), "project_id:", fmt.Sprintf("project_id: %s\n#project_id:", acquiaPullTestSite), -1)
	x = strings.Replace(x, "database_name: ", fmt.Sprintf("database_name: %s\n#database_name:", acquiaPullDatabase), -1)

	err = os.WriteFile(app.GetConfigPath("providers/acquia.yaml"), []byte(x), 0666)
	assert.NoError(err)
	err = app.WriteConfig()
	require.NoError(t, err)
	err = app.MutagenSyncFlush()
	require.NoError(t, err)

	provider, err := app.GetProvider("acquia")
	require.NoError(t, err)
	err = app.Pull(provider, false, false, false)
	require.NoError(t, err)

	assert.FileExists(filepath.Join(app.GetHostUploadDirFullPath(), "chocolate-brownie-umami.jpg"))
	out, err := exec.RunCommand("bash", []string{"-c", fmt.Sprintf(`echo 'select COUNT(*) from users_field_data where mail="randy@example.com";' | %s mysql -N`, DdevBin)})
	assert.NoError(err)
	assert.True(strings.HasPrefix(out, "1\n"))
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
	sshkey = strings.Replace(sshkey, "<SPLIT>", "\n", -1)

	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	origDir, _ := os.Getwd()

	webEnvSave := globalconfig.DdevGlobalConfig.WebEnvironment

	globalconfig.DdevGlobalConfig.WebEnvironment = []string{"ACQUIA_API_KEY=" + acquiaKey, "ACQUIA_API_SECRET=" + acquiaSecret}
	err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	assert.NoError(err)

	// Use a D9 codebase for drush to work right
	d9code := FullTestSites[8]
	d9code.Name = t.Name()
	err = globalconfig.RemoveProjectInfo(t.Name())
	require.NoError(t, err)
	err = d9code.Prepare()
	require.NoError(t, err)
	app, err := NewApp(d9code.Dir, false)
	require.NoError(t, err)
	_ = app.Stop(true, false)

	err = os.Chdir(d9code.Dir)
	require.NoError(t, err)

	err = setupSSHKey(t, sshkey, filepath.Join(origDir, "testdata", t.Name()))
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)

		globalconfig.DdevGlobalConfig.WebEnvironment = webEnvSave
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		assert.NoError(err)

		_ = os.Chdir(origDir)
		err = os.RemoveAll(d9code.Dir)
		assert.NoError(err)
	})

	app.Hooks = map[string][]YAMLTask{"post-push": {{"exec-host": "touch hello-post-push-" + app.Name}}, "pre-push": {{"exec-host": "touch hello-pre-push-" + app.Name}}}
	_ = app.Stop(true, false)

	err = app.WriteConfig()
	require.NoError(t, err)

	testcommon.ClearDockerEnv()

	err = PopulateExamplesCommandsHomeadditions(app.Name)
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

	// Since allow-plugins isn't there and you can't even set it with composer...
	_, _, err = app.Exec(&ExecOpts{
		Cmd: `composer config --no-plugins allow-plugins true`,
	})
	require.NoError(t, err)
	// Make sure we have drush
	_, _, err = app.Exec(&ExecOpts{
		Cmd: "composer require --no-interaction drush/drush >/dev/null 2>/dev/null",
	})
	require.NoError(t, err)

	_, _, err = app.Exec(&ExecOpts{
		Cmd: "time drush si -y minimal",
	})
	require.NoError(t, err)

	// Create database and files entries that we can verify after push
	writeQuery := fmt.Sprintf(`mysql -e 'CREATE TABLE IF NOT EXISTS %s ( title VARCHAR(255) NOT NULL ); INSERT INTO %s VALUES("%s");'`, t.Name(), t.Name(), tval)
	_, _, err = app.Exec(&ExecOpts{
		Cmd: writeQuery,
	})
	require.NoError(t, err)

	// Make sure that the file we created exists in the container
	_, _, err = app.Exec(&ExecOpts{
		Cmd: fmt.Sprintf("ls %s", path.Join("/var/www/html", app.Docroot, "sites/default/files", fName)),
	})
	require.NoError(t, err)

	// Build our PUSH acquia.yaml from the example file
	s, err := os.ReadFile(app.GetConfigPath("providers/acquia.yaml.example"))
	require.NoError(t, err)
	x := strings.Replace(string(s), "project_id:", fmt.Sprintf("project_id: %s\n#project_id:", acquiaPushTestSite), -1)
	err = os.WriteFile(app.GetConfigPath("providers/acquia.yaml"), []byte(x), 0666)
	assert.NoError(err)
	err = app.WriteConfig()
	require.NoError(t, err)

	provider, err := app.GetProvider("acquia")
	require.NoError(t, err)

	err = app.Push(provider, false, false)
	require.NoError(t, err)

	// Test that the database row was added
	readQuery := fmt.Sprintf(`echo 'SELECT title FROM %s WHERE title="%s"' | drush @%s --alias-path=~/.drush sql-cli --extra=-N`, t.Name(), tval, acquiaPushTestSite)
	out, _, err := app.Exec(&ExecOpts{
		Cmd: readQuery,
	})
	require.NoError(t, err)
	assert.Contains(out, tval)

	// Test that the file arrived there (by rsyncing it back)
	out, _, err = app.Exec(&ExecOpts{
		Cmd: fmt.Sprintf("drush --alias-path=~/.drush rsync -y @%s:%%files/%s /tmp && cat /tmp/%s", acquiaPushTestSite, fName, fName),
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

// isPullSiteValid just checks to make sure the site we're testing against is OK
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
