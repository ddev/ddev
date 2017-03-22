package ddevapp

import (
	"os"
	"path/filepath"
	"testing"

	"io/ioutil"

	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/version"
	"github.com/stretchr/testify/assert"
)

// TestNewConfig tests functionality around creating a new config, writing it to disk, and reading the resulting config.
func TestNewConfig(t *testing.T) {
	assert := assert.New(t)
	// Create a temporary directory and change to it for the duration of this test.
	testDir := testcommon.CreateTmpDir()
	defer testcommon.Chdir(testDir)()
	defer testcommon.CleanupDir(testDir)

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
	_, err = os.Stat(newConfig.ConfigPath)
	assert.NoError(err)

	loadedConfig, err := NewConfig(testDir)
	// There should be no error this time, since the config should be available for loading.
	assert.NoError(err)
	assert.Equal(newConfig.Name, loadedConfig.Name)
	assert.Equal(newConfig.AppType, loadedConfig.AppType)
}

// TestAllowedAppType tests the isAllowedAppType function.
func TestAllowedAppTypes(t *testing.T) {
	assert := assert.New(t)
	for _, v := range allowedAppTypes {
		assert.True(isAllowedAppType(v))
	}

	for i := 1; i <= 50; i++ {
		randomType := testcommon.RandString(32)
		assert.False(isAllowedAppType(randomType))
	}
}

// TestPrepDirectory ensures the configuraion directory can be created with the correct permissions.
func TestPrepDirectory(t *testing.T) {
	assert := assert.New(t)
	// Create a temporary directory and change to it for the duration of this test.
	testDir := testcommon.CreateTmpDir()
	defer testcommon.Chdir(testDir)()
	defer testcommon.CleanupDir(testDir)

	config, err := NewConfig(testDir)
	// We should get an error here, since no config exists.
	assert.Error(err)

	// Prep the directory.
	err = prepDDevDirectory(filepath.Dir(config.ConfigPath))
	assert.NoError(err)

	// Read directory info an ensure it exists.
	dirinfo, err := os.Stat(filepath.Dir(config.ConfigPath))
	assert.NoError(err)
	assert.Equal(dirinfo.Mode(), 0644)
}

// TestHostName tests that the TestSite.Hostname() field returns the hostname as expected.
func TestHostName(t *testing.T) {
	assert := assert.New(t)
	testDir := testcommon.CreateTmpDir()
	defer testcommon.Chdir(testDir)()
	defer testcommon.CleanupDir(testDir)
	config, err := NewConfig(testDir)
	assert.Error(err)
	config.Name = testcommon.RandString(32)

	assert.Equal(config.Hostname(), config.Name+"."+DDevTLD)
}

// TestWriteDockerComposeYaml tests the writing of a docker-compose.yaml file.
func TestWriteDockerComposeYaml(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := assert.New(t)
	testDir := testcommon.CreateTmpDir()
	defer testcommon.Chdir(testDir)()
	defer testcommon.CleanupDir(testDir)

	// Create a config
	config, err := NewConfig(testDir)
	assert.Error(err)
	config.Name = testcommon.RandString(32)
	config.AppType = allowedAppTypes[0]
	config.Docroot = testcommon.RandString(16)

	err = config.WriteDockerComposeConfig()
	// We should get an error here since no config or directory path exists.
	assert.Error(err)

	// Write a config to create/prep necessary directories.
	err = config.Write()
	assert.NoError(err)

	// After the config has been written and directories exist, the write should work.
	err = config.WriteDockerComposeConfig()
	assert.NoError(err)

	// Ensure we can read from the file and that it's a regular file with the expected name.
	fileinfo, err := os.Stat(config.DockerComposeYAMLPath())
	assert.NoError(err)
	assert.False(fileinfo.IsDir())
	assert.Equal(fileinfo.Name(), filepath.Base(config.DockerComposeYAMLPath()))

	composeBytes, err := ioutil.ReadFile(config.DockerComposeYAMLPath())
	assert.NoError(err)
	contentString := string(composeBytes)
	assert.Contains(contentString, config.Docroot)
	assert.Contains(contentString, config.Name)
	assert.Contains(contentString, config.Platform)
	assert.Contains(contentString, config.AppType)
}
