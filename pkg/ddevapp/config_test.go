package ddevapp

import (
	"testing"

	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/version"
	"github.com/stretchr/testify/assert"
)

// TestNewConfigNoLoad tests the functionality that is called when "ddev start" is executed
func TestNewConfig(t *testing.T) {
	assert := assert.New(t)
	// Create a temporary directory and change to it for the duration of this test.
	testDir := testcommon.CreateTmpDir()
	defer testcommon.Chdir(testDir)()

	// Load a new Config
	newConfig, err := NewConfig(testDir)

	// An error should be returned because no config file is present.
	assert.Error(err)

	// Ensure the config uses specified defaults.
	assert.Equal(newConfig.APIVersion, CurrentAppVersion)
	assert.Equal(newConfig.Platform, DDevDefaultPlatform)
	assert.Equal(newConfig.DBImage, version.DBImg+":"+version.DBTag)
	assert.Equal(newConfig.WebImage, version.WebImg+":"+version.WebTag)
	newConfig.Name = testcommon.RandString(32)
	newConfig.AppType = "drupal8"

	// Write the newConfig.
	err = newConfig.Write()
	assert.NoError(err)

	loadedConfig, err := NewConfig(testDir)
	// There should be no error this time, since the config should be available for loading.
	assert.NoError(err)
	assert.Equal(newConfig.Name, loadedConfig.Name)
	assert.Equal(newConfig.AppType, loadedConfig.AppType)
}
