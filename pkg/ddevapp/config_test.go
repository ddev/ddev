package ddevapp_test

import (
	"bufio"
	"fmt"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	. "github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/google/uuid"
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
	app, err := NewApp(testDir, true, nodeps.ProviderDefault)
	assert.NoError(err)

	// Ensure the config uses specified defaults.
	assert.Equal(app.APIVersion, version.DdevVersion)
	assert.Equal(app.GetDBImage(), version.GetDBImage(nodeps.MariaDB))
	assert.Equal(app.WebImage, version.GetWebImage())
	assert.Equal(app.DBAImage, version.GetDBAImage())
	app.Name = util.RandString(32)
	app.Type = nodeps.AppTypeDrupal8

	// WriteConfig the app.
	err = app.WriteConfig()
	assert.NoError(err)
	_, err = os.Stat(app.ConfigPath)
	assert.NoError(err)

	loadedConfig, err := NewApp(testDir, true, nodeps.ProviderDefault)
	assert.NoError(err)
	assert.Equal(app.Name, loadedConfig.Name)
	assert.Equal(app.Type, loadedConfig.Type)
}

// TestDisasterConfig tests for disaster opportunities (configing wrong directory, home dir, etc).
func TestDisasterConfig(t *testing.T) {
	assert := asrt.New(t)

	testDir, _ := os.Getwd()

	// Make sure we're not allowed to config in home directory.
	tmpDir, _ := homedir.Dir()
	_, err := NewApp(tmpDir, false, nodeps.ProviderDefault)
	assert.Error(err)
	assert.Contains(err.Error(), "ddev config is not useful in home directory")
	_ = os.Chdir(testDir)

	// Create a temporary directory and change to it for the duration of this test.
	tmpDir = testcommon.CreateTmpDir("TestDisasterConfig")

	defer testcommon.CleanupDir(tmpDir)
	defer testcommon.Chdir(tmpDir)()

	// Load a new Config
	app, err := NewApp(tmpDir, false, nodeps.ProviderDefault)
	assert.NoError(err)

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
	subdirApp, err := NewApp(subdir, false, nodeps.ProviderDefault)
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
	// Create a temporary directory and change to it for the duration of this test.
	testDir := testcommon.CreateTmpDir("TestPrepDirectory")
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	app, err := NewApp(testDir, true, nodeps.ProviderDefault)
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
	app, err := NewApp(testDir, true, nodeps.ProviderDefault)
	assert.NoError(err)
	app.Name = util.RandString(32)

	assert.Equal(app.GetHostname(), app.Name+"."+app.ProjectTLD)
}

// TestWriteDockerComposeYaml tests the writing of a docker-compose.yaml file.
func TestWriteDockerComposeYaml(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir("TestWriteDockerCompose")
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	app, err := NewApp(testDir, true, nodeps.ProviderDefault)
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
		"drupal6phpversion": {nodeps.AppTypeDrupal6, nodeps.PHP56},
		"drupal7phpversion": {nodeps.AppTypeDrupal7, nodeps.PHP72},
		"drupal8phpversion": {nodeps.AppTypeDrupal8, nodeps.PHP73},
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
		app, err := NewApp(testDir, true, nodeps.ProviderDefault)
		assert.NoError(err)

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
		// a valid document root,
		// a valid app type
		input = fmt.Sprintf("invalid_project_name\n%s\ndocroot\n%s", name, testValues[apptypePos])
		scanner = bufio.NewScanner(strings.NewReader(input))
		util.SetInputScanner(scanner)

		restoreOutput = util.CaptureUserOut()
		err = app.PromptForConfig()
		assert.NoError(err, t)
		out = restoreOutput()

		// Ensure we have expected vales in output.
		assert.Contains(out, testDir)
		assert.Contains(out, "invalid_project_name is not a valid project name")

		// Ensure values were properly set on the app struct.
		assert.Equal(name, app.Name)
		assert.Equal(testValues[apptypePos], app.Type)
		assert.Equal("docroot", app.Docroot)
		assert.EqualValues(testValues[phpVersionPos], app.PHPVersion, "PHP value incorrect for app %v", app)
		err = PrepDdevDirectory(testDir)
		assert.NoError(err)
	}
}

