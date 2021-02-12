package ddevapp_test

import (
	"fmt"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/testcommon"
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
const acquiaTestSite = "eeamoreno.dev"

// TestAcquiaPull ensures we can pull backups from Acquia
func TestAcquiaPull(t *testing.T) {
	acquiaKey := ""
	acquiaSecret := ""
	sshkey := ""
	if acquiaKey = os.Getenv("DDEV_ACQUIA_API_KEY"); acquiaKey == "" {
		t.Skipf("No DDEV_ACQUIA_KEY env var has been set. Skipping %v", t.Name())
	}
	if acquiaSecret = os.Getenv("DDEV_ACQUIA_API_SECRET"); acquiaSecret == "" {
		t.Skipf("No DDEV_ACQUIA_SECRET env var has been set. Skipping %v", t.Name())
	}
	if sshkey = os.Getenv("DDEV_ACQUIA_SSH_KEY"); sshkey == "" {
		t.Skipf("No DDEV_ACQUIA_SSH_KEY env var has been set. Skipping %v", t.Name())
	}
	sshkey = strings.Replace(sshkey, "<SPLIT>", "\n", -1)

	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testDir, _ := os.Getwd()

	webEnvSave := globalconfig.DdevGlobalConfig.WebEnvironment

	globalconfig.DdevGlobalConfig.WebEnvironment = []string{"ACQUIA_API_KEY=" + acquiaKey, "ACQUIA_API_SECRET=" + acquiaSecret}
	err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	assert.NoError(err)

	siteDir := testcommon.CreateTmpDir(t.Name())
	err = os.MkdirAll(filepath.Join(siteDir, "docroot/sites/default"), 0777)
	assert.NoError(err)
	err = os.Chdir(siteDir)
	assert.NoError(err)

	// Provide an ssh key for `ddev auth ssh`
	err = os.Mkdir("sshtest", 0755)
	assert.NoError(err)
	err = ioutil.WriteFile(filepath.Join("sshtest", "id_rsa_test"), []byte(sshkey), 0600)
	assert.NoError(err)
	out, err := exec.RunCommand("expect", []string{filepath.Join(testDir, "testdata", t.Name(), "ddevauthssh.expect"), DdevBin, "./sshtest"})
	assert.NoError(err)
	assert.Contains(string(out), "Identity added:")

	app, err := NewApp(siteDir, true)
	assert.NoError(err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)

		globalconfig.DdevGlobalConfig.WebEnvironment = webEnvSave
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		assert.NoError(err)

		_ = os.Chdir(testDir)
		err = os.RemoveAll(siteDir)
		assert.NoError(err)
	})

	app.Name = t.Name()
	app.Type = nodeps.AppTypeDrupal9
	app.ComposerVersion = "2"

	err = app.WriteConfig()
	assert.NoError(err)

	testcommon.ClearDockerEnv()

	// Run ddev once to create all the files in .ddev, including the example
	_, err = exec.RunCommand("bash", []string{"-c", fmt.Sprintf("%s >/dev/null", DdevBin)})
	require.NoError(t, err)

	_, err = exec.RunCommand("bash", []string{"-c", fmt.Sprintf("%s composer require drush/drush >/dev/null 2>&1", DdevBin)})
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	// Build our acquia.yaml from the example file
	s, err := ioutil.ReadFile(app.GetConfigPath("providers/acquia.yaml.example"))
	require.NoError(t, err)
	x := strings.Replace(string(s), "project_id:", fmt.Sprintf("project_id: %s\n#project_id:", acquiaTestSite), -1)
	err = ioutil.WriteFile(app.GetConfigPath("providers/acquia.yaml"), []byte(x), 0666)
	assert.NoError(err)
	err = app.WriteConfig()
	require.NoError(t, err)

	provider, err := app.GetProvider("acquia")
	require.NoError(t, err)
	err = app.Pull(provider, false, false, false)
	assert.NoError(err)

	assert.FileExists(filepath.Join(app.GetUploadDir(), "chocolate-brownie-umami.jpg"))
	out, err = exec.RunCommand("bash", []string{"-c", fmt.Sprintf(`echo 'select COUNT(*) from users_field_data where mail="randy@example.com";' | %s mysql -N`, DdevBin)})
	assert.NoError(err)
	assert.True(strings.HasPrefix(out, "1\n"))

}
