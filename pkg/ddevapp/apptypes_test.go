package ddevapp_test

import (
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"

	"bufio"
	"fmt"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
)

// TestApptypeDetection does a simple test of various filesystem setups to make
// sure the expected apptype is returned.
func TestApptypeDetection(t *testing.T) {
	assert := asrt.New(t)
	origDir, _ := os.Getwd()
	appTypes := ddevapp.GetValidAppTypes()
	var notSimplePHPAppTypes = []string{}
	for _, t := range appTypes {
		if t != nodeps.AppTypePHP {
			notSimplePHPAppTypes = append(notSimplePHPAppTypes, t)
		}
	}
	tmpDir := testcommon.CreateTmpDir(t.Name())

	t.Cleanup(func() {
		err := os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(tmpDir)
		assert.NoError(err)
	})

	err := fileutil.CopyDir(filepath.Join(origDir, "testdata", t.Name()), filepath.Join(tmpDir, "sampleapptypes"))
	require.NoError(t, err)
	for _, appType := range notSimplePHPAppTypes {
		app, err := ddevapp.NewApp(filepath.Join(tmpDir, "sampleapptypes", appType), true)
		assert.NoError(err)
		app.Docroot = ddevapp.DiscoverDefaultDocroot(app)
		t.Cleanup(func() {
			err = app.Stop(true, false)
			assert.NoError(err)
		})

		foundType := app.DetectAppType()
		assert.EqualValues(appType, foundType)
	}
}

// TestPostConfigAction tests that the post-config action is properly applied, but only if the
// config is not included in the config.yaml.
func TestPostConfigAction(t *testing.T) {
	assert := asrt.New(t)
	origDir, _ := os.Getwd()

	appTypes := map[string]string{
		nodeps.AppTypeBackdrop:     nodeps.PHPDefault,
		nodeps.AppTypeCraftCms:     nodeps.PHPDefault,
		nodeps.AppTypeDrupal6:      nodeps.PHP56,
		nodeps.AppTypeDrupal7:      nodeps.PHPDefault,
		nodeps.AppTypeDrupal8:      nodeps.PHP74,
		nodeps.AppTypeDrupal9:      nodeps.PHPDefault,
		nodeps.AppTypeDrupal10:     nodeps.PHP81,
		nodeps.AppTypeLaravel:      nodeps.PHP81,
		nodeps.AppTypeMagento:      nodeps.PHP74,
		nodeps.AppTypeMagento2:     nodeps.PHP81,
		nodeps.AppTypeWordPress:    nodeps.PHPDefault,
		nodeps.AppTypeSilverstripe: nodeps.PHP81,
	}

	for appType, expectedPHPVersion := range appTypes {
		testDir := testcommon.CreateTmpDir(t.Name())

		app, err := ddevapp.NewApp(testDir, true)
		assert.NoError(err)

		t.Cleanup(func() {
			err = os.Chdir(origDir)
			assert.NoError(err)
			err = app.Stop(true, false)
			assert.NoError(err)
			err = os.RemoveAll(testDir)
			assert.NoError(err)
		})

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
		assert.EqualValues(expectedPHPVersion, app.PHPVersion, "expected PHP version %s but got %s for apptype=%s", expectedPHPVersion, app.PHPVersion, appType)

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
