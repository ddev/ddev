package cmd

import (
	"strconv"
	"testing"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"

	"os"
	"path/filepath"

	"strings"

	"fmt"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// TestCmdConfigHooks tests that pre-config and post-config hooks run
func TestCmdConfigHooks(t *testing.T) {
	// Change to the first DevTestSite for the duration of this test.
	site := TestSites[0]
	defer site.Chdir()()
	assert := asrt.New(t)

	app, err := ddevapp.NewApp(site.Dir, true)
	assert.NoError(err)
	app.Hooks = map[string][]ddevapp.YAMLTask{"post-config": {{"exec-host": "touch hello-post-config-" + app.Name}}, "pre-config": {{"exec-host": "touch hello-pre-config-" + app.Name}}}
	err = app.WriteConfig()
	assert.NoError(err)
	// Make sure we get rid of this for other uses
	defer func() {
		app.Hooks = nil
		_ = app.WriteConfig()
	}()

	_, err = exec.RunCommand(DdevBin, []string{"config", "--project-type=" + app.Type})
	assert.NoError(err)

	assert.FileExists("hello-pre-config-" + app.Name)
	assert.FileExists("hello-post-config-" + app.Name)
	err = os.Remove("hello-pre-config-" + app.Name)
	assert.NoError(err)
	err = os.Remove("hello-post-config-" + app.Name)
	assert.NoError(err)
}

// TestConfigDescribeLocation tries out the --show-config-location flag.
func TestConfigDescribeLocation(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	// Create a temporary directory and switch to it.
	tmpDir := testcommon.CreateTmpDir(t.Name())
	err := os.Chdir(tmpDir)
	assert.NoError(err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		out, err := exec.RunHostCommand(DdevBin, "delete", "-Oy", t.Name())
		assert.NoError(err, "output=%s", out)
		err = os.RemoveAll(tmpDir)
		assert.NoError(err)

	})
	out, err := exec.RunHostCommand(DdevBin, "config", "--docroot=.", "--project-name="+t.Name())
	assert.NoError(err)
	assert.Contains(string(out), "Configuring unrecognized codebase as project of type 'php'")

	// Now see if we can detect it
	out, err = exec.RunHostCommand(DdevBin, "config", "--show-config-location")
	assert.NoError(err)
	assert.Contains(string(out), tmpDir)

	// Now try it in a directory that doesn't have a config
	tmpDir = testcommon.CreateTmpDir(t.Name())
	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(tmpDir)
		assert.NoError(err)
	})
	err = os.Chdir(tmpDir)
	assert.NoError(err)

	out, err = exec.RunHostCommand(DdevBin, "config", "--show-config-location")
	assert.Error(err)
	assert.Contains(string(out), "No project configuration currently exists")
}