// TestConfigCommandInteractiveCreateDocrootDenied
func TestConfigCommandInteractiveCreateDocrootDenied(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)

	noninteractiveEnv := "DRUD_NONINTERACTIVE"
	// nolint: errcheck
	defer os.Setenv(noninteractiveEnv, os.Getenv(noninteractiveEnv))
	err := os.Unsetenv(noninteractiveEnv)
	assert.NoError(err)

	testMatrix := map[string][]string{
		"drupal6phpversion": {nodeps.AppTypeDrupal6, nodeps.PHP56},
		"drupal7phpversion": {nodeps.AppTypeDrupal7, nodeps.PHP72},
		"drupal8phpversion": {nodeps.AppTypeDrupal8, nodeps.PHP73},
	}

	for testName := range testMatrix {
		testDir := testcommon.CreateTmpDir(t.Name() + testName)

		// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
		defer testcommon.CleanupDir(testDir)
		defer testcommon.Chdir(testDir)()

		// Create the ddevapp we'll use for testing.
		// This will not return an error, since there is no existing configuration.
		app, err := NewApp(testDir, true, nodeps.ProviderDefault)
		assert.NoError(err)

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

		err = PrepDdevDirectory(testDir)
		assert.NoError(err)
	}
}

// TestConfigCommandCreateDocrootAllowed
func TestConfigCommandCreateDocrootAllowed(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)

	const apptypePos = 0
	const phpVersionPos = 1
	testMatrix := map[string][]string{
		"drupal6phpversion": {nodeps.AppTypeDrupal6, nodeps.PHP56},
		"drupal7phpversion": {nodeps.AppTypeDrupal7, nodeps.PHP72},
		"drupal8phpversion": {nodeps.AppTypeDrupal8, nodeps.PHP73},
	}

	for testName, testValues := range testMatrix {
		testDir := testcommon.CreateTmpDir(t.Name() + testName)

		// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
		defer testcommon.CleanupDir(testDir)
		defer testcommon.Chdir(testDir)()

		// Create the ddevapp we'll use for testing.
		// This will not return an error, since there is no existing configuration.
		app, err := NewApp(testDir, true, nodeps.ProviderDefault)
		assert.NoError(err)

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
		_, err = os.OpenFile(filepath.Join(testDir, filepath.Join(testDocrootName), "index.php"), os.O_RDONLY|os.O_CREATE, 0664)
		assert.NoError(err)

		// Create the ddevapp we'll use for testing.
		// This will not return an error, since there is no existing configuration.
		app, err := NewApp(testDir, true, nodeps.ProviderDefault)
		assert.NoError(err)

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
	_, err = os.OpenFile(filepath.Join(testDir, "docroot", "index.php"), os.O_RDONLY|os.O_CREATE, 0664)
	assert.NoError(err)

	// Create the ddevapp we'll use for testing.
	// This will not return an error, since there is no existing configuration.
	app, err := NewApp(testDir, true, nodeps.ProviderDefault)
	assert.NoError(err)

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
	err = PrepDdevDirectory(testDir)
	assert.NoError(err)
}

// TestReadConfig tests reading config values from file and fallback to defaults for values not exposed.
func TestReadConfig(t *testing.T) {
	assert := asrt.New(t)

	// This closely resembles the values one would have from NewApp()
	app := &DdevApp{
		APIVersion: version.DdevVersion,
		ConfigPath: filepath.Join("testdata", "config.yaml"),
		AppRoot:    "testdata",
		Name:       "TestRead",
		Provider:   nodeps.ProviderDefault,
	}

	_, err := app.ReadConfig(false)
	if err != nil {
		t.Fatalf("Unable to c.ReadConfig(), err: %v", err)
	}

	// Values not defined in file, we should still have default values
	assert.Equal(app.Name, "TestRead")
	assert.Equal(app.APIVersion, version.DdevVersion)

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
		APIVersion: version.DdevVersion,
		ConfigPath: filepath.Join("testdata", t.Name(), ".ddev", "config.yaml"),
		AppRoot:    filepath.Join("testdata", t.Name()),
		Name:       t.Name(),
		Provider:   nodeps.ProviderDefault,
	}

	_, err := app.ReadConfig(false)
	if err != nil {
		t.Fatalf("Unable to c.ReadConfig(), err: %v", err)
	}

	// Values not defined in file, we should still have default values
	assert.Equal(app.Name, t.Name())
	assert.Equal(app.APIVersion, version.DdevVersion)

	// Values defined in file, we should have values from file
	assert.Equal(app.Docroot, "public")
}

