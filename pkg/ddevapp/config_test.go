package ddevapp_test

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	asrt "github.com/stretchr/testify/assert"
)

// TestNewConfig tests functionality around creating a new config, writing it to disk, and reading the resulting config.
func TestNewConfig(t *testing.T) {
	assert := asrt.New(t)
	// Create a temporary directory and change to it for the duration of this test.
	testDir := testcommon.CreateTmpDir("TestNewConfig")

	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	// Load a new Config
	app, err := NewApp(testDir, DefaultProviderName)
	assert.NoError(err)

	// Ensure the config uses specified defaults.
	assert.Equal(app.APIVersion, CurrentAppVersion)
	assert.Equal(app.DBImage, version.DBImg+":"+version.DBTag)
	assert.Equal(app.WebImage, version.WebImg+":"+version.WebTag)
	assert.Equal(app.DBAImage, version.DBAImg+":"+version.DBATag)
	app.Name = util.RandString(32)
	app.Type = "drupal8"

	// WriteConfig the app.
	err = app.WriteConfig()
	assert.NoError(err)
	_, err = os.Stat(app.ConfigPath)
	assert.NoError(err)

	loadedConfig, err := NewApp(testDir, DefaultProviderName)
	assert.NoError(err)
	assert.Equal(app.Name, loadedConfig.Name)
	assert.Equal(app.Type, loadedConfig.Type)

}

// TestAllowedAppType tests the IsAllowedAppType function.
func TestAllowedAppTypes(t *testing.T) {
	assert := asrt.New(t)
	for _, v := range GetValidAppTypes() {
		assert.True(IsValidAppType(v))
	}

	for i := 1; i <= 50; i++ {
		randomType := util.RandString(32)
		assert.False(IsValidAppType(randomType))
	}
}

// TestPrepDirectory ensures the configuration directory can be created with the correct permissions.
func TestPrepDirectory(t *testing.T) {
	assert := asrt.New(t)
	// Create a temporary directory and change to it for the duration of this test.
	testDir := testcommon.CreateTmpDir("TestPrepDirectory")
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	app, err := NewApp(testDir, DefaultProviderName)
	assert.NoError(err)

	// Prep the directory.
	err = PrepDdevDirectory(filepath.Dir(app.ConfigPath))
	assert.NoError(err)

	// Read directory info an ensure it exists.
	_, err = os.Stat(filepath.Dir(app.ConfigPath))
	assert.NoError(err)
}

// TestHostName tests that the TestSite.Hostname() field returns the hostname as expected.
func TestHostName(t *testing.T) {
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir("TestHostName")
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()
	app, err := NewApp(testDir, DefaultProviderName)
	assert.NoError(err)
	app.Name = util.RandString(32)

	assert.Equal(app.GetHostname(), app.Name+"."+version.DDevTLD)
}

// TestWriteDockerComposeYaml tests the writing of a docker-compose.yaml file.
func TestWriteDockerComposeYaml(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir("TestWriteDockerCompose")
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	app, err := NewApp(testDir, DefaultProviderName)
	assert.NoError(err)
	app.Name = util.RandString(32)
	app.Type = GetValidAppTypes()[0]

	// WriteConfig a config to create/prep necessary directories.
	err = app.WriteConfig()
	assert.NoError(err)

	// After the config has been written and directories exist, the write should work.
	err = app.WriteDockerComposeConfig()
	assert.NoError(err)

	// Ensure we can read from the file and that it's a regular file with the expected name.
	fileinfo, err := os.Stat(app.DockerComposeYAMLPath())
	assert.NoError(err)
	assert.False(fileinfo.IsDir())
	assert.Equal(fileinfo.Name(), filepath.Base(app.DockerComposeYAMLPath()))

	composeBytes, err := ioutil.ReadFile(app.DockerComposeYAMLPath())
	assert.NoError(err)
	contentString := string(composeBytes)
	assert.Contains(contentString, app.Platform)
	assert.Contains(contentString, app.Type)
}

