package ddevapp_test

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	. "github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	"github.com/google/uuid"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isSemver returns true if a string is a semantic version.
func isSemver(s string) bool {
	_, err := semver.NewVersion(s)
	return err == nil
}

// TestNewConfig tests functionality around creating a new config, writing it to disk, and reading the resulting config.
func TestNewConfig(t *testing.T) {
	assert := asrt.New(t)
	// Create a temporary directory and change to it for the duration of this test.
	testDir := testcommon.CreateTmpDir(t.Name())

	origDir, _ := os.Getwd()

	// Load a new Config
	app, err := NewApp(testDir, true)
	assert.NoError(err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.RemoveAll(testDir)
		assert.NoError(err)
	})

	// Ensure the config uses specified defaults.
	assert.Equal(app.GetDBImage(), docker.GetDBImage(nodeps.MariaDB, ""))
	assert.Equal(app.WebImage, docker.GetWebImage())
	app.Name = util.RandString(32)
	app.Type = nodeps.AppTypeDrupal8

	// WriteConfig the app.
	err = app.WriteConfig()
	assert.NoError(err)
	_, err = os.Stat(app.ConfigPath)
	assert.NoError(err)

	loadedConfig, err := NewApp(testDir, true)
	assert.NoError(err)
	assert.Equal(app.Name, loadedConfig.Name)
	assert.Equal(app.Type, loadedConfig.Type)
}

// TestDisasterConfig tests for disaster opportunities (configing wrong directory, home dir, etc).
func TestDisasterConfig(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	// Make sure we're not allowed to config in home directory.
	tmpDir, _ := os.UserHomeDir()
	_, err := NewApp(tmpDir, false)
	assert.Error(err)
	assert.Contains(err.Error(), "ddev config is not useful")
	_ = os.Chdir(origDir)

	// Create a temporary directory and change to it for the duration of this test.
	tmpDir = testcommon.CreateTmpDir(t.Name())

	// Load a new Config
	app, err := NewApp(tmpDir, false)
	assert.NoError(err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(tmpDir)
	})

	// WriteConfig the app.
	err = app.WriteConfig()
	assert.NoError(err)
	_, err = os.Stat(app.ConfigPath)
	assert.NoError(err)

	subdir := filepath.Join(app.AppRoot, "subdir")
	err = os.Mkdir(subdir, 0777)
	assert.NoError(err)
	err = os.Chdir(subdir)
	assert.NoError(err)
	subdirApp, err := NewApp(subdir, false)
	assert.NoError(err)
	_ = subdirApp

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
	origDir, _ := os.Getwd()
	// Create a temporary directory and change to it for the duration of this test.
	testDir := testcommon.CreateTmpDir("TestPrepDirectory")
	err := os.Chdir(testDir)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(testDir)
		assert.NoError(err)
	})
	app, err := NewApp(testDir, true)
	assert.NoError(err)

	// Prep the directory.
	err = PrepDdevDirectory(app)
	assert.NoError(err)

	// Read directory info an ensure it exists.
	_, err = os.Stat(filepath.Dir(app.ConfigPath))
	assert.NoError(err)
}

// TestHostName tests that the TestSite.Hostname() field returns the hostname as expected.
func TestHostName(t *testing.T) {
	assert := asrt.New(t)
	origDir, _ := os.Getwd()
	testDir := testcommon.CreateTmpDir("TestHostName")
	err := os.Chdir(testDir)
	require.NoError(t, err)
	app, err := NewApp(testDir, true)
	assert.NoError(err)
	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.RemoveAll(testDir)
		assert.NoError(err)
	})
	app.Name = util.RandString(32)

	assert.Equal(app.GetHostname(), strings.ToLower(app.Name+"."+app.ProjectTLD))
}

// TestWriteDockerComposeYaml tests the writing of a .ddev/docker-compose-* file.
func TestWriteDockerComposeYaml(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	origDir, _ := os.Getwd()
	testDir := testcommon.CreateTmpDir(t.Name())

	app, err := NewApp(testDir, true)
	assert.NoError(err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(testDir)
	})

	app.Name = util.RandString(32)
	app.Type = GetValidAppTypes()[0]

	// WriteConfig a config to create/prep necessary directories.
	err = app.WriteConfig()
	assert.NoError(err)

	// After the config has been written and directories exist, the write should work.
	app.DockerEnv()
	err = app.WriteDockerComposeYAML()
	assert.NoError(err)

	// Ensure we can read from the file and that it's a regular file with the expected name.
	fileinfo, err := os.Stat(app.DockerComposeYAMLPath())
	assert.NoError(err)
	assert.False(fileinfo.IsDir())
	assert.Equal(fileinfo.Name(), filepath.Base(app.DockerComposeYAMLPath()))

	composeBytes, err := os.ReadFile(app.DockerComposeYAMLPath())
	assert.NoError(err)
	contentString := string(composeBytes)
	assert.Contains(contentString, app.Type)
}