// TestConfigValidate tests validation of configuration values.
func TestConfigValidate(t *testing.T) {
	assert := asrt.New(t)

	cwd, err := os.Getwd()
	assert.NoError(err)

	app := &DdevApp{
		Name:           "TestConfigValidate",
		ConfigPath:     filepath.Join("testdata", "config.yaml"),
		AppRoot:        cwd,
		Docroot:        "testdata",
		Type:           nodeps.AppTypeWordPress,
		PHPVersion:     nodeps.PHPDefault,
		MariaDBVersion: version.MariaDBDefaultVersion,
		WebserverType:  nodeps.WebserverDefault,
		ProjectTLD:     nodeps.DdevDefaultTLD,
		Provider:       nodeps.ProviderDefault,
	}

	err = app.ValidateConfig()
	if err != nil {
		t.Fatalf("Failed to app.ValidateConfig(), err=%v", err)
	}

	app.Name = "Invalid!"
	err = app.ValidateConfig()
	assert.Error(err)
	assert.Contains(err.Error(), "not a valid project name")

	app.Docroot = "testdata"
	app.Name = "valid"
	app.Type = "potato"
	err = app.ValidateConfig()
	assert.Error(err)
	assert.Contains(err.Error(), "invalid app type")

	app.Type = nodeps.AppTypeWordPress
	app.PHPVersion = "1.1"
	err = app.ValidateConfig()
	assert.Error(err)
	assert.Contains(err.Error(), "invalid PHP")

	app.PHPVersion = nodeps.PHPDefault
	app.WebserverType = "server"
	err = app.ValidateConfig()
	assert.Error(err)
	assert.Contains(err.Error(), "invalid webserver type")

	app.WebserverType = nodeps.WebserverDefault
	app.AdditionalHostnames = []string{"good", "b@d"}
	err = app.ValidateConfig()
	assert.Error(err)
	assert.Contains(err.Error(), "invalid hostname")

	app.AdditionalHostnames = []string{}
	app.AdditionalFQDNs = []string{"good.com", "b@d.com"}
	err = app.ValidateConfig()
	assert.Error(err)
	assert.Contains(err.Error(), "invalid hostname")

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
}

// TestWriteConfig tests writing config values to file
func TestWriteConfig(t *testing.T) {
	assert := asrt.New(t)

	testDir, _ := os.Getwd()
	//nolint: errcheck
	defer os.Chdir(testDir)

	projDir, err := filepath.Abs(testcommon.CreateTmpDir("TestWriteConfig"))
	require.NoError(t, err)

	err = fileutil.CopyDir("./testdata/TestWriteConfig/.ddev", filepath.Join(projDir, ".ddev"))
	require.NoError(t, err)
	//nolint: errcheck
	defer os.RemoveAll(projDir)

	app, err := NewApp(projDir, true, "")
	assert.NoError(err)
	err = os.Chdir(projDir)
	assert.NoError(err)

	// The default NewApp read should read config overrides, so we should have "config.extra.yaml"
	// as the APIVersion
	assert.Equal("config.extra.yaml", app.APIVersion)

	// However, if we ReadConfig() without includeOverrides, we should get "config.yaml" as the APIVersion
	_, err = app.ReadConfig(false)
	assert.NoError(err)
	assert.Equal("config.yaml", app.APIVersion)

	err = app.WriteConfig()
	assert.NoError(err)

	// Now read the config we just wrote; it should have config.yaml because ignored overrides.
	_, err = app.ReadConfig(false)
	assert.NoError(err)
	// app.WriteConfig() writes the version.DdevVersion to the updated config.yaml
	assert.Equal(version.DdevVersion, app.APIVersion)
}

