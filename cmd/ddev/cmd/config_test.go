package cmd

import (
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
	}

}