// TestConfigCommand tests the interactive config options.
func TestConfigCommand(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	origDir, _ := os.Getwd()

	const apptypePos = 0
	const phpVersionPos = 1
	testMatrix := map[string][]string{
		"magentophpversion":  {nodeps.AppTypeMagento, nodeps.PHP74},
		"drupal7phpversion":  {nodeps.AppTypeDrupal7, nodeps.PHPDefault},
		"drupal9phpversion":  {nodeps.AppTypeDrupal9, nodeps.PHPDefault},
		"drupal10phpversion": {nodeps.AppTypeDrupal10, nodeps.PHP81},
	}

	for testName, testValues := range testMatrix {
		testDir := testcommon.CreateTmpDir(t.Name() + "_" + testName)

		// Create a docroot folder.
		err := os.Mkdir(filepath.Join(testDir, "docroot"), 0644)
		if err != nil {
			t.Errorf("Could not create docroot directory under %s", testDir)
		}

		// Create the ddevapp we'll use for testing.
		// This will not return an error, since there is no existing configuration.
		app, err := NewApp(testDir, true)
		assert.NoError(err)

		t.Cleanup(func() {
			err = app.Stop(true, false)
			assert.NoError(err)
			err = os.Chdir(origDir)
			assert.NoError(err)
			err = os.RemoveAll(testDir)
			assert.NoError(err)
		})
		// Randomize some values to use for Stdin during testing.
		name := strings.ToLower(util.RandString(16))
		invalidAppType := strings.ToLower(util.RandString(8))

		// Create an example input buffer that writes the sitename, a valid document root,
		// an invalid app type, and finally a valid app type (from test matrix)
		input := fmt.Sprintf("%s\ndocroot\n%s\n%s", name, invalidAppType, testValues[apptypePos])
		scanner := bufio.NewScanner(strings.NewReader(input))
		util.SetInputScanner(scanner)

		restoreOutput := util.CaptureUserOut()
		err = app.PromptForConfig()
		assert.NoError(err, t)
		out := restoreOutput()

		// Ensure we have expected vales in output.
		assert.Contains(out, testDir)
		assert.Contains(out, fmt.Sprintf("'%s' is not a valid project type", invalidAppType))

		// Create an example input buffer that writes an invalid projectname, then a valid-project-name,
		// a valid document root, a valid app type
		input = fmt.Sprintf("invalid_project_name\n%s\ndocroot\n%s", name, testValues[apptypePos])
		scanner = bufio.NewScanner(strings.NewReader(input))
		util.SetInputScanner(scanner)

		restoreOutput = util.CaptureUserOut()
		err = app.PromptForConfig()
		assert.NoError(err)
		out = restoreOutput()

		// Ensure we have expected vales in output.
		assert.Contains(out, testDir)
		assert.Contains(out, "invalid_project_name is not a valid project name")

		// Ensure values were properly set on the app struct.
		assert.Equal(name, app.Name)
		assert.Equal(testValues[apptypePos], app.Type)
		assert.Equal("docroot", app.Docroot)
		assert.EqualValues(testValues[phpVersionPos], app.PHPVersion, "PHP value incorrect for apptype %v (expected %s got %s) (%v)", app.Type, testValues[phpVersionPos], app.PHPVersion, app)
		err = PrepDdevDirectory(app)
		assert.NoError(err)
	}
}

// TestConfigCommandInteractiveCreateDocrootDenied
func TestConfigCommandInteractiveCreateDocrootDenied(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	origDir, _ := os.Getwd()
	noninteractiveEnv := "DDEV_NONINTERACTIVE"
	// nolint: errcheck
	defer os.Setenv(noninteractiveEnv, os.Getenv(noninteractiveEnv))
	err := os.Unsetenv(noninteractiveEnv)
	assert.NoError(err)

	testMatrix := map[string][]string{
		"drupal6phpversion": {nodeps.AppTypeDrupal6, nodeps.PHP56},
		"drupal7phpversion": {nodeps.AppTypeDrupal7, nodeps.PHPDefault},
		"drupal8phpversion": {nodeps.AppTypeDrupal8, nodeps.PHPDefault},
	}

	for testName := range testMatrix {
		testDir := testcommon.CreateTmpDir(t.Name() + testName)
		err = os.Chdir(testDir)
		require.NoError(t, err)

		// Create the ddevapp we'll use for testing.
		// This will not return an error, since there is no existing configuration.
		app, err := NewApp(testDir, true)
		require.NoError(t, err)

		t.Cleanup(func() {
			err = app.Stop(true, false)
			assert.NoError(err)
			err = os.Chdir(origDir)
			assert.NoError(err)
			err = os.RemoveAll(testDir)
			assert.NoError(err)
		})

		// Randomize some values to use for Stdin during testing.
		name := uuid.New().String()
		nonexistentDocroot := filepath.Join("does", "not", "exist")

		// Create an example input buffer that writes the sitename, a nonexistent document root,
		// and a "no"
		input := fmt.Sprintf("%s\n%s\nno", name, nonexistentDocroot)
		scanner := bufio.NewScanner(strings.NewReader(input))
		util.SetInputScanner(scanner)

		err = app.PromptForConfig()
		assert.Error(err, t)

		// Ensure we have expected vales in output.
		assert.Contains(err.Error(), "docroot must exist to continue configuration")

		err = PrepDdevDirectory(app)
		assert.NoError(err)
		util.Success("Finished %s", t.Name())
	}
}

// TestConfigCommandCreateDocrootAllowed
func TestConfigCommandCreateDocrootAllowed(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	const apptypePos = 0
	const phpVersionPos = 1
	testMatrix := map[string][]string{
		"drupal6phpversion":  {nodeps.AppTypeDrupal6, nodeps.PHP56},
		"drupal7phpversion":  {nodeps.AppTypeDrupal7, nodeps.PHPDefault},
		"drupal10phpversion": {nodeps.AppTypeDrupal10, nodeps.PHP81},
	}

	for testName, testValues := range testMatrix {
		tmpDir := testcommon.CreateTmpDir(t.Name() + testName)

		err := os.Chdir(tmpDir)
		require.NoError(t, err)

		// Create the ddevapp we'll use for testing.
		// This will not return an error, since there is no existing configuration.
		app, err := NewApp(tmpDir, true)
		assert.NoError(err)

		t.Cleanup(func() {
			err = app.Stop(true, false)
			assert.NoError(err)
			err = os.Chdir(origDir)
			assert.NoError(err)
			err = os.RemoveAll(tmpDir)
			assert.NoError(err)
		})

		// Randomize some values to use for Stdin during testing.
		name := uuid.New().String()
		nonexistentDocroot := filepath.Join("does", "not", "exist")

		// Create an example input buffer that writes the sitename, a nonexistent document root,
		// a "yes", and a valid apptype
		input := fmt.Sprintf("%s\n%s\nyes\n%s", name, nonexistentDocroot, testValues[apptypePos])
		scanner := bufio.NewScanner(strings.NewReader(input))
		util.SetInputScanner(scanner)

		restoreOutput := util.CaptureUserOut()
		err = app.PromptForConfig()
		assert.NoError(err, t)
		out := restoreOutput()

		// Ensure we have expected vales in output.
		assert.Contains(out, nonexistentDocroot)
		assert.Contains(out, "Created docroot")

		// Ensure values were properly set on the app struct.
		assert.Equal(name, app.Name)
		assert.Equal(testValues[apptypePos], app.Type)
		assert.Equal(nonexistentDocroot, app.Docroot)
		assert.Equal(testValues[phpVersionPos], app.PHPVersion, "expected php%v for apptype %s", testValues[phpVersionPos], app.Type)

		err = PrepDdevDirectory(app)
		assert.NoError(err)
	}
	util.Success("Finished %s", t.Name())
}