// TestConfigOverrideDetection tests to make sure we tell them about config overrides.
func TestConfigOverrideDetection(t *testing.T) {
	assert := asrt.New(t)
	app := &DdevApp{}
	testDir, _ := os.Getwd()

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()

	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s ConfigOverrideDetection", site.Name))

	// Copy test overrides into the project .ddev directory
	for _, item := range []string{"nginx", "apache", "php", "mysql"} {
		err := fileutil.CopyDir(filepath.Join(testDir, "testdata/TestConfigOverrideDetection/.ddev", item), filepath.Join(site.Dir, ".ddev", item))
		assert.NoError(err)
		err = fileutil.CopyFile(filepath.Join(testDir, "testdata/TestConfigOverrideDetection/.ddev", "nginx-site.conf"), filepath.Join(site.Dir, ".ddev", "nginx-site.conf"))
		assert.NoError(err)
	}

	// And when we're done, we have to clean those out again.
	defer func() {
		for _, item := range []string{"apache", "php", "mysql", "nginx", "nginx-site.conf"} {
			_ = os.RemoveAll(filepath.Join(".ddev", item))
		}
	}()
	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)

	restoreOutput := util.CaptureUserOut()
	startErr := app.StartAndWait(2)
	out := restoreOutput()
	//nolint: errcheck
	defer app.Stop(true, false)

	var logs string
	if startErr != nil {
		logs, _ = GetErrLogsFromApp(app, startErr)
	}

	require.NoError(t, startErr, "app.StartAndWait() did not succeed: output:\n=====\n%s\n===== logs:\n========= logs =======\n%s\n========\n", out, logs)

	assert.Contains(out, "utf.cnf")
	assert.Contains(out, "my-php.ini")

	switch app.WebserverType {
	case nodeps.WebserverNginxFPM:
		assert.Contains(out, "nginx-site.conf")
		assert.NotContains(out, "apache-site.conf")
		assert.Contains(out, "junker99.conf")
	default:
		assert.Contains(out, "apache-site.conf")
		assert.NotContains(out, "nginx-site.conf")
	}
	assert.Contains(out, "Custom configuration takes effect")
	runTime()
}

// TestPHPOverrides tests to make sure that PHP overrides work in all webservers.
func TestPHPOverrides(t *testing.T) {
	assert := asrt.New(t)
	app := &DdevApp{}
	testDir, _ := os.Getwd()

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()

	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s %s", site.Name, t.Name()))

	// Copy test overrides into the project .ddev directory
	err := fileutil.CopyDir(filepath.Join(testDir, "testdata/TestPHPOverrides/.ddev/php"), filepath.Join(site.Dir, ".ddev/php"))
	assert.NoError(err)
	err = fileutil.CopyFile(filepath.Join(testDir, "testdata/TestPHPOverrides/phpinfo.php"), filepath.Join(site.Dir, site.Docroot, "phpinfo.php"))
	assert.NoError(err)

	// And when we're done, we have to clean those out again.
	defer func() {
		_ = os.RemoveAll(filepath.Join(site.Dir, ".ddev/php"))
		_ = os.RemoveAll(filepath.Join(site.Dir, "phpinfo.php"))
	}()

	for _, webserverType := range []string{nodeps.WebserverNginxFPM, nodeps.WebserverApacheFPM, nodeps.WebserverApacheCGI} {
		testcommon.ClearDockerEnv()
		app.WebserverType = webserverType
		err = app.Init(site.Dir)
		assert.NoError(err)
		_ = app.Stop(true, false)
		// nolint: errcheck
		defer app.Stop(true, false)
		startErr := app.StartAndWait(5)
		if startErr != nil {
			logs, _ := GetErrLogsFromApp(app, startErr)
			t.Logf("failed app.StartAndWait(): %v", startErr)
			t.Fatalf("============== logs from app.StartAndWait() ==============\n%s\n", logs)
		}

		_, _ = testcommon.EnsureLocalHTTPContent(t, "http://"+app.GetHostname()+"/phpinfo.php", `max_input_time</td><td class="v">999`)
		err = app.Stop(true, false)
		assert.NoError(err)
	}

	runTime()
}