// TestConfigCommand tests the interactive config options.
func TestConfigCommand(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)

	const apptypePos = 0
	const phpVersionPos = 1
	testMatrix := map[string][]string{
		"drupal6phpversion": {"drupal6", "5.6"},
		"drupal7phpversion": {"drupal7", "7.1"},
		"drupal8phpversion": {"drupal8", "7.1"},
	}

	for testName, testValues := range testMatrix {

		testDir := testcommon.CreateTmpDir("TestConfigCommand_" + testName)

		// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
		defer testcommon.CleanupDir(testDir)
		defer testcommon.Chdir(testDir)()

		// Create a docroot folder.
		err := os.Mkdir(filepath.Join(testDir, "docroot"), 0644)
		if err != nil {
			t.Errorf("Could not create docroot directory under %s", testDir)
		}

		// Create the ddevapp we'll use for testing.
		// This will not return an error, since there is no existing configuration.
		app, err := NewApp(testDir, DefaultProviderName)
		assert.NoError(err)

		// Randomize some values to use for Stdin during testing.
		name := strings.ToLower(util.RandString(16))
		invalidDir := strings.ToLower(util.RandString(16))
		invalidAppType := strings.ToLower(util.RandString(8))

		// Create an example input buffer that writes the sitename, an invalid
		// document root, a valid document root,
		// an invalid app type, and finally a valid app type (from test matrix)
		input := fmt.Sprintf("%s\n%s\ndocroot\n%s\n%s", name, invalidDir, invalidAppType, testValues[apptypePos])
		scanner := bufio.NewScanner(strings.NewReader(input))
		util.SetInputScanner(scanner)

		restoreOutput := testcommon.CaptureUserOut()
		err = app.PromptForConfig()
		assert.NoError(err, t)
		out := restoreOutput()

		// Ensure we have expected vales in output.
		assert.Contains(out, testDir)
		assert.Contains(out, fmt.Sprintf("No directory could be found at %s", filepath.Join(testDir, invalidDir)))
		assert.Contains(out, fmt.Sprintf("'%s' is not a valid application type", invalidAppType))

		// Ensure values were properly set on the app struct.
		assert.Equal(name, app.Name)
		assert.Equal(testValues[apptypePos], app.Type)
		assert.Equal("docroot", app.Docroot)
		assert.EqualValues(testValues[phpVersionPos], app.PHPVersion)
		err = PrepDdevDirectory(testDir)
		assert.NoError(err)
	}
}

// TestConfigCommandDocrootDetection asserts the default docroot is detected.
func TestConfigCommandDocrootDetection(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)

	testMatrix := AvailableDocrootLocations()
	for index, testDocrootName := range testMatrix {
		testDir := testcommon.CreateTmpDir(fmt.Sprintf("TestConfigCommand_%v", index))

		// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
		defer testcommon.CleanupDir(testDir)
		defer testcommon.Chdir(testDir)()

		// Create a document root folder.
		err := os.MkdirAll(filepath.Join(testDir, filepath.Join(testDocrootName)), 0755)
		if err != nil {
			t.Errorf("Could not create %s directory under %s", testDocrootName, testDir)
		}
		os.OpenFile(filepath.Join(testDir, filepath.Join(testDocrootName), "index.php"), os.O_RDONLY|os.O_CREATE, 0664)

		// Create the ddevapp we'll use for testing.
		// This will not return an error, since there is no existing configuration.
		app, err := NewApp(testDir, DefaultProviderName)
		assert.NoError(err)

		// Randomize some values to use for Stdin during testing.
		name := strings.ToLower(util.RandString(16))

		// Create an example input buffer that writes the site name, accepts the
		// default document root and provides a valid app type.
		input := fmt.Sprintf("%s\n\ndrupal8", name)
		scanner := bufio.NewScanner(strings.NewReader(input))
		util.SetInputScanner(scanner)

		restoreOutput := testcommon.CaptureStdOut()
		err = app.PromptForConfig()
		assert.NoError(err, t)
		out := restoreOutput()

		assert.Contains(out, fmt.Sprintf("Docroot Location (%s)", testDocrootName))

		// Ensure values were properly set on the app struct.
		assert.Equal(name, app.Name)
		assert.Equal("drupal8", app.Type)
		assert.Equal(testDocrootName, app.Docroot)
		err = PrepDdevDirectory(testDir)
		assert.NoError(err)
	}
}

// TestConfigCommandDocrootDetection asserts the default docroot is detected and has index.php.
// The `web` docroot check is before `docroot` this verifies the directory with an
// existing index.php is selected.
func TestConfigCommandDocrootDetectionIndexVerification(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)

	testDir := testcommon.CreateTmpDir("TestConfigCommand_testDocrootIndex")

	// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	// Create a document root folder.
	err := os.MkdirAll(filepath.Join(testDir, filepath.Join("web")), 0755)
	if err != nil {
		t.Errorf("Could not create %s directory under %s", "web", testDir)
	}
	err = os.MkdirAll(filepath.Join(testDir, filepath.Join("docroot")), 0755)
	if err != nil {
		t.Errorf("Could not create %s directory under %s", "docroot", testDir)
	}
	os.OpenFile(filepath.Join(testDir, "docroot", "index.php"), os.O_RDONLY|os.O_CREATE, 0664)

	// Create the ddevapp we'll use for testing.
	// This will not return an error, since there is no existing configuration.
	app, err := NewApp(testDir, DefaultProviderName)
	assert.NoError(err)

	// Randomize some values to use for Stdin during testing.
	name := strings.ToLower(util.RandString(16))

	// Create an example input buffer that writes the site name, accepts the
	// default document root and provides a valid app type.
	input := fmt.Sprintf("%s\n\ndrupal8", name)
	scanner := bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)

	restoreOutput := testcommon.CaptureStdOut()
	err = app.PromptForConfig()
	assert.NoError(err, t)
	out := restoreOutput()

	assert.Contains(out, fmt.Sprintf("Docroot Location (%s)", "docroot"))

	// Ensure values were properly set on the app struct.
	assert.Equal(name, app.Name)
	assert.Equal("drupal8", app.Type)
	assert.Equal("docroot", app.Docroot)
	err = PrepDdevDirectory(testDir)
	assert.NoError(err)
}