// TestConfigCommandDocrootDetection asserts the default docroot is detected.
func TestConfigCommandDocrootDetection(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	origDir, _ := os.Getwd()

	testMatrix := AvailablePHPDocrootLocations()
	for index, testDocrootName := range testMatrix {
		tmpDir := testcommon.CreateTmpDir(fmt.Sprintf("TestConfigCommand_%v", index))

		// Create a document root folder.
		err := os.MkdirAll(filepath.Join(tmpDir, filepath.Join(testDocrootName)), 0755)
		if err != nil {
			t.Errorf("Could not create %s directory under %s", testDocrootName, tmpDir)
		}
		_, err = os.OpenFile(filepath.Join(tmpDir, filepath.Join(testDocrootName), "index.php"), os.O_RDONLY|os.O_CREATE, 0664)
		assert.NoError(err)

		// Create the ddevapp we'll use for testing.
		// This will not return an error, since there is no existing configuration.
		app, err := NewApp(tmpDir, true)
		assert.NoError(err)

		t.Cleanup(func() {
			err = os.Chdir(origDir)
			assert.NoError(err)
			err = app.Stop(true, false)
			assert.NoError(err)
			_ = os.RemoveAll(tmpDir)
		})
		// Randomize some values to use for Stdin during testing.
		name := strings.ToLower(util.RandString(16))

		// Create an example input buffer that writes the site name, accepts the
		// default document root and provides a valid app type.
		input := fmt.Sprintf("%s\n\ndrupal8", name)
		scanner := bufio.NewScanner(strings.NewReader(input))
		util.SetInputScanner(scanner)

		restoreOutput := util.CaptureStdOut()
		err = app.PromptForConfig()
		assert.NoError(err, t)
		out := restoreOutput()

		assert.Contains(out, fmt.Sprintf("Docroot Location (%s)", testDocrootName))

		// Ensure values were properly set on the app struct.
		assert.Equal(name, app.Name)
		assert.Equal(nodeps.AppTypeDrupal8, app.Type)
		assert.Equal(testDocrootName, app.Docroot)
		err = PrepDdevDirectory(app)
		assert.NoError(err)
	}
}

// TestConfigCommandDocrootDetection asserts the default docroot is detected and has index.php.
// The `web` docroot check is before `docroot` this verifies the directory with an
// existing index.php is selected.
func TestConfigCommandDocrootDetectionIndexVerification(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	testDir := testcommon.CreateTmpDir("TestConfigCommand_testDocrootIndex")

	// Create a document root folder.
	err := os.MkdirAll(filepath.Join(testDir, filepath.Join("web")), 0755)
	if err != nil {
		t.Errorf("Could not create %s directory under %s", "web", testDir)
	}
	err = os.MkdirAll(filepath.Join(testDir, filepath.Join("docroot")), 0755)
	if err != nil {
		t.Errorf("Could not create %s directory under %s", "docroot", testDir)
	}
	_, err = os.OpenFile(filepath.Join(testDir, "docroot", "index.php"), os.O_RDONLY|os.O_CREATE, 0664)
	assert.NoError(err)

	// Create the ddevapp we'll use for testing.
	// This will not return an error, since there is no existing configuration.
	app, err := NewApp(testDir, true)
	assert.NoError(err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		_ = os.RemoveAll(testDir)
	})
	// Randomize some values to use for Stdin during testing.
	name := strings.ToLower(util.RandString(16))

	// Create an example input buffer that writes the site name, accepts the
	// default document root and provides a valid app type.
	input := fmt.Sprintf("%s\n\ndrupal8", name)
	scanner := bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)

	restoreOutput := util.CaptureStdOut()
	err = app.PromptForConfig()
	assert.NoError(err, t)
	out := restoreOutput()

	assert.Contains(out, fmt.Sprintf("Docroot Location (%s)", "docroot"))

	// Ensure values were properly set on the app struct.
	assert.Equal(name, app.Name)
	assert.Equal(nodeps.AppTypeDrupal8, app.Type)
	assert.Equal("docroot", app.Docroot)
	err = PrepDdevDirectory(app)
	assert.NoError(err)
}

// TestReadConfig tests reading config values from file and fallback to defaults for values not exposed.
func TestReadConfig(t *testing.T) {
	assert := asrt.New(t)

	// This closely resembles the values one would have from NewApp()
	app := &DdevApp{
		ConfigPath: filepath.Join("testdata", "config.yaml"),
		AppRoot:    "testdata",
		Name:       "TestRead",
	}

	_, err := app.ReadConfig(false)
	d, _ := os.Getwd()
	require.NoError(t, err, "Unable to c.ReadConfig() in %s, err: %v", d, err)

	// Values not defined in file, we should still have default values
	assert.Equal(app.Name, "TestRead")

	// Values defined in file, we should have values from file
	assert.Equal(app.Type, nodeps.AppTypeDrupal8)
	assert.Equal(app.Docroot, "test")
	assert.Equal(app.WebImage, "test/testimage:latest")
}

