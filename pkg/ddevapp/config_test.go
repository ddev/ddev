package ddevapp_test

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"io/ioutil"

	. "github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/stretchr/testify/assert"
)

// TestNewConfig tests functionality around creating a new config, writing it to disk, and reading the resulting config.
func TestNewConfig(t *testing.T) {
	assert := assert.New(t)
	// Create a temporary directory and change to it for the duration of this test.
	testDir := testcommon.CreateTmpDir("TestNewConfig")

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
	assert.Equal(newConfig.DBAImage, version.DBAImg+":"+version.DBATag)
	newConfig.Name = util.RandString(32)
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

// TestAllowedAppType tests the IsAllowedAppType function.
func TestAllowedAppTypes(t *testing.T) {
	assert := assert.New(t)
	for _, v := range AllowedAppTypes {
		assert.True(IsAllowedAppType(v))
	}

	for i := 1; i <= 50; i++ {
		randomType := util.RandString(32)
		assert.False(IsAllowedAppType(randomType))
	}
}

// TestPrepDirectory ensures the configuration directory can be created with the correct permissions.
func TestPrepDirectory(t *testing.T) {
	assert := assert.New(t)
	// Create a temporary directory and change to it for the duration of this test.
	testDir := testcommon.CreateTmpDir("TestPrepDirectory")
	defer testcommon.Chdir(testDir)()
	defer testcommon.CleanupDir(testDir)

	config, err := NewConfig(testDir)
	// We should get an error here, since no config exists.
	assert.Error(err)

	// Prep the directory.
	err = PrepDdevDirectory(filepath.Dir(config.ConfigPath))
	assert.NoError(err)

	// Read directory info an ensure it exists.
	_, err = os.Stat(filepath.Dir(config.ConfigPath))
	assert.NoError(err)
}

// TestHostName tests that the TestSite.Hostname() field returns the hostname as expected.
func TestHostName(t *testing.T) {
	assert := assert.New(t)
	testDir := testcommon.CreateTmpDir("TestHostName")
	defer testcommon.Chdir(testDir)()
	defer testcommon.CleanupDir(testDir)
	config, err := NewConfig(testDir)
	assert.Error(err)
	config.Name = util.RandString(32)

	assert.Equal(config.Hostname(), config.Name+"."+DDevTLD)
}

// TestWriteDockerComposeYaml tests the writing of a docker-compose.yaml file.
func TestWriteDockerComposeYaml(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := assert.New(t)
	testDir := testcommon.CreateTmpDir("TestWriteDockerCompose")
	defer testcommon.Chdir(testDir)()
	defer testcommon.CleanupDir(testDir)

	// Create a config
	config, err := NewConfig(testDir)
	assert.Error(err)
	config.Name = util.RandString(32)
	config.AppType = AllowedAppTypes[0]

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
	assert.Contains(contentString, config.Platform)
	assert.Contains(contentString, config.AppType)
}

// TestConfigCommand tests the interactive config options.
func TestConfigCommand(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := assert.New(t)
	testDir := testcommon.CreateTmpDir("TestConfigCommand")

	// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
	defer testcommon.Chdir(testDir)()
	defer testcommon.CleanupDir(testDir)

	// Create a docroot folder.
	err := os.Mkdir(filepath.Join(testDir, "docroot"), 0644)
	if err != nil {
		t.Errorf("Could not create docroot directory under %s", testDir)
	}

	// Create the ddevapp we'll use for testing.
	// This should return an error, since no existing config can be read.
	config, err := NewConfig(testDir)
	assert.Error(err)

	// Randomize some values to use for Stdin during testing.
	name := strings.ToLower(util.RandString(16))
	invalidDir := strings.ToLower(util.RandString(16))
	invalidAppType := strings.ToLower(util.RandString(8))

	// This is a bit hard to follow, but we create an example input buffer that writes the sitename, a (invalid) document root, a valid document root,
	// an invalid app type, and finally a valid site type (drupal8)
	input := fmt.Sprintf("%s\n%s\ndocroot\n%s\ndrupal8", name, invalidDir, invalidAppType)
	scanner := bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)

	restoreOutput := testcommon.CaptureStdOut()
	err = config.Config()
	assert.NoError(err, t)
	out := restoreOutput()

	// Ensure we have expected vales in output.
	assert.Contains(out, "Creating a new ddev project")
	assert.Contains(out, testDir)
	assert.Contains(out, fmt.Sprintf("No directory could be found at %s", filepath.Join(testDir, invalidDir)))
	assert.Contains(out, fmt.Sprintf("%s is not a valid application type", invalidAppType))

	// Ensure values were properly set on the config struct.
	assert.Equal(name, config.Name)
	assert.Equal("drupal8", config.AppType)
	assert.Equal("docroot", config.Docroot)
	err = PrepDdevDirectory(testDir)
	assert.NoError(err)

}

// TestRead tests reading config values from file and fallback to defaults for values not exposed.
func TestRead(t *testing.T) {
	assert := assert.New(t)

	// This closely resembles the values one would have from NewConfig()
	c := &Config{
		ConfigPath: filepath.Join("testing", "config.yaml"),
		AppRoot:    "testing",
		APIVersion: CurrentAppVersion,
		Platform:   DDevDefaultPlatform,
		Name:       "TestRead",
		WebImage:   version.WebImg + ":" + version.WebTag,
		DBImage:    version.DBImg + ":" + version.DBTag,
		DBAImage:   version.DBAImg + ":" + version.DBATag,
	}

	err := c.Read()
	assert.NoError(err)

	// Values not defined in file, we should still have default values
	assert.Equal(c.Name, "TestRead")
	assert.Equal(c.DBImage, version.DBImg+":"+version.DBTag)

	// Values defined in file, we should have values from file
	assert.Equal(c.AppType, "drupal8")
	assert.Equal(c.WebImage, "test/testimage:latest")
}

// TestValidate tests validation of configuration values.
func TestValidate(t *testing.T) {
	assert := assert.New(t)

	cwd, err := os.Getwd()
	assert.NoError(err)

	c := &Config{
		Name:    "TestValidate",
		AppRoot: cwd,
		Docroot: "testing",
		AppType: "wordpress",
	}

	err = c.Validate()
	assert.NoError(err)

	c.Name = "Invalid!"
	err = c.Validate()
	assert.EqualError(err, fmt.Sprintf("%s is not a valid hostname. Please enter a site name in your configuration that will allow for a valid hostname. See https://en.wikipedia.org/wiki/Hostname#Restrictions_on_valid_hostnames for valid hostname requirements", c.Hostname()))

	c.Name = "valid"
	c.Docroot = "invalid"
	err = c.Validate()
	assert.EqualError(err, fmt.Sprintf("no directory could be found at %s. Please enter a valid docroot in your configuration", filepath.Join(cwd, c.Docroot)))

	c.Docroot = "testing"
	c.AppType = "potato"
	err = c.Validate()
	assert.EqualError(err, fmt.Sprintf("%s is not a valid apptype", c.AppType))
}
