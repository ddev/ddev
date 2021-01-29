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
 * need to set an environment variable called "DDEV_platform_API_TOKEN" with credentials for
 * this account. If no such environment variable is present, these tests will be skipped.
 *
 * A valid site (with backups) must be present which matches the test site and environment name
 * defined in the constants below.
 */

var platformTestSiteID = "lago3j23xu2w6"

// TestPlatformPull ensures we can pull backups from platform.sh for a configured environment.
func TestPlatformPull(t *testing.T) {
	var token string
	if token = os.Getenv("DDEV_PLATFORM_API_TOKEN"); token == "" {
		t.Skipf("No DDEV_PLATFORM_API_TOKEN env var has been set. Skipping %v", t.Name())
	}
	assert := asrt.New(t)
	var err error

	testDir, _ := os.Getwd()

	siteDir := testcommon.CreateTmpDir(t.Name())

	err = os.Chdir(siteDir)
	assert.NoError(err)
	app, err := NewApp(siteDir, true, "")
	assert.NoError(err)
	app.Name = t.Name()
	app.Type = nodeps.AppTypeDrupal9
	err = app.Stop(true, false)
	require.NoError(t, err)
	err = app.WriteConfig()
	require.NoError(t, err)

	globalconfig.DdevGlobalConfig.WebEnvironment = []string{"PLATFORMSH_CLI_TOKEN=" + token}
	err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	assert.NoError(err)

	testcommon.ClearDockerEnv()

	t.Cleanup(func() {
		globalconfig.DdevGlobalConfig.WebEnvironment = []string{}
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		assert.NoError(err)

		err = app.Stop(true, false)
		assert.NoError(err)

		_ = os.Chdir(testDir)
		err = os.RemoveAll(siteDir)
		assert.NoError(err)
	})

	_, err = exec.RunCommand("bash", []string{"-c", fmt.Sprintf("%s >/dev/null", DdevBin)})
	require.NoError(t, err)

	// Build our platform.yaml from the example file
	s, err := ioutil.ReadFile(app.GetConfigPath("providers/platform.yaml.example"))
	require.NoError(t, err)
	x := strings.Replace(string(s), "project_id:", "#project_id:", 1)
	x = x + "\nproject_id: " + platformTestSiteID + "\n"
	err = ioutil.WriteFile(app.GetConfigPath("providers/platform.yaml"), []byte(x), 0666)
	assert.NoError(err)
	app.Provider = "platform"
	err = app.WriteConfig()
	require.NoError(t, err)

	provider, err := app.GetProvider()
	require.NoError(t, err)
	err = app.Start()
	require.NoError(t, err)
	err = app.Pull(provider, &PullOptions{})
	assert.NoError(err)

	assert.FileExists(filepath.Join(app.GetUploadDir(), "victoria-sponge-umami.jpg"))
	out, err := exec.RunCommand("bash", []string{"-c", fmt.Sprintf(`echo 'select COUNT(*) from users_field_data where mail="margaret.hopper@example.com";' | %s mysql -N`, DdevBin)})
	assert.NoError(err)
	assert.True(strings.HasPrefix(out, "1\n"))
}