// TestReadConfigCRLF tests reading config values from a file with Windows
// CRLF line endings in it.
func TestReadConfigCRLF(t *testing.T) {
	assert := asrt.New(t)

	// This closely resembles the values one would have from NewApp()
	app := &DdevApp{
		ConfigPath: filepath.Join("testdata", t.Name(), ".ddev", "config.yaml"),
		AppRoot:    filepath.Join("testdata", t.Name()),
		Name:       t.Name(),
	}

	_, err := app.ReadConfig(false)
	if err != nil {
		t.Fatalf("Unable to c.ReadConfig(), err: %v", err)
	}

	// Values not defined in file, we should still have default values
	assert.Equal(app.Name, t.Name())

	// Values defined in file, we should have values from file
	assert.Equal(app.Docroot, "public")
}

// TestConfigValidate tests validation of configuration values.
func TestConfigValidate(t *testing.T) {
	if nodeps.IsAppleSilicon() {
		t.Skip("Skipping on mac M1 to ignore problems with 'connection reset by peer'")
	}

	assert := asrt.New(t)
	site := TestSites[0]
	app, err := NewApp(site.Dir, false)
	assert.NoError(err)
	savedApp := *app

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		app = &savedApp
		err = app.WriteConfig()
		assert.NoError(err)
	})

	appName := app.Name
	appType := app.Type

	err = app.ValidateConfig()
	if err != nil {
		t.Fatalf("Failed to app.ValidateConfig(), err=%v", err)
	}

	app.Name = "Invalid!"
	err = app.ValidateConfig()
	assert.Error(err)
	assert.Contains(err.Error(), "not a valid project name")

	app.Name = appName
	app.Type = "potato"
	err = app.ValidateConfig()
	assert.Error(err)
	assert.Contains(err.Error(), "invalid app type")

	app.Type = appType
	app.PHPVersion = "1.1"
	err = app.ValidateConfig()
	assert.Error(err)
	assert.Contains(err.Error(), "unsupported PHP")

	app.PHPVersion = nodeps.PHPDefault
	app.WebserverType = "server"
	err = app.ValidateConfig()
	assert.Error(err)
	assert.Contains(err.Error(), "unsupported webserver type")

	app.WebserverType = nodeps.WebserverDefault
	app.AdditionalHostnames = []string{"good", "b@d"}
	err = app.ValidateConfig()
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "invalid hostname")
	}

	app.AdditionalHostnames = []string{}
	app.AdditionalFQDNs = []string{"good.com", "b@d.com"}
	err = app.ValidateConfig()
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "invalid hostname")
	}

	app.AdditionalFQDNs = []string{}
	// Timezone validation isn't possible on Windows.
	if runtime.GOOS != "windows" {
		app.Timezone = "xxx"
		err = app.ValidateConfig()
		assert.Error(err)
		app.Timezone = "America/Chicago"
		err = app.ValidateConfig()
		assert.NoError(err)
	}

	// Make sure that wildcards work
	app.AdditionalHostnames = []string{"x", "*.any"}
	err = app.ValidateConfig()
	assert.NoError(err)
	err = app.WriteConfig()
	assert.NoError(err)
	// This seems to completely fail on git-bash/Windows/mutagen. Hard to figure out why.
	// Traditional Windows is not a very high priority
	// This apparently started failing with Docker Desktop 4.19.0
	// rfay 2023-05-02
	if runtime.GOOS != "windows" {
		err = app.Start()
		assert.NoError(err)
		err = app.MutagenSyncFlush()
		assert.NoError(err)
		staticURI := site.Safe200URIWithExpectation.URI
		_, _, err = testcommon.GetLocalHTTPResponse(t, "http://x.ddev.site/"+staticURI)
		assert.NoError(err)
		_, _, err = testcommon.GetLocalHTTPResponse(t, "http://somethjingrandom.any.ddev.site/"+staticURI)
		assert.NoError(err)
	}

	// Make sure that a bare "*" in the additional_hostnames does *not* work
	app.AdditionalHostnames = []string{"x", "*"}
	err = app.ValidateConfig()
	assert.Error(err)
}

// TestWriteConfig tests writing config values to file
func TestWriteConfig(t *testing.T) {
	assert := asrt.New(t)

	pwd, _ := os.Getwd()

	projDir, err := filepath.Abs(testcommon.CreateTmpDir("TestWriteConfig"))
	require.NoError(t, err)

	err = fileutil.CopyDir("./testdata/TestWriteConfig/.ddev", filepath.Join(projDir, ".ddev"))
	require.NoError(t, err)

	app, err := NewApp(projDir, true)
	assert.NoError(err)
	err = os.Chdir(projDir)
	assert.NoError(err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(pwd)
		assert.NoError(err)
		err = os.RemoveAll(projDir)
		assert.NoError(err)
	})

	// The default NewApp read should read config overrides, so we should have "config.extra.yaml"
	// as the APIVersion
	assert.Equal("drupal9", app.Type)

	// However, if we ReadConfig() without includeOverrides, we should get "php" as the type
	app, err = NewApp(projDir, false)
	assert.NoError(err)
	assert.Equal("php", app.Type)

	err = app.WriteConfig()
	assert.NoError(err)

	// Now read the config we just wrote; it should have type php because ignored overrides.
	_, err = app.ReadConfig(false)
	assert.NoError(err)
	// app.WriteConfig() writes the version.DdevVersion to the updated config.yaml
	assert.Equal("php", app.Type)
}