// TestConfigWithSitenameFlagDetectsDocroot tests that the docroot is detected when
// flags like --project-name are passed.
func TestConfigWithSitenameFlagDetectsDocroot(t *testing.T) {
	assert := asrt.New(t)

	// Create a temporary directory and switch to it.
	testDocrootName := "web"
	tmpdir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()
	// Create a document root folder.

	expectedFile := filepath.Join(tmpdir, testDocrootName, "misc/ahah.js")
	err := os.MkdirAll(filepath.Dir(expectedFile), 0777)
	assert.NoError(err)

	// create index.php that defines docroot
	_, err = os.OpenFile(filepath.Join(tmpdir, testDocrootName, "index.php"), os.O_RDONLY|os.O_CREATE, 0666)
	assert.NoError(err)

	// create the misc/ahah.js that signals drupal6
	_, err = os.OpenFile(expectedFile, os.O_RDONLY|os.O_CREATE, 0666)
	assert.NoError(err)

	// Create a config
	args := []string{"config", "--project-name=config-with-sitename", "--php-version=7.2"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	defer func() {
		_, _ = exec.RunCommand(DdevBin, []string{"delete", "-Oy", "config-with-sitename"})
	}()
	assert.Contains(string(out), "Configuring a 'drupal6' codebase with docroot", nodeps.AppTypeDrupal6)
}

// TestConfigSetValues sets all available configuration values using command flags, then confirms that the
// values have been correctly written to the config file.
func TestConfigSetValues(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	// Create a temporary directory and switch to it.
	tmpDir := testcommon.CreateTmpDir(t.Name())
	err := os.Chdir(tmpDir)
	assert.NoError(err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		out, err := exec.RunHostCommand(DdevBin, "delete", "-Oy", t.Name())
		assert.NoError(err, "output=%s", out)
		err = os.RemoveAll(tmpDir)
		assert.NoError(err)
	})

	// Create an existing docroot
	docroot := "web"
	if err = os.MkdirAll(filepath.Join(tmpDir, docroot), 0755); err != nil {
		t.Errorf("Could not create docroot %s in %s", docroot, tmpDir)
	}

	err = os.Chdir(tmpDir)
	assert.NoError(err)

	// Build config args
	projectName := t.Name()
	projectType := nodeps.AppTypeTYPO3
	phpVersion := nodeps.PHP71
	httpPort := "81"
	httpsPort := "444"
	hostDBPort := "60001"
	hostWebserverPort := "60002"
	hostHTTPSPort := "60003"
	xdebugEnabled := true
	noProjectMount := true
	composerRoot := "composer-root"
	composerVersion := "2.0.0-RC2"
	additionalHostnamesSlice := []string{"abc", "123", "xyz"}
	additionalHostnames := strings.Join(additionalHostnamesSlice, ",")
	additionalFQDNsSlice := []string{"abc.com", "123.pizza", "xyz.co.uk"}
	additionalFQDNs := strings.Join(additionalFQDNsSlice, ",")
	omitContainersSlice := []string{"ddev-ssh-agent"}
	omitContainers := strings.Join(omitContainersSlice, ",")
	webimageExtraPackagesSlice := []string{"php-bcmath", "php7.3-tidy"}
	webimageExtraPackages := strings.Join(webimageExtraPackagesSlice, ",")
	dbimageExtraPackagesSlice := []string{"netcat", "ncdu"}
	dbimageExtraPackages := strings.Join(dbimageExtraPackagesSlice, ",")

	uploadDirsSlice := []string{"custom", "config", "path"}
	webserverType := nodeps.WebserverApacheFPM
	webImage := "custom-web-image"
	webWorkingDir := "/custom/web/dir"
	dbWorkingDir := "/custom/db/dir"
	mailhogPort := "5001"
	projectTLD := "nowhere.example.com"
	useDNSWhenPossible := false
	timezone := "America/Chicago"
	webEnv := "SOMEENV=some+val"
	nodejsVersion := "14"
	defaultContainerTimeout := 300

	args := []string{
		"config",
		"--project-name", projectName,
		"--docroot", docroot,
		"--project-type", projectType,
		"--php-version", phpVersion,
		"--composer-root", composerRoot,
		"--composer-version", composerVersion,
		"--http-port", httpPort,
		"--https-port", httpsPort,
		fmt.Sprintf("--xdebug-enabled=%t", xdebugEnabled),
		fmt.Sprintf("--no-project-mount=%t", noProjectMount),
		"--additional-hostnames", additionalHostnames,
		"--additional-fqdns", additionalFQDNs,
		"--upload-dirs=" + strings.Join(uploadDirsSlice, ","),
		"--webserver-type", webserverType,
		"--web-image", webImage,
		"--web-working-dir", webWorkingDir,
		"--db-working-dir", dbWorkingDir,
		"--omit-containers", omitContainers,
		"--host-db-port", hostDBPort,
		"--host-webserver-port", hostWebserverPort,
		"--host-https-port", hostHTTPSPort,
		"--webimage-extra-packages", webimageExtraPackages,
		"--dbimage-extra-packages", dbimageExtraPackages,
		"--mailhog-port", mailhogPort,
		"--project-tld", projectTLD,
		"--web-environment", webEnv,
		"--nodejs-version", nodejsVersion,
		"--default-container-timeout", strconv.FormatInt(int64(defaultContainerTimeout), 10),
		fmt.Sprintf("--use-dns-when-possible=%t", useDNSWhenPossible),
		"--timezone", timezone,
		"--disable-upload-dirs-warning",
	}

	out, err := exec.RunHostCommand(DdevBin, args...)
	assert.NoError(err, "error running ddev %v: %v, output=%s", args, err, out)

	configFile := filepath.Join(tmpDir, ".ddev", "config.yaml")
	configContents, err := os.ReadFile(configFile)
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
	assert.Equal(composerRoot, app.ComposerRoot)
	assert.Equal(composerVersion, app.ComposerVersion)
	assert.Equal(httpPort, app.RouterHTTPPort)
	assert.Equal(httpsPort, app.RouterHTTPSPort)
	assert.Equal(hostWebserverPort, app.HostWebserverPort)
	assert.Equal(hostDBPort, app.HostDBPort)
	assert.Equal(xdebugEnabled, app.XdebugEnabled)
	assert.Equal(noProjectMount, app.NoProjectMount)
	assert.Equal(additionalHostnamesSlice, app.AdditionalHostnames)
	assert.Equal(additionalFQDNsSlice, app.AdditionalFQDNs)
	assert.Equal(uploadDirsSlice, app.GetUploadDirs())
	assert.Equal(webserverType, app.WebserverType)
	assert.Equal(webImage, app.WebImage)
	assert.Equal(webWorkingDir, app.WorkingDir["web"])
	assert.Equal(dbWorkingDir, app.WorkingDir["db"])
	assert.Equal(webimageExtraPackagesSlice, app.WebImageExtraPackages)
	assert.Equal(dbimageExtraPackagesSlice, app.DBImageExtraPackages)
	assert.Equal(mailhogPort, app.MailhogPort)
	assert.Equal(useDNSWhenPossible, app.UseDNSWhenPossible)
	assert.Equal(projectTLD, app.ProjectTLD)
	assert.Equal(timezone, app.Timezone)
	require.NotEmpty(t, app.WebEnvironment)
	assert.Equal(webEnv, app.WebEnvironment[0])
	assert.Equal(nodejsVersion, app.NodeJSVersion)
	assert.Equal(strconv.Itoa(defaultContainerTimeout), app.DefaultContainerTimeout)
	assert.Equal(true, app.DisableUploadDirsWarning)

	// Test that container images, working dirs and composer root dir can be unset with default flags
	args = []string{
		"config",
		"--composer-root-default",
		"--web-image-default",
		"--db-image-default",
		"--web-working-dir-default",
		"--db-working-dir-default",
	}

	_, err = exec.RunHostCommand(DdevBin, args...)
	assert.NoError(err)

	configContents, err = os.ReadFile(configFile)
	assert.NoError(err, "Unable to read %s: %v", configFile, err)

	app = &ddevapp.DdevApp{}
	err = yaml.Unmarshal(configContents, app)
	assert.NoError(err, "Could not unmarshal %s: %v", configFile, err)

	assert.Equal(app.ComposerRoot, "")
	assert.Equal(app.WebImage, "")
	assert.Equal(len(app.WorkingDir), 0)

	// Test that all container images and working dirs can each be unset with single default images flag
	args = []string{
		"config",
		"--web-image", webImage,
		"--web-working-dir", webWorkingDir,
		"--db-working-dir", dbWorkingDir,
	}

	_, err = exec.RunHostCommand(DdevBin, args...)
	assert.NoError(err)

	args = []string{
		"config",
		"--image-defaults",
		"--working-dir-defaults",
	}

	_, err = exec.RunHostCommand(DdevBin, args...)
	assert.NoError(err)

	configContents, err = os.ReadFile(configFile)
	assert.NoError(err, "Unable to read %s: %v", configFile, err)

	app = &ddevapp.DdevApp{}
	err = yaml.Unmarshal(configContents, app)
	assert.NoError(err, "Could not unmarshal %s: %v", configFile, err)

	assert.Equal(app.WebImage, "")
	assert.Equal(len(app.WorkingDir), 0)

	// Test that variables can be appended to the web environment
	args = []string{
		"config",
		"--web-environment-add", webEnv,
	}

	_, err = exec.RunHostCommand(DdevBin, args...)
	assert.NoError(err)

	configContents, err = os.ReadFile(configFile)
	assert.NoError(err, "Unable to read %s: %v", configFile, err)

	app = &ddevapp.DdevApp{}
	err = yaml.Unmarshal(configContents, app)
	assert.NoError(err, "Could not unmarshal %s: %v", configFile, err)

	assert.Equal(1, len(app.WebEnvironment))
	assert.Equal([]string{webEnv}, app.WebEnvironment)

	args = []string{
		"config",
		"--web-environment-add", "SPACES=with spaces,FOO=bar,BAR=baz",
	}

	_, err = exec.RunHostCommand(DdevBin, args...)
	assert.NoError(err)

	configContents, err = os.ReadFile(configFile)
	assert.NoError(err, "Unable to read %s: %v", configFile, err)

	app = &ddevapp.DdevApp{}
	err = yaml.Unmarshal(configContents, app)
	assert.NoError(err, "Could not unmarshal %s: %v", configFile, err)

	assert.Equal(4, len(app.WebEnvironment))
	assert.Equal("BAR=baz", app.WebEnvironment[0])
	assert.Equal("FOO=bar", app.WebEnvironment[1])
	assert.Equal("SPACES=with spaces", app.WebEnvironment[3])
	assert.Equal(webEnv, app.WebEnvironment[2])
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
		args = []string{
			"stop",
			"--unlist", projName,
		}
		_, _ = exec.RunCommand(DdevBin, args)

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

// TestCmdDisasterConfig tests to make sure we can't accidentally
// config in homedir, and that config in a subdir is handled correctly
func TestCmdDisasterConfig(t *testing.T) {
	var err error
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	// Make sure we're not allowed to config in home directory.
	home, _ := os.UserHomeDir()
	err = os.Chdir(home)
	assert.NoError(err)
	out, err := exec.RunHostCommand(DdevBin, "config", "--project-type=php")
	assert.Error(err)
	assert.Contains(out, "not useful in")

	// Create a temporary directory and switch to it.
	tmpDir := testcommon.CreateTmpDir(t.Name())
	err = os.Chdir(tmpDir)
	assert.NoError(err)
	// Create a project
	_, err = exec.RunHostCommand(DdevBin, "config", "--project-type=php", "--project-name="+t.Name())
	assert.NoError(err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		_, err = exec.RunHostCommand(DdevBin, "delete", "-Oy", t.Name())
		assert.NoError(err)
		_, err = exec.RunHostCommand(DdevBin, "delete", "-Oy", t.Name()+"_subdir")
		assert.Error(err)
		_ = os.RemoveAll(tmpDir)
	})

	subdirName := t.Name() + fileutil.RandomFilenameBase()
	subdir := filepath.Join(tmpDir, subdirName)
	err = os.Mkdir(subdir, 0777)
	assert.NoError(err)
	err = os.Chdir(subdir)
	assert.NoError(err)

	// Make sure that ddev config in a subdir gives an error
	out, err = exec.RunHostCommand(DdevBin, "config", "--project-type=php", "--project-name="+t.Name()+"_subdir")
	assert.Error(err)
	assert.Contains(out, "possible you wanted to")
	assert.Contains(out, fmt.Sprintf("parent directory %s?", tmpDir))
	assert.NoFileExists(filepath.Join(subdir, ".ddev/config.yaml"))
}

