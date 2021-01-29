package ddevapp_test

import (
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"os"
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

var platformTestSiteID = "w5vxjqzsumvoq"
var platformTestEnvName = "master"

// TestPlatformPull ensures we can pull backups from platform.sh for a configured environment.
func TestPlatformPull(t *testing.T) {
	var token = ""
	if token = os.Getenv("DDEV_PLATFORM_API_TOKEN"); token == "" {
		t.Skipf("No DDEV_PLATFORM_API_TOKEN env var has been set. Skipping %v", t.Name())
	}
	assert := asrt.New(t)
	var err error

	testDir, _ := os.Getwd()

	siteDir := testcommon.CreateTmpDir(t.Name())

	err = os.Chdir(siteDir)
	assert.NoError(err)
	app, err := NewApp(siteDir, true, "platform")
	assert.NoError(err)
	app.Name = t.Name()
	app.Type = nodeps.AppTypePHP
	err = app.WriteConfig()
	assert.NoError(err)

	globalconfig.DdevGlobalConfig.WebEnvironment = []string{"PLATFORM_CLI_TOKEN=" + token}
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

	err = fileutil.CopyFile(app.GetConfigPath("providers/platform.yaml.example"), app.GetConfigPath("providers/platform.yaml"))
	s, err := os.ReadFile(app.GetConfigPath("providers/platform.yaml"))
	assert.NoError(err)
	x := strings.Replace("project_id:", "project_id_backup:", string(s), 1)
	x = x + "\nproject_id: " + platformTestSiteID + "\n"
	err = os.WriteFile(app.GetConfigPath("providers/platform.yaml"), []byte(x), 0666)
	assert.NoError(err)

	err = app.Pull(app.ProviderInstance, &PullOptions{})
	assert.NoError(err)
}