// TestReadConfig tests reading config values from file and fallback to defaults for values not exposed.
func TestReadConfig(t *testing.T) {
	assert := asrt.New(t)

	// This closely resembles the values one would have from NewApp()
	app := &DdevApp{
		ConfigPath: filepath.Join("testdata", "config.yaml"),
		AppRoot:    "testdata",
		APIVersion: CurrentAppVersion,
		Name:       "TestRead",
		WebImage:   version.WebImg + ":" + version.WebTag,
		DBImage:    version.DBImg + ":" + version.DBTag,
		DBAImage:   version.DBAImg + ":" + version.DBATag,
		Provider:   DefaultProviderName,
	}

	err := app.ReadConfig()
	if err != nil {
		t.Fatalf("Unable to c.ReadConfig(), err: %v", err)
	}

	// Values not defined in file, we should still have default values
	assert.Equal(app.Name, "TestRead")
	assert.Equal(app.DBImage, version.DBImg+":"+version.DBTag)

	// Values defined in file, we should have values from file
	assert.Equal(app.Type, "drupal8")
	assert.Equal(app.WebImage, "test/testimage:latest")
}

// TestValidate tests validation of configuration values.
func TestValidate(t *testing.T) {
	assert := asrt.New(t)

	cwd, err := os.Getwd()
	assert.NoError(err)

	app := &DdevApp{
		Name:    "TestValidate",
		AppRoot: cwd,
		Docroot: "testdata",
		Type:    "wordpress",
	}

	err = app.ValidateConfig()
	if err != nil {
		t.Fatalf("Failed to app.ValidateConfig(), err=%v", err)
	}

	app.Name = "Invalid!"
	err = app.ValidateConfig()
	assert.EqualError(err, fmt.Sprintf("%s is not a valid hostname. Please enter a site name in your configuration that will allow for a valid hostname. See https://en.wikipedia.org/wiki/Hostname#Restrictions_on_valid_hostnames for valid hostname requirements", app.GetHostname()))

	app.Name = "valid"
	app.Docroot = "invalid"
	err = app.ValidateConfig()
	assert.EqualError(err, fmt.Sprintf("no directory could be found at %s. Please enter a valid docroot in your configuration", filepath.Join(cwd, app.Docroot)))

	app.Docroot = "testdata"
	app.Type = "potato"
	err = app.ValidateConfig()
	assert.EqualError(err, fmt.Sprintf("'%s' is not a valid apptype", app.Type))
}

// TestWriteConfig tests writing config values to file
func TestWriteConfig(t *testing.T) {
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir("TestConfigWrite")

	// This closely resembles the values one would have from NewApp()
	app := &DdevApp{
		ConfigPath: filepath.Join(testDir, "config.yaml"),
		AppRoot:    testDir,
		APIVersion: CurrentAppVersion,
		Name:       "TestWrite",
		WebImage:   version.WebImg + ":" + version.WebTag,
		DBImage:    version.DBImg + ":" + version.DBTag,
		DBAImage:   version.DBAImg + ":" + version.DBATag,
		Type:       "drupal8",
		Provider:   DefaultProviderName,
	}

	err := app.WriteConfig()
	assert.NoError(err)

	out, err := ioutil.ReadFile(filepath.Join(testDir, "config.yaml"))
	assert.NoError(err)
	assert.Contains(string(out), "TestWrite")
	assert.Contains(string(out), `exec: "drush cr"`)

	app.Type = "wordpress"
	err = app.WriteConfig()
	assert.NoError(err)

	out, err = ioutil.ReadFile(filepath.Join(testDir, "config.yaml"))
	assert.NoError(err)
	assert.Contains(string(out), `exec: "wp search-replace`)

	err = os.RemoveAll(testDir)
	assert.NoError(err)
}