// TestConfigOverrideDetection tests to make sure we tell them about config overrides.
func TestConfigOverrideDetection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping because unreliable on Windows")
	}
	testDataDdevDir := filepath.Join("testdata", t.Name(), ".ddev")

	assert := asrt.New(t)
	app := &DdevApp{}
	testDir, _ := os.Getwd()

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()

	defer util.TimeTrackC(fmt.Sprintf("%s ConfigOverrideDetection", site.Name))()

	// Copy test overrides into the project .ddev directory
	for _, item := range []string{"nginx", "nginx_full", "apache", "php", "mysql"} {
		_ = os.RemoveAll(filepath.Join(site.Dir, ".ddev", item))
		err := fileutil.CopyDir(filepath.Join(testDir, testDataDdevDir, item), filepath.Join(site.Dir, ".ddev", item))
		assert.NoError(err)
	}

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)

	t.Cleanup(func() {
		_ = app.Stop(true, false)
		for _, item := range []string{"apache", "php", "mysql", "nginx", "nginx_full"} {
			f := app.GetConfigPath(item)
			err = os.RemoveAll(f)
			if err != nil {
				t.Logf("failed to delete %s: %v", f, err)
			}
		}
	})

	stdoutFunc, err := util.CaptureOutputToFile()
	assert.NoError(err)
	startErr := app.StartAndWait(2)
	stdout := stdoutFunc()

	var logs, health string
	if startErr != nil {
		logs, health, _ = GetErrLogsFromApp(app, startErr)
	}

	require.NoError(t, startErr, "app.StartAndWait() did not succeed: output:\n=====\n%s\n===== health:\n========= health =======\n%s\n========\n===== logs:\n========= logs =======\n%s\n========\n", stdout, health, logs)

	assert.Contains(stdout, "collation.cnf")
	assert.Contains(stdout, "my-php.ini")

	switch app.WebserverType {
	case nodeps.WebserverNginxFPM:
		fallthrough
	case nodeps.WebserverNginxGunicorn:
		assert.Contains(stdout, "nginx-site.conf")
		assert.NotContains(stdout, "apache-site.conf")
		assert.Contains(stdout, "junker99.conf")
	default:
		assert.Contains(stdout, "apache-site.conf")
		assert.NotContains(stdout, "nginx-site.conf")
	}
	assert.Contains(stdout, "Custom configuration is updated")
}

// TestPHPOverrides tests to make sure that PHP overrides work in all webservers.
func TestPHPOverrides(t *testing.T) {
	if nodeps.IsAppleSilicon() {
		t.Skip("Skipping on mac M1 to ignore problems with 'connection reset by peer'")
	}

	assert := asrt.New(t)
	origDir, _ := os.Getwd()
	app := &DdevApp{}

	site := TestSites[0]

	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = app.Stop(true, false)
		err = os.Chdir(origDir)
		assert.NoError(err)
	})
	runTime := util.TimeTrackC(fmt.Sprintf("%s %s", site.Name, t.Name()))

	testcommon.ClearDockerEnv()
	err = app.Init(site.Dir)
	assert.NoError(err)
	err = app.Stop(true, false)
	require.NoError(t, err)

	// Copy test overrides into the project .ddev directory
	err = fileutil.CopyDir(filepath.Join(origDir, "testdata/"+t.Name()+"/.ddev/php"), filepath.Join(site.Dir, ".ddev/php"))
	assert.NoError(err)
	err = fileutil.CopyFile(filepath.Join(origDir, "testdata/"+t.Name()+"/phpinfo.php"), filepath.Join(app.AppRoot, app.Docroot, "phpinfo.php"))
	assert.NoError(err)

	// And when we're done, we have to clean those out again.
	t.Cleanup(func() {
		runTime()
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)

		err = os.RemoveAll(filepath.Join(app.AppRoot, ".ddev/php"))
		if err != nil {
			t.Logf("failed to remove .ddev/php: %v", err)
		}
		err = os.RemoveAll(filepath.Join(app.AppRoot, app.Docroot, "phpinfo.php"))
		if err != nil {
			t.Logf("failed to remove phpinfo.php: %v", err)
		}
	})

	startErr := app.StartAndWait(5)
	assert.NoError(startErr)
	if startErr != nil {
		logs, health, _ := GetErrLogsFromApp(app, startErr)
		t.Fatalf("============== health from app.StartAndWait() ==============\n%s\n============== logs from app.StartAndWait() ==============\n%s\n", health, logs)
	}

	err = app.MutagenSyncFlush()
	require.NoError(t, err, "failed to flush mutagen sync")
	_, _ = testcommon.EnsureLocalHTTPContent(t, "http://"+app.GetHostname()+"/phpinfo.php", `max_input_time</td><td class="v">999`, 60)

}

// TestPHPConfig checks some key PHP configuration items
func TestPHPConfig(t *testing.T) {

	assert := asrt.New(t)
	origDir, _ := os.Getwd()
	app := &DdevApp{}
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	testcommon.ClearDockerEnv()
	err = app.Init(site.Dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		err = app.Stop(true, false)
		assert.NoError(err)
	})

	// Most of the time there's no reason to do all versions of PHP
	phpKeys := []string{}
	exclusions := []string{"5.6", "7.0", "7.1", "7.2", "7.3", "7.4"}
	for k := range nodeps.ValidPHPVersions {
		if os.Getenv("GOTEST_SHORT") != "" && !nodeps.ArrayContainsString(exclusions, k) {
			phpKeys = append(phpKeys, k)
		}
	}
	sort.Strings(phpKeys)

	for _, v := range phpKeys {
		app.PHPVersion = v
		err = app.Start()
		require.NoError(t, err)

		out, _, err := app.Exec(&ExecOpts{
			Cmd: "php --version",
		})
		require.NoError(t, err)
		t.Logf("============= PHP version=%s ================", out)

		// Look for problems with serialize_precision,https://github.com/ddev/ddev/issues/5092
		out, _, err = app.Exec(&ExecOpts{
			Cmd: `php -r "var_dump(0.6);"`,
		})
		require.NoError(t, err)
		out = strings.Trim(out, "\n")
		require.Equal(t, `float(0.6)`, out)
	}

	err = app.Stop(true, false)
	require.NoError(t, err)
}

