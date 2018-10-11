package ddevapp_test

import (
	"os"
	"path/filepath"
	"testing"

	"bufio"
	"fmt"
	"strings"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
)

// TestApptypeDetection does a simple test of various filesystem setups to make
// sure the expected apptype is returned.
func TestApptypeDetection(t *testing.T) {
	assert := asrt.New(t)

	fileLocations := map[string]string{
		ddevapp.AppTypeDrupal6:   "misc/ahah.js",
		ddevapp.AppTypeDrupal7:   "misc/ajax.js",
		ddevapp.AppTypeDrupal8:   "core/scripts/drupal.sh",
		ddevapp.AppTypeWordPress: "wp-settings.php",
		ddevapp.AppTypeBackdrop:  "core/scripts/backdrop.sh",
	}

	for expectedType, expectedPath := range fileLocations {
		testDir := testcommon.CreateTmpDir("TestApptype")

		// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
		defer testcommon.CleanupDir(testDir)
		defer testcommon.Chdir(testDir)()

		err := os.MkdirAll(filepath.Join(testDir, filepath.Dir(expectedPath)), 0777)
		assert.NoError(err)

		_, err = os.OpenFile(filepath.Join(testDir, expectedPath), os.O_RDONLY|os.O_CREATE, 0666)
		assert.NoError(err)

		app, err := ddevapp.NewApp(testDir, ddevapp.ProviderDefault)
		assert.NoError(err)

		foundType := app.DetectAppType()
		assert.EqualValues(expectedType, foundType)
	}
}

// TestPostConfigAction tests that the post-config action is properly applied, but only if the
// config is not included in the config.yaml.
func TestPostConfigAction(t *testing.T) {
	assert := asrt.New(t)

	appTypes := map[string]string{
		ddevapp.AppTypeDrupal6:   ddevapp.PHP56,
		ddevapp.AppTypeDrupal7:   ddevapp.PHP71,
		ddevapp.AppTypeDrupal8:   ddevapp.PHPDefault,
		ddevapp.AppTypeWordPress: ddevapp.PHPDefault,
		ddevapp.AppTypeBackdrop:  ddevapp.PHPDefault,
	}

	for appType, expectedPHPVersion := range appTypes {
		testDir := testcommon.CreateTmpDir("TestApptype")

		// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
		defer testcommon.CleanupDir(testDir)
		defer testcommon.Chdir(testDir)()

		app, err := ddevapp.NewApp(testDir, ddevapp.ProviderDefault)
		assert.NoError(err)

		// Prompt for apptype as a way to get it into the config.
		input := fmt.Sprintf(appType + "\n")
		scanner := bufio.NewScanner(strings.NewReader(input))
		util.SetInputScanner(scanner)
		err = app.AppTypePrompt()
		assert.NoError(err)
		fmt.Println("")

		// With no config file written, the ConfigFileOverrideAction should result in an override
		err = app.ConfigFileOverrideAction()
		assert.NoError(err)

		// With a basic new app, the expectedPHPVersion should be the default
		assert.EqualValues(app.PHPVersion, expectedPHPVersion)

		newVersion := "19.0-" + appType
		app.PHPVersion = newVersion
		err = app.WriteConfig()
		assert.NoError(err)
		err = app.ConfigFileOverrideAction()
		assert.NoError(err)
		// But with a config that has been written with a specified version, the version should be untouched by
		// app.ConfigFileOverrideAction()
		assert.EqualValues(app.PHPVersion, newVersion)
	}

}