// TestConfigDatabaseVersion checks to make sure that both
// ddev config --database behaves correctly,
func TestConfigDatabaseVersion(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	versionsToTest := nodeps.GetValidDatabaseVersions()
	if os.Getenv("GOTEST_SHORT") != "" {
		versionsToTest = []string{"mariadb:10.3", "mysql:5.7"}
	}

	// Create a temporary directory and switch to it.
	tmpDir := testcommon.CreateTmpDir(t.Name())
	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	err = globalconfig.RemoveProjectInfo(t.Name())
	assert.NoError(err)

	out, err := exec.RunHostCommand(DdevBin, "config", "--project-name", t.Name())
	assert.NoError(err, "Failed running ddev config --auto: %s", out)

	err = globalconfig.ReadGlobalConfig()
	require.NoError(t, err)

	app, err := ddevapp.GetActiveApp("")
	assert.NoError(err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(tmpDir)
		assert.NoError(err)
	})

	_, err = app.ReadConfig(false)
	assert.NoError(err)
	assert.Equal(nodeps.MariaDB, app.Database.Type)
	assert.Equal(nodeps.MariaDBDefaultVersion, app.Database.Version)

	err = app.Start()
	assert.NoError(err)
	err = app.Stop(true, false)
	assert.NoError(err)

	// Verify behavior with no existing config.yaml. It should
	// just add database into the config and nothing else
	for _, dbTypeVersion := range versionsToTest {
		parts := strings.Split(dbTypeVersion, ":")
		err = os.RemoveAll(filepath.Join(tmpDir, ".ddev"))
		assert.NoError(err)
		out, err := exec.RunHostCommand(DdevBin, "config", "--database", dbTypeVersion)
		assert.NoError(err, "Failed to run ddev config --database %s: %s", dbTypeVersion, out)
		assert.Contains(out, "You may now run 'ddev start'")

		// First test the bare explicit values found in the config.yaml,
		// without the NewApp adjustments
		app := &ddevapp.DdevApp{}
		assert.NoError(err)
		err = app.LoadConfigYamlFile(filepath.Join(tmpDir, ".ddev", "config.yaml"))
		assert.NoError(err)
		assert.Equal(parts[0], app.Database.Type)
		assert.Equal(parts[1], app.Database.Version)

		// Now use NewApp() to load, so that we get the full logic of that function.
		app, err = ddevapp.NewApp(tmpDir, false)
		assert.NoError(err)
		t.Cleanup(func() {
			err = app.Stop(true, false)
			assert.NoError(err)
		})
		_, err = app.ReadConfig(false)
		assert.NoError(err)
		assert.Equal(parts[0], app.Database.Type)
		assert.Equal(parts[1], app.Database.Version)
		err = app.Stop(true, false)
		assert.NoError(err)
	}
}