// TestPostgresConfigOverride makes sure that overriding Postgres config works
func TestPostgresConfigOverride(t *testing.T) {
	assert := asrt.New(t)
	origDir, _ := os.Getwd()
	tmpDir := testcommon.CreateTmpDir(t.Name())
	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	app, err := NewApp(tmpDir, false)
	require.NoError(t, err)
	app.Name = t.Name()
	app.Database = DatabaseDesc{Type: nodeps.Postgres, Version: nodeps.PostgresDefaultVersion}
	err = app.WriteConfig()
	require.NoError(t, err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		_ = os.RemoveAll(tmpDir)
	})

	err = app.Start()
	require.NoError(t, err)

	out, stderr, err := app.Exec(&ExecOpts{
		Service: "db",
		Cmd:     `psql -t -c "SELECT setting FROM pg_settings WHERE name='max_connections'"`,
	})
	assert.NoError(err)
	assert.Equal(" 100\n\n", out, "out: %s, stderr: %s", out, stderr)

	cfg, err := fileutil.ReadFileIntoString(app.GetConfigPath("postgres/postgresql.conf"))
	require.NoError(t, err)
	cfg = strings.ReplaceAll(cfg, "#ddev-generated", "#")
	cfg = strings.ReplaceAll(cfg, `max_connections = 100`, `max_connections = 200`)
	err = fileutil.TemplateStringToFile(cfg, nil, app.GetConfigPath("postgres/postgresql.conf"))
	require.NoError(t, err)
	err = app.Restart()
	require.NoError(t, err)
	out, stderr, err = app.Exec(&ExecOpts{
		Service: "db",
		Cmd:     `psql -t -c "SELECT setting FROM pg_settings WHERE name='max_connections'"`,
	})
	assert.NoError(err)
	assert.Equal(" 200\n\n", out, "out: %s, stderr: %s", out, stderr)
}

// TestExtraPackages tests to make sure that *extra_packages config.yaml directives
// work (and are overridden by *-build/Dockerfile).
func TestExtraPackages(t *testing.T) {
	assert := asrt.New(t)
	app := &DdevApp{}

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()

	runTime := util.TimeTrackC(fmt.Sprintf("%s %s", site.Name, t.Name()))

	err := app.Init(site.Dir)
	assert.NoError(err)
	err = app.Stop(true, false)
	assert.NoError(err)

	// Make sure no "-built" items are still hanging around
	command := fmt.Sprintf("docker rmi -f %s-%s-built %s-%s-built", app.WebImage, app.Name, app.GetDBImage(), app.Name)
	_, _ = exec.RunCommand("bash", []string{"-c", command})

	t.Cleanup(func() {
		runTime()
		_ = app.Stop(true, false)
		app.WebImageExtraPackages = nil
		app.DBImageExtraPackages = nil
		app.PHPVersion = nodeps.PHPDefault
		_ = app.WriteConfig()
		_ = fileutil.RemoveContents(app.GetConfigPath("web-build"))
		_ = fileutil.RemoveContents(app.GetConfigPath("db-build"))
		command := fmt.Sprintf("docker rmi -f %s-%s-built %s-%s-built >/dev/null 2>&1", app.WebImage, app.Name, app.GetDBImage(), app.Name)
		_, _ = exec.RunCommand("bash", []string{"-c", command})
	})

	// Start and make sure that the packages don't exist already
	err = app.Start()
	require.NoError(t, err)

	// Test db container to make sure no ncdu in there at beginning
	_, _, err = app.Exec(&ExecOpts{
		Service: "db",
		Cmd:     "command -v ncdu 2>/dev/null",
	})
	assert.Error(err)
	assert.Contains(err.Error(), "exit status 1")

	addedPackage := "tidy"
	addedPackageTitle := "Tidy"

	_, _, err = app.Exec(&ExecOpts{
		Service: "web",
		Cmd:     fmt.Sprintf("dpkg -s php%s-%s >/dev/null 2>&1", app.PHPVersion, addedPackage),
	})
	assert.Error(err)
	assert.Contains(err.Error(), "exit status 1")

	// Now add the packages and start again, they should be in there
	app.WebImageExtraPackages = []string{"php" + app.PHPVersion + "-" + addedPackage}
	app.DBImageExtraPackages = []string{"ncdu"}
	err = app.Restart()
	require.NoError(t, err)

	stdout, stderr, err := app.Exec(&ExecOpts{
		Service: "web",
		Cmd:     "dpkg -s php" + app.PHPVersion + "-" + addedPackage,
	})
	assert.NoError(err, "dpkg -s php%s-%s failed", app.PHPVersion, addedPackage, stdout, stderr)

	stdout, stderr, err = app.Exec(&ExecOpts{
		Service: "web",
		Cmd:     fmt.Sprintf("php -i | grep  '%s support =. enabled'", addedPackageTitle),
	})
	assert.NoError(err, "failed to grep for %s support, stdout=%s, stderr=%s", addedPackage, stdout, stderr)

	stdout, _, err = app.Exec(&ExecOpts{
		Service: "db",
		Cmd:     "command -v ncdu",
	})
	assert.NoError(err)
	assert.Equal("/usr/bin/ncdu", strings.Trim(stdout, "\r\n"))
}

// TestTimezoneConfig tests to make sure setting timezone config takes effect in the container.
func TestTimezoneConfig(t *testing.T) {
	assert := asrt.New(t)
	app := &DdevApp{}

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()

	runTime := util.TimeTrackC(fmt.Sprintf("%s %s", t.Name(), site.Name))

	err := app.Init(site.Dir)
	assert.NoError(err)
	err = app.Stop(true, false)
	assert.NoError(err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
	})

	err = app.Start()
	assert.NoError(err)

	// Without timezone set, we should find Etc/UTC
	stdout, _, err := app.Exec(&ExecOpts{
		Service: "web",
		Cmd:     "printf \"timezone=$(date +%Z)\n\" && php -r 'print \"phptz=\" . date_default_timezone_get();'",
	})
	assert.NoError(err)
	assert.Equal("timezone=UTC\nphptz=UTC", stdout)

	// Make sure db container is also working
	stdout, _, err = app.Exec(&ExecOpts{
		Service: "db",
		Cmd:     "echo -n timezone=$(date +%Z)",
	})
	assert.NoError(err)
	assert.Equal("timezone=UTC", stdout)

	// With timezone set, we the correct timezone operational
	app.Timezone = "Europe/Paris"
	err = app.Start()
	require.NoError(t, err)
	stdout, _, err = app.Exec(&ExecOpts{
		Service: "web",
		Cmd:     "printf \"timezone=$(date +%Z)\n\" && php -r 'print \"phptz=\" . date_default_timezone_get();'",
	})
	assert.NoError(err)
	assert.Regexp(regexp.MustCompile("timezone=CES?T\nphptz=Europe/Paris"), stdout)

	// Make sure db container is also working with CET
	stdout, _, err = app.Exec(&ExecOpts{
		Service: "db",
		Cmd:     "echo -n timezone=$(date +%Z)",
	})
	assert.NoError(err)
	assert.Regexp(regexp.MustCompile("timezone=CES?T"), stdout)

	runTime()
}

