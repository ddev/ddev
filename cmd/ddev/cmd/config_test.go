package cmd

import (
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/version"
	"testing"

	"os"
	"path/filepath"

	"strings"

	"fmt"

	"io/ioutil"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

// TestConfigDescribeLocation tries out the --show-config-location flag.
func TestConfigDescribeLocation(t *testing.T) {
	assert := asrt.New(t)

	// Create a temporary directory and switch to it.
	tmpdir := testcommon.CreateTmpDir("config-show-location")
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()

	// Create a config
	args := []string{"config", "--docroot=."}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Found a php codebase")

	// Now see if we can detect it
	args = []string{"config", "--show-config-location"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(string(out), tmpdir)

	// Now try it in a directory that doesn't have a config
	tmpdir = testcommon.CreateTmpDir("config_show_location")
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()

	args = []string{"config", "--show-config-location"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "No project configuration currently exists")

}

// TestConfigWithSitenameFlagDetectsDocroot tests docroot detected when flags passed.
func TestConfigWithSitenameFlagDetectsDocroot(t *testing.T) {
	assert := asrt.New(t)

	// Create a temporary directory and switch to it.
	testDocrootName := "web"
	tmpdir := testcommon.CreateTmpDir("config-with-sitename")
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()
	// Create a document root folder.
	err := os.MkdirAll(filepath.Join(tmpdir, testDocrootName), 0755)
	if err != nil {
		t.Errorf("Could not create %s directory under %s", testDocrootName, tmpdir)
	}
	err = os.MkdirAll(filepath.Join(tmpdir, testDocrootName, "sites", "default"), 0755)
	assert.NoError(err)
	_, err = os.OpenFile(filepath.Join(tmpdir, testDocrootName, "index.php"), os.O_RDONLY|os.O_CREATE, 0666)
	assert.NoError(err)

	expectedPath := "web/core/scripts/drupal.sh"
	err = os.MkdirAll(filepath.Join(tmpdir, filepath.Dir(expectedPath)), 0777)
	assert.NoError(err)

	_, err = os.OpenFile(filepath.Join(tmpdir, expectedPath), os.O_RDONLY|os.O_CREATE, 0666)
	assert.NoError(err)

	// Create a config
	args := []string{"config", "--sitename=config-with-sitename"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Found a drupal8 codebase")
}

// TestConfigSetValues sets all available configuration values using command flags, then confirms that the
// values have been correctly written to the config file.
func TestConfigSetValues(t *testing.T) {
	var err error
	assert := asrt.New(t)

	// Create a temporary directory and switch to it.
	tmpdir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()

	// Create an existing docroot
	docroot := "web"
	if err = os.MkdirAll(filepath.Join(tmpdir, docroot), 0755); err != nil {
		t.Errorf("Could not create docroot %s in %s", docroot, tmpdir)
	}

	// Build config args
	projectName := "my-project-name"
	projectType := ddevapp.AppTypeTYPO3
	phpVersion := ddevapp.PHP71
	httpPort := "81"
	httpsPort := "444"
	hostDBPort := "60001"
	hostWebserverPort := "60002"
	xdebugEnabled := true
	additionalHostnamesSlice := []string{"abc", "123", "xyz"}
	additionalHostnames := strings.Join(additionalHostnamesSlice, ",")
	additionalFQDNsSlice := []string{"abc.com", "123.pizza", "xyz.co.uk"}
	additionalFQDNs := strings.Join(additionalFQDNsSlice, ",")
	omitContainersSlice := []string{"dba", "ddev-ssh-agent"}
	omitContainers := strings.Join(omitContainersSlice, ",")

	uploadDir := filepath.Join("custom", "config", "path")
	webserverType := ddevapp.WebserverApacheFPM
	webImage := "custom-web-image"
	dbImage := "custom-db-image"
	dbaImage := "custom-dba-image"
	webWorkingDir := "/custom/web/dir"
	dbWorkingDir := "/custom/db/dir"
	dbaWorkingDir := "/custom/dba/dir"

	args := []string{
		"config",
		"--project-name", projectName,
		"--docroot", docroot,
		"--project-type", projectType,
		"--php-version", phpVersion,
		"--http-port", httpPort,
		"--https-port", httpsPort,
		fmt.Sprintf("--xdebug-enabled=%t", xdebugEnabled),
		"--additional-hostnames", additionalHostnames,
		"--additional-fqdns", additionalFQDNs,
		"--upload-dir", uploadDir,
		"--webserver-type", webserverType,
		"--web-image", webImage,
		"--db-image", dbImage,
		"--dba-image", dbaImage,
		"--web-working-dir", webWorkingDir,
		"--db-working-dir", dbWorkingDir,
		"--dba-working-dir", dbaWorkingDir,
		"--omit-containers", omitContainers,
		"--host-db-port", hostDBPort,
		"--host-webserver-port", hostWebserverPort,
	}

	_, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)

	configFile := filepath.Join(tmpdir, ".ddev", "config.yaml")
	configContents, err := ioutil.ReadFile(configFile)
	if err != nil {
		t.Errorf("Unable to read %s: %v", configFile, err)
	}

	app := &ddevapp.DdevApp{}
	if err = yaml.Unmarshal(configContents, app); err != nil {
		t.Errorf("Could not unmarshal %s: %v", configFile, err)
	}

	assert.Equal(projectName, app.Name)
	assert.Equal(docroot, app.Docroot)
	assert.Equal(projectType, app.Type)
	assert.Equal(phpVersion, app.PHPVersion)
	assert.Equal(httpPort, app.RouterHTTPPort)
	assert.Equal(httpsPort, app.RouterHTTPSPort)
	assert.Equal(hostWebserverPort, app.HostWebserverPort)
	assert.Equal(hostDBPort, app.HostDBPort)
	assert.Equal(xdebugEnabled, app.XdebugEnabled)
	assert.Equal(additionalHostnamesSlice, app.AdditionalHostnames)
	assert.Equal(additionalFQDNsSlice, app.AdditionalFQDNs)
	assert.Equal(uploadDir, app.UploadDir)
	assert.Equal(webserverType, app.WebserverType)
	assert.Equal(webImage, app.WebImage)
	assert.Equal(dbImage, app.DBImage)
	assert.Equal(dbaImage, app.DBAImage)
	assert.Equal(webWorkingDir, app.WorkingDir["web"])
	assert.Equal(dbWorkingDir, app.WorkingDir["db"])
	assert.Equal(dbaWorkingDir, app.WorkingDir["dba"])

	// Test that container images and working dirs can be unset with default flags
	args = []string{
		"config",
		"--web-image-default",
		"--db-image-default",
		"--dba-image-default",
		"--web-working-dir-default",
		"--db-working-dir-default",
		"--dba-working-dir-default",
	}

	_, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)

	configContents, err = ioutil.ReadFile(configFile)
	assert.NoError(err, "Unable to read %s: %v", configFile, err)

	app = &ddevapp.DdevApp{}
	err = yaml.Unmarshal(configContents, app)
	assert.NoError(err, "Could not unmarshal %s: %v", configFile, err)

	assert.Equal(app.WebImage, "")
	assert.Equal(app.DBImage, "")
	assert.Equal(app.DBAImage, "")
	assert.Equal(len(app.WorkingDir), 0)

	// Test that all container images and working dirs can each be unset with single default images flag
	args = []string{
		"config",
		"--web-image", webImage,
		"--db-image", dbImage,
		"--dba-image", dbaImage,
		"--web-working-dir", webWorkingDir,
		"--db-working-dir", dbWorkingDir,
		"--dba-working-dir", dbaWorkingDir,
	}

	_, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)

	args = []string{
		"config",
		"--image-defaults",
		"--working-dir-defaults",
	}

	_, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)

	configContents, err = ioutil.ReadFile(configFile)
	assert.NoError(err, "Unable to read %s: %v", configFile, err)

	app = &ddevapp.DdevApp{}
	err = yaml.Unmarshal(configContents, app)
	assert.NoError(err, "Could not unmarshal %s: %v", configFile, err)

	assert.Equal(app.WebImage, "")
	assert.Equal(app.DBImage, "")
	assert.Equal(app.DBAImage, "")
	assert.Equal(len(app.WorkingDir), 0)
}