// TestConfigGitignore checks that our gitignore is ignoring the right things.
func TestConfigGitignore(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	// Create a temporary directory and switch to it.
	testDir := testcommon.CreateTmpDir(t.Name())

	err := os.Chdir(testDir)
	require.NoError(t, err)

	_, err = exec.RunHostCommand(DdevBin, "config", "--auto")
	assert.NoError(err)
	t.Cleanup(func() {
		_, err = exec.RunHostCommand(DdevBin, "delete", "-Oy")
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		_, err = exec.RunHostCommand("bash", "-c", fmt.Sprintf("rm -f ~/.ddev/commands/web/%s ~/.ddev/homeadditions/%s", t.Name(), t.Name()))
		assert.NoError(err)
		err = os.RemoveAll(testDir)
		assert.NoError(err)
	})

	_, err = exec.RunHostCommand("git", "init")
	assert.NoError(err)
	_, err = exec.RunHostCommand("git", "add", ".")
	assert.NoError(err)
	out, err := exec.RunHostCommand("git", "status")
	assert.NoError(err)

	// git status should have one new file, config.yaml
	assert.Contains(out, "new file:   .ddev/config.yaml")
	// .ddev/config.yaml should be the only new file, remove it and check
	out = strings.ReplaceAll(out, "new file:   .ddev/config.yaml", "")
	assert.NotContains(out, "new file:")

	_, err = exec.RunHostCommand("bash", "-c", fmt.Sprintf("touch ~/.ddev/commands/web/%s ~/.ddev/homeadditions/%s", t.Name(), t.Name()))
	assert.NoError(err)
	if err != nil {
		out, err = exec.RunHostCommand("bash", "-c", "ls -l ~/.ddev && ls -lR ~/.ddev/commands ~/.ddev/homeadditions")
		assert.NoError(err)
		t.Logf("Contents of global .ddev: \n=====\n%s\n====", out)
	}

	_, err = exec.RunHostCommand(DdevBin, "start", "-y")
	assert.NoError(err)
	statusOut, err := exec.RunHostCommand("bash", "-c", "git status")
	assert.NoError(err)
	_, err = exec.RunHostCommand("bash", "-c", "git status | grep 'Untracked files'")
	assert.Error(err, "Untracked files were found where we didn't expect them: %s", statusOut)
}