// TestComposerVersionConfig tests to make sure setting composer version takes effect in the container.
func TestComposerVersionConfig(t *testing.T) {
	if nodeps.IsAppleSilicon() || dockerutil.IsColima() {
		t.Skip("Skipping on Mac M1 and Colima, just lots of failed network connections")
	}
	assert := asrt.New(t)
	app := &DdevApp{}

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()

	runTime := util.TimeTrackC(fmt.Sprintf("%s %s", t.Name(), site.Name))

	err := app.Init(site.Dir)
	assert.NoError(err)
	err = app.Stop(true, false)
	assert.NoError(err)

	t.Cleanup(func() {
		app.ComposerVersion = ""
		err = app.WriteConfig()
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
	})

	for _, testVersion := range []string{"2", "2.2", "2.5.5", "1", "stable", "preview", "snapshot"} {
		app.ComposerVersion = testVersion
		err = app.Start()
		assert.NoError(err)

		stdout, _, err := app.Exec(&ExecOpts{
			Service: "web",
			Cmd:     "composer --version 2>/dev/null | awk '/Composer version/ {print $3;}'",
		})
		assert.NoError(err)

		// Ignore the non semantic versions for the moment e.g. stable or preview
		// TODO: figure out a way to test version key words
		if isSemver(testVersion) {
			if strings.Count(testVersion, ".") < 2 {
				assert.Contains(strings.TrimSpace(stdout), testVersion)
			} else {
				assert.Equal(testVersion, strings.TrimSpace(stdout))
			}
		}
	}

	runTime()
}

// TestCustomBuildDockerfiles tests to make sure that custom web-build and db-build
// Dockerfiles work properly
func TestCustomBuildDockerfiles(t *testing.T) {
	assert := asrt.New(t)
	app := &DdevApp{}

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()

	runTime := util.TimeTrackC(fmt.Sprintf("%s TestCustomBuildDockerfiles", site.Name))

	err := app.Init(site.Dir)
	assert.NoError(err)
	err = app.Stop(true, false)
	assert.NoError(err)

	t.Cleanup(func() {
		runTime()
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.RemoveAll(app.GetConfigPath("web-build"))
		assert.NoError(err)
		err = os.RemoveAll(app.GetConfigPath("db-build"))
		assert.NoError(err)
	})

	// Create simple dockerfiles that just touch /var/tmp/added-by-<container>txt
	for _, item := range []string{"web", "db"} {
		err = fileutil.TemplateStringToFile("junkfile", nil, app.GetConfigPath(fmt.Sprintf("%s-build/junkfile", item)))
		assert.NoError(err)
		err = WriteImageDockerfile(app.GetConfigPath(item+"-build/Dockerfile"), []byte(`
RUN touch /var/tmp/`+"added-by-"+item+".txt"))
		assert.NoError(err)
		// Add also Dockerfile.* alternatives
		// Last one includes previously recommended ARG/FROM that needs to be removed
		err = WriteImageDockerfile(app.GetConfigPath(item+"-build/Dockerfile.test1"), []byte(`
ADD junkfile /
RUN touch /var/tmp/`+"added-by-"+item+"-test1.txt"))
		assert.NoError(err)

		err = WriteImageDockerfile(app.GetConfigPath(item+"-build/Dockerfile.test2"), []byte(`
RUN touch /var/tmp/`+"added-by-"+item+"-test2.txt"))
		assert.NoError(err)

		// Testing pre.Dockerfile.*
		err = WriteImageDockerfile(app.GetConfigPath(item+"-build/pre.Dockerfile.test3"), []byte(`
RUN touch /var/tmp/`+"added-by-"+item+"-test3.txt"))
		assert.NoError(err)

		// Testing that pre comes before post, we create a file on pre and remove
		// it on post
		err = WriteImageDockerfile(app.GetConfigPath(item+"-build/pre.Dockerfile.test4"), []byte(`
RUN touch /var/tmp/`+"added-by-"+item+"-test4.txt"))
		assert.NoError(err)
		err = WriteImageDockerfile(app.GetConfigPath(item+"-build/Dockerfile.test4"), []byte(`
RUN rm /var/tmp/`+"added-by-"+item+"-test4.txt"))
		assert.NoError(err)
	}

	// Make sure that DDEV_PHP_VERSION gets into the build
	err = WriteImageDockerfile(app.GetConfigPath("web-build/Dockerfile.ddev-php-version"), []byte(`
ARG DDEV_PHP_VERSION
RUN touch /var/tmp/running-php-${DDEV_PHP_VERSION}
`))
	require.NoError(t, err)

	// Start and make sure that the packages don't exist already
	err = app.Start()
	require.NoError(t, err)

	// Make sure that the expected in-container file has been created
	for _, item := range []string{"web", "db"} {
		_, _, err = app.Exec(&ExecOpts{
			Service: item,
			Cmd:     "ls /junkfile",
		})
		assert.NoError(err)
		_, _, err = app.Exec(&ExecOpts{
			Service: item,
			Cmd:     "ls /var/tmp/added-by-" + item + ".txt >/dev/null",
		})
		assert.NoError(err)
		_, _, err = app.Exec(&ExecOpts{
			Service: item,
			Cmd:     "ls /var/tmp/added-by-" + item + "-test1.txt >/dev/null",
		})
		assert.NoError(err)
		_, _, err = app.Exec(&ExecOpts{
			Service: item,
			Cmd:     "ls /var/tmp/added-by-" + item + "-test2.txt >/dev/null",
		})
		assert.NoError(err)
		_, _, err = app.Exec(&ExecOpts{
			Service: item,
			Cmd:     "ls /var/tmp/added-by-" + item + "-test3.txt >/dev/null",
		})
		assert.NoError(err)
		_, _, err = app.Exec(&ExecOpts{
			Service: item,
			Cmd:     "ls /var/tmp/added-by-" + item + "-test4.txt 2>/dev/null",
		})
		assert.Error(err)
	}

	_, _, err = app.Exec(&ExecOpts{
		Cmd: fmt.Sprintf("ls /var/tmp/running-php-%s >/dev/null", app.PHPVersion),
	})
	assert.NoError(err)
}