// TestConfigInvalidProjectname tests to make sure that invalid projectnames
// are not accepted and valid names are accepted.
func TestConfigInvalidProjectname(t *testing.T) {
	var err error
	assert := asrt.New(t)

	// Create a temporary directory and switch to it.
	tmpdir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()

	// Create an existing docroot
	docroot := "web"
	if err = os.MkdirAll(filepath.Join(tmpdir, docroot), 0755); err != nil {
		t.Errorf("Could not create docroot %s in %s", docroot, tmpdir)
	}

	// Test some valid project names
	for _, projName := range []string{"no-spaces-but-hyphens", "UpperAndLower", "should.work.with.dots"} {
		args := []string{
			"config",
			"--project-name", projName,
		}

		out, err := exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.NotContains(out, "is not a valid project name")
		assert.Contains(out, "You may now run 'ddev start'")
		_ = os.Remove(filepath.Join(tmpdir, ".ddev", "config.yaml"))
	}

	// test some invalid project names.
	for _, projName := range []string{"with spaces", "with_underscores", "no,commas-will-make-it"} {
		args := []string{
			"config",
			"--project-name", projName,
		}

		out, err := exec.RunCommand(DdevBin, args)
		assert.Error(err)
		assert.Contains(out, fmt.Sprintf("%s is not a valid project name", projName))
		assert.NotContains(out, "You may now run 'ddev start'")
		_ = os.Remove(filepath.Join(tmpdir, ".ddev", "config.yaml"))
	}
}