// TestExtraPackages tests to make sure that *extra_packages config.yaml directives
// work (and are overridden by *-build/Dockerfile).
func TestExtraPackages(t *testing.T) {
	assert := asrt.New(t)
	app := &DdevApp{}

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()

	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s TestExtraPackages", site.Name))

	err := app.Init(site.Dir)
	assert.NoError(err)
	err = app.Stop(true, false)
	assert.NoError(err)

	defer func() {
		app.WebImageExtraPackages = nil
		app.DBImageExtraPackages = nil
		_ = app.WriteConfig()
		_ = os.RemoveAll(app.GetConfigPath("web-build"))
		_ = os.RemoveAll(app.GetConfigPath("db-build"))
		_ = app.Stop(true, false)
	}()

	// Start and make sure that the packages don't exist already
	err = app.Start()
	assert.NoError(err)

	_, _, err = app.Exec(&ExecOpts{
		Service: "web",
		Cmd:     "command -v zsh",
	})
	assert.Error(err)
	assert.Contains(err.Error(), "exit status 1")

	_, _, err = app.Exec(&ExecOpts{
		Service: "db",
		Cmd:     "command -v ncdu",
	})
	assert.Error(err)
	assert.Contains(err.Error(), "exit status 1")

	// Now add the packages and start again, they should be in there
	app.WebImageExtraPackages = []string{"zsh"}
	app.DBImageExtraPackages = []string{"ncdu"}
	err = app.Start()
	assert.NoError(err)

	stdout, _, err := app.Exec(&ExecOpts{
		Service: "web",
		Cmd:     "command -v zsh",
	})
	assert.NoError(err)
	assert.Equal("/usr/bin/zsh", strings.Trim(stdout, "\n"))

	stdout, _, err = app.Exec(&ExecOpts{
		Service: "db",
		Cmd:     "command -v ncdu",
	})
	assert.NoError(err)
	assert.Equal("/usr/bin/ncdu", strings.Trim(stdout, "\n"))

	err = app.Stop(true, false)
	assert.NoError(err)

	runTime()
}

// TestTimezoneConfig tests to make sure setting timezone config takes effect in the container.
func TestTimezoneConfig(t *testing.T) {
	assert := asrt.New(t)
	app := &DdevApp{}

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()

	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s %s", t.Name(), site.Name))

	err := app.Init(site.Dir)
	assert.NoError(err)
	err = app.Stop(true, false)
	assert.NoError(err)

	// nolint: errcheck
	defer app.Stop(true, false)

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
	assert.Equal("timezone=CET\nphptz=Europe/Paris", stdout)

	// Make sure db container is also working with Dublin time/IST
	stdout, _, err = app.Exec(&ExecOpts{
		Service: "db",
		Cmd:     "echo -n timezone=$(date +%Z)",
	})
	assert.NoError(err)
	assert.Equal("timezone=CET", stdout)

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

	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s TestCustomBuildDockerfiles", site.Name))

	err := app.Init(site.Dir)
	assert.NoError(err)
	err = app.Stop(true, false)
	assert.NoError(err)

	defer func() {
		_ = os.RemoveAll(app.GetConfigPath("web-build"))
		_ = os.RemoveAll(app.GetConfigPath("db-build"))
		_ = app.Stop(true, false)
	}()

	// Create simple dockerfiles that just touch /var/tmp/added-by-<container>txt
	for _, item := range []string{"web", "db"} {
		err = WriteImageDockerfile(app.GetConfigPath(item+"-build/Dockerfile"), []byte(`
ARG BASE_IMAGE
FROM $BASE_IMAGE
RUN touch /var/tmp/`+"added-by-"+item+".txt"))
		assert.NoError(err)
	}
	// Start and make sure that the packages don't exist already
	err = app.Start()
	assert.NoError(err)

	// Make sure that the expected in-container file has been created
	for _, item := range []string{"web", "db"} {
		_, _, err = app.Exec(&ExecOpts{
			Service: item,
			Cmd:     "ls /var/tmp/added-by-" + item + ".txt",
		})
		assert.NoError(err)
	}

	err = app.Stop(true, false)
	assert.NoError(err)

	runTime()
}