// TestConfigLoadingOrder verifies that configs load in lexicographical order
// AFTER config.yaml
func TestConfigLoadingOrder(t *testing.T) {
	assert := asrt.New(t)
	pwd, _ := os.Getwd()

	projDir, err := filepath.Abs(testcommon.CreateTmpDir(t.Name()))
	require.NoError(t, err)

	err = fileutil.CopyDir("./testdata/TestConfigLoadingOrder/.ddev", filepath.Join(projDir, ".ddev"))
	require.NoError(t, err)

	app, err := NewApp(projDir, true)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = os.Chdir(pwd)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.RemoveAll(projDir)
		assert.NoError(err)
	})
	err = os.Chdir(app.AppRoot)
	assert.NoError(err)
	_, err = app.ReadConfig(true)
	assert.NoError(err)
	assert.Equal("config.yaml", app.WebImage)

	matches, err := filepath.Glob(filepath.Join(projDir, ".ddev/linkedconfigs/config.*.y*ml"))
	assert.NoError(err)

	// First make sure that each possible item in .ddev/linkedconfigs can override by itself
	for _, item := range matches {
		linkedMatch := filepath.Join(app.AppConfDir(), filepath.Base(item))
		assert.NoError(err)
		err = os.Symlink(item, linkedMatch)
		assert.NoError(err)
		app, err = NewApp(app.AppRoot, true)
		assert.NoError(err)
		assert.Equal(filepath.Base(item), app.WebImage)
		err = os.Remove(linkedMatch)
		assert.NoError(err)
	}

	// Now make sure that the matches are in lexicographical order and each larger one can override the others
	orderedMatches := matches
	sort.Strings(orderedMatches)
	assert.Equal(matches, orderedMatches)

	for _, item := range matches {
		linkedMatch := filepath.Join(app.AppConfDir(), filepath.Base(item))
		assert.NoError(err)
		err = os.Symlink(item, linkedMatch)
		assert.NoError(err)
		app, err = NewApp(app.AppRoot, true)
		assert.Equal(filepath.Base(item), app.WebImage)
	}

	// Now we still have all those linked overrides, but do a NewApp() without allowing them
	// and verify that they don't get loaded
	app, err = NewApp(app.AppRoot, false)
	assert.NoError(err)
	assert.Equal("config.yaml", app.WebImage)
}

// TestPkgConfigDatabaseDBVersion tests config for database
func TestPkgConfigDatabaseDBVersion(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	// Create a temporary directory and switch to it.
	tmpDir := testcommon.CreateTmpDir(t.Name())
	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	_, _ = exec.RunHostCommand(DdevBin, "delete", "-Oy", t.Name())
	err = globalconfig.ReadGlobalConfig()
	require.NoError(t, err)

	app, err := NewApp(tmpDir, false)
	require.NoError(t, err)
	app.Name = t.Name()
	err = app.WriteConfig()
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		_ = os.RemoveAll(tmpDir)
		err = os.Chdir(origDir)
		assert.NoError(err)
	})
	dbVersions := nodeps.GetValidDatabaseVersions()
	for _, v := range dbVersions {
		parts := strings.Split(v, ":")
		require.True(t, len(parts) == 2)
		configFile := app.ConfigPath
		err = os.RemoveAll(configFile)
		assert.NoError(err)
		err = fileutil.AppendStringToFile(configFile, fmt.Sprintf("database:\n  type: %s\n  version: %s ", parts[0], parts[1]))
		err = app.LoadConfigYamlFile(configFile)
		assert.NoError(err)
		assert.Equal(parts[0], app.Database.Type)
		assert.Equal(parts[1], app.Database.Version)
	}
}

// TestDatabaseConfigUpgrade tests whether upgrade from mariadb_version/mysql_version
// to database format works correctly
func TestDatabaseConfigUpgrade(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	// Create a temporary directory and switch to it.
	tmpDir := testcommon.CreateTmpDir(t.Name())
	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	_, _ = exec.RunHostCommand(DdevBin, "delete", "-Oy", t.Name())
	err = globalconfig.ReadGlobalConfig()
	require.NoError(t, err)

	app, err := NewApp(tmpDir, false)
	require.NoError(t, err)
	app.Name = t.Name()
	err = app.WriteConfig()
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		_ = os.RemoveAll(tmpDir)
		err = os.Chdir(origDir)
		assert.NoError(err)
	})
	for _, v := range []string{"mariadb:10.5", "mysql:5.7"} {
		parts := strings.Split(v, ":")
		require.True(t, len(parts) == 2)
		configFile := app.ConfigPath
		err = os.RemoveAll(configFile)
		assert.NoError(err)
		err = fileutil.AppendStringToFile(configFile, fmt.Sprintf("name: %s\n%s_version: %s\n", t.Name(), parts[0], parts[1]))
		app, err := NewApp(tmpDir, false)
		require.NoError(t, err)
		assert.Equal(parts[0], app.Database.Type)
		assert.Equal(parts[1], app.Database.Version)
		assert.Empty(app.MySQLVersion)
		assert.Empty(app.MariaDBVersion)
	}
}