// TestConfigSubdir ensures an existing config can be found from subdirectories
func TestConfigSubdir(t *testing.T) {
	var err error
	assert := asrt.New(t)

	// Create a temporary directory and switch to it.
	tmpdir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()

	// Create a simple config.
	args := []string{
		"config",
		"--project-type",
		"php",
	}

	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(out, "You may now run 'ddev start'")

	// Create a subdirectory and switch to it.
	subdir := filepath.Join(tmpdir, "some", "sub", "dir")
	err = os.MkdirAll(subdir, 0755)
	assert.NoError(err)
	defer testcommon.Chdir(subdir)()

	// Confirm that the detected config location is in the original dir.
	args = []string{
		"config",
		"--show-config-location",
	}

	// Ensure the existing config can be found
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(out, filepath.Join(tmpdir, ".ddev", "config.yaml"))
}

/**
  	Overall expected behavior of `ddev config` and config.yaml is this:
	* If no config.yaml exists when `ddev config` is run, a new one will be created with default
		values except whatever is specified in the ddev config args.
	* If an item exists in config.yaml, it will be rewritten there after a `ddev config` action
	* If the `ddev config` action overrides explicitly the item in config.yaml, the config.yaml
		will be rewritten with the updated value and no other changes.

	In the case of mariadb_version and dbimage there's quite a lot of overlap sadly, but
	the bottom line is that dbimage must win any battles, and if specified either via
	`ddev config --db-image` or in the config.yaml, it wins when the dbimage is chosen for the
	ddevapp. So:
	* `ddev config --mariadb-version` must
		* Never add a dbimage line to the config.yaml
		* Add or update the mariadb_version line in config.yaml
	* `ddev config --db-image` must
		* Not change the mariadb_version (although people may be broken if they do it wrong)
		* Add a line specifying the dbimage
	* ddevapp.NewApp() must:
		* Set the dbimage based on any dbimage found in the config.yaml if there is one
		* Otherwise (default case) set it based on mariadb_version and the default dbimage.
*/
// TestConfigMariaDBVersion checks to make sure that both
// ddev config --mariadb-version and ddev config --dbimage behave correctly,
// either separately or together.
func TestConfigMariaDBVersion(t *testing.T) {

	assert := asrt.New(t)

	testDir, _ := os.Getwd()
	versionsToTest := []string{"10.1", "10.2"}

	// Create a temporary directory and switch to it.
	tmpDir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(tmpDir)
	defer testcommon.Chdir(tmpDir)()

	args := []string{
		"config",
		"--mariadb-version",
	}

	// Verify behavior with no existing config.yaml. It should
	// just add mariadb_version into the config and nothing else
	// The app.DBImage should end up being the constructed one based
	// on the default and the mariadb_version
	for _, cmdMariaDBVersion := range versionsToTest {
		_ = os.RemoveAll(filepath.Join(tmpDir, ".ddev"))
		configArgs := append(args, cmdMariaDBVersion, "--project-name", "noconfigyet-"+cmdMariaDBVersion)
		ddevCmd := "ddev " + strings.Join(configArgs, " ")
		out, err := exec.RunCommand(DdevBin, configArgs)
		assert.NoError(err)
		assert.Contains(out, "You may now run 'ddev start'")

		// First test the bare explicit values found in the config.yaml,
		// without the NewApp adjustments
		app := &ddevapp.DdevApp{}
		assert.NoError(err)
		err = app.LoadConfigYamlFile(filepath.Join(tmpDir, ".ddev", "config.yaml"))
		assert.NoError(err)
		assert.Equal(cmdMariaDBVersion, app.MariaDBVersion)
		assert.Empty(app.DBImage, "generated config.yaml dbimage should have been empty for command '%s'", ddevCmd)

		// Now use NewApp() to load, so that we get the full logic of that function.
		app, err = ddevapp.NewApp(tmpDir, false, "")
		//nolint: errcheck
		defer app.Down(true, false)
		assert.NoError(err)
		_, err = app.ReadConfig(false)
		assert.NoError(err)
		assert.Equal(cmdMariaDBVersion, app.MariaDBVersion)
		assert.Equal(version.DBImg+":"+version.BaseDBTag+"-"+cmdMariaDBVersion, app.DBImage)
	}

	// If we start with a config.yaml specifying basically anything for mariadb_version
	// using ddev config --mariadb-version should overwrite it with the specified version.
	for _, cmdMariaDBVersion := range versionsToTest {
		for _, configMariaDBVersion := range versionsToTest {
			_ = os.Remove(filepath.Join(tmpDir, ".ddev", "config.yaml"))

			err := fileutil.CopyFile(filepath.Join(testDir, "testdata/TestConfigMariaDBVersion", "config.yaml."+configMariaDBVersion), filepath.Join(tmpDir, ".ddev", "config.yaml"))
			assert.NoError(err)
			configArgs := append(args, cmdMariaDBVersion, "--project-name", "hasconfig-"+cmdMariaDBVersion)
			out, err := exec.RunCommand(DdevBin, configArgs)
			assert.NoError(err)
			assert.Contains(out, "You may now run 'ddev start'")

			// First test the bare explicit values found in the config.yaml,
			// without the NewApp adjustments
			app := &ddevapp.DdevApp{}
			assert.NoError(err)
			err = app.LoadConfigYamlFile(filepath.Join(tmpDir, ".ddev", "config.yaml"))
			assert.NoError(err)
			assert.Equal(cmdMariaDBVersion, app.MariaDBVersion)
			assert.Empty(app.DBImage)

			// Now test with the logical additions made by NewApp()
			app, err = ddevapp.NewApp(tmpDir, false, "")
			//nolint: errcheck
			defer app.Down(true, false)
			assert.NoError(err)
			_, err = app.ReadConfig(false)
			assert.NoError(err)
			assert.Equal(cmdMariaDBVersion, app.MariaDBVersion)
			assert.Equal(version.DBImg+":"+version.BaseDBTag+"-"+cmdMariaDBVersion, app.DBImage)
		}
	}

	// If we start with a config.yaml specifying a dbimage, the mariadb_version
	// should be set but the dbimage should then override all other values
	for _, cmdMariaDBVersion := range versionsToTest {
		for _, configMariaDBVersion := range versionsToTest {
			_ = os.Remove(filepath.Join(tmpDir, ".ddev", "config.yaml"))

			err := fileutil.CopyFile(filepath.Join(testDir, "testdata/TestConfigMariaDBVersion", "config.yaml.imagespec."+configMariaDBVersion), filepath.Join(tmpDir, ".ddev", "config.yaml"))
			assert.NoError(err)
			configArgs := append(args, cmdMariaDBVersion, "--project-name", "imagespec-"+cmdMariaDBVersion)
			out, err := exec.RunCommand(DdevBin, configArgs)
			assert.NoError(err)
			assert.Contains(out, "You may now run 'ddev start'")

			// First test the bare explicit values found in the config.yaml,
			// without the NewApp adjustments
			app := &ddevapp.DdevApp{}
			assert.NoError(err)
			err = app.LoadConfigYamlFile(filepath.Join(tmpDir, ".ddev", "config.yaml"))
			assert.NoError(err)
			assert.Equal(cmdMariaDBVersion, app.MariaDBVersion)
			// The ddev config --mariadb-version should *not* have changed the dbimg
			// which was in the config.yaml
			assert.Equal("somerandomdbimg:v0.9999-"+configMariaDBVersion, app.DBImage, "ddev %s did not result in respect for existing configured dbimg", strings.Join(configArgs, " "))

			// Now test with NewApp's additions, which should leave app.DBImage alone.
			app, err = ddevapp.NewApp(tmpDir, false, "")
			//nolint: errcheck
			defer app.Down(true, false)
			assert.NoError(err)
			_, err = app.ReadConfig(false)
			assert.NoError(err)
			assert.Equal(cmdMariaDBVersion, app.MariaDBVersion)
			assert.Equal("somerandomdbimg:v0.9999-"+configMariaDBVersion, app.DBImage)
		}
	}

	// If we specify both the --mariadb-version and the --dbimage flags,
	// both should be set, but the actual NewApp() will result in
	// both items, and an app.Start() would use the dbimage specified.
	for _, cmdMariaDBVersion := range versionsToTest {
		for _, cmdDBImageVersion := range versionsToTest {
			_ = os.Remove(filepath.Join(tmpDir, ".ddev", "config.yaml"))

			configArgs := append(args, cmdMariaDBVersion)
			configArgs = append(configArgs, []string{"--db-image", "drud/ddev-dbserver:v1.6.0-" + cmdDBImageVersion}...)
			configArgs = append(configArgs, "--project-name", "both-args-"+cmdMariaDBVersion)

			out, err := exec.RunCommand(DdevBin, configArgs)
			assert.NoError(err)
			assert.Contains(out, "You may now run 'ddev start'")

			// First test the bare explicit values found in the config.yaml,
			// without the NewApp adjustments
			app := &ddevapp.DdevApp{}
			assert.NoError(err)
			err = app.LoadConfigYamlFile(filepath.Join(tmpDir, ".ddev", "config.yaml"))
			assert.NoError(err)
			assert.Equal(cmdMariaDBVersion, app.MariaDBVersion)
			assert.Equal("drud/ddev-dbserver:v1.6.0-"+cmdDBImageVersion, app.DBImage)

			// Now test with NewApp()'s adjustments
			app, err = ddevapp.NewApp(tmpDir, false, "")
			//nolint: errcheck
			defer app.Down(true, false)
			assert.NoError(err)
			_, err = app.ReadConfig(false)
			assert.NoError(err)
			assert.Equal(cmdMariaDBVersion, app.MariaDBVersion)
			assert.Equal(version.DBImg+":v1.6.0-"+cmdDBImageVersion, app.DBImage)
		}
	}
}