// TestConfigLoadingOrder verifies that configs load in lexicographical order
// AFTER config.yaml
func TestConfigLoadingOrder(t *testing.T) {
	assert := asrt.New(t)
	testDir, _ := os.Getwd()
	//nolint: errcheck
	defer os.Chdir(testDir)

	projDir, err := filepath.Abs(testcommon.CreateTmpDir("TestConfigLoadingOrder"))
	require.NoError(t, err)

	err = fileutil.CopyDir("./testdata/TestConfigLoadingOrder/.ddev", filepath.Join(projDir, ".ddev"))
	require.NoError(t, err)
	//nolint: errcheck
	defer os.RemoveAll(projDir)

	app, err := NewApp(projDir, true, "")
	require.NoError(t, err)
	err = os.Chdir(app.AppRoot)
	assert.NoError(err)
	_, err = app.ReadConfig(true)
	assert.NoError(err)
	assert.Equal("config.yaml", app.APIVersion)

	matches, err := filepath.Glob(filepath.Join(projDir, ".ddev/linkedconfigs/config.*.y*ml"))
	assert.NoError(err)

	// First make sure that each possible item in .ddev/linkedconfigs can override by itself
	for _, item := range matches {
		linkedMatch := filepath.Join(app.AppConfDir(), filepath.Base(item))
		assert.NoError(err)
		err = os.Symlink(item, linkedMatch)
		assert.NoError(err)
		_, err = app.ReadConfig(true)
		assert.NoError(err)
		assert.Equal(filepath.Base(item), app.APIVersion)
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
		_, err = app.ReadConfig(true)
		assert.Equal(filepath.Base(item), app.APIVersion)
	}

	// Now we still have all those linked overrides, but do a ReadConfig() without allowing them
	// and verify that they don't get loaded
	_, err = app.ReadConfig(false)
	assert.NoError(err)
	assert.Equal("config.yaml", app.APIVersion)

}

func TestPkgConfigMariaDBVersion(t *testing.T) {
	// NewApp from scratch
	// NewApp with config.yaml with mariadb_version and no dbimage
	// NewApp with config.yaml with dbimage and no mariadb_version
	// NewApp with both dbimage and
	assert := asrt.New(t)

	testDir, _ := os.Getwd()

	// Create a temporary directory and switch to it.
	tmpDir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(tmpDir)
	defer testcommon.Chdir(tmpDir)()

	systemTempDir, _ := testcommon.OsTempDir()

	targetBase := filepath.Join(systemTempDir, "TestPkgConfigMariaDBVersion")
	_ = os.RemoveAll(targetBase)
	err := fileutil.CopyDir(filepath.Join(testDir, "testdata", "TestPkgConfigMariaDBVersion"), targetBase)
	require.NoError(t, err)

	mariaDBVersions := nodeps.ValidMariaDBVersions
	for v := range mariaDBVersions {
		for _, configType := range []string{"dbimage", "mariadb-version"} {
			app := &DdevApp{}
			appRoot := filepath.Join(targetBase, configType+"-"+v)
			err = app.LoadConfigYamlFile(filepath.Join(appRoot, ".ddev", "config.yaml"))
			assert.NoError(err)
			if configType == "dbimage" {
				assert.Equal("somedbimage:"+v, app.DBImage)
			}
			if configType == "mariadb-version" {
				assert.Equal(v, app.MariaDBVersion)
			}

			app, err = NewApp(appRoot, false, "")
			assert.NoError(err)
			if configType == "dbimage" {
				assert.Equal("somedbimage:"+v, app.DBImage, "NewApp() failed to respect existing dbimage")
			}
			if configType == "mariadb-version" {
				assert.Equal(v, app.MariaDBVersion)
				assert.Equal(version.GetDBImage(nodeps.MariaDB, v), app.GetDBImage(), "dbimage derived from app.MariaDBVersion was incorrect")
			}

		}
	}

}
