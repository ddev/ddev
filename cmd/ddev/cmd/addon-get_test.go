package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	copy2 "github.com/otiai10/copy"
	"github.com/stretchr/testify/require"

	asrt "github.com/stretchr/testify/assert"
)

// TestCmdAddonComplex tests advanced usages
func TestCmdAddonComplex(t *testing.T) {
	if os.Getenv("DDEV_RUN_GET_TESTS") != "true" {
		t.Skip("Skipping because DDEV_RUN_GET_TESTS is not set")
	}

	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	err = copy2.Copy(filepath.Join(origDir, "testdata", t.Name(), "project"), app.GetAppRoot())
	require.NoError(t, err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		for _, f := range []string{".platform", ".platform.app.yaml"} {
			err = os.RemoveAll(filepath.Join(app.GetAppRoot(), f))
		}
		for _, f := range []string{fmt.Sprintf("junk_%s_%s.txt", runtime.GOOS, runtime.GOARCH), "config.platformsh.yaml"} {
			err = os.RemoveAll(app.GetConfigPath(f))
			assert.NoError(err)
		}
		// We have to completely kill off app because the install.yaml + config.platformsh.yaml got us a completely different
		// database.
		err = app.Stop(true, false)
		assert.NoError(err)
		app, err = ddevapp.NewApp(app.AppRoot, true)
		assert.NoError(err)
		err = app.Start()
		assert.NoError(err)
	})

	// create no-ddev-generated.txt so we make sure we get warning about it.
	_ = os.MkdirAll(app.GetConfigPath("extra"), 0755)
	_, err = os.Create(app.GetConfigPath("extra/no-ddev-generated.txt"))
	require.NoError(t, err)

	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", filepath.Join(origDir, "testdata", t.Name(), "recipe"))
	require.NoError(t, err, "out=%s", out)

	app, err = ddevapp.GetActiveApp("")
	require.NoError(t, err)

	// Make sure that all the interpolations we wrote via go templates got in there
	assert.Equal("web99", app.Docroot)
	assert.Equal("mariadb", app.Database.Type)
	assert.Equal("10.7", app.Database.Version)
	assert.Equal("8.1", app.PHPVersion)

	// Make sure that environment variable interpolation happened. If it did, we'll have the one file
	// we're looking for.
	assert.FileExists(app.GetConfigPath(fmt.Sprintf("junk_%s_%s.txt", runtime.GOOS, runtime.GOARCH)))
	info, err := os.Stat(app.GetConfigPath("extra/no-ddev-generated.txt"))
	require.NoError(t, err, "stat of no-ddev-generated.txt failed")
	assert.True(info.Size() == 0)

	assert.Contains(out, "👍 extra/has-ddev-generated.txt")
	assert.NotContains(out, "👍 extra/no-ddev-generated.txt")
	assert.Regexp(regexp.MustCompile(`NOT overwriting [^ ]*`+"extra/no-ddev-generated.txt"), out)
}

// TestCmdAddonDependencies tests the dependency behavior is correct
func TestCmdAddonDependencies(t *testing.T) {
	if os.Getenv("DDEV_RUN_GET_TESTS") != "true" {
		t.Skip("Skipping because DDEV_RUN_GET_TESTS is not set")
	}

	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	err = copy2.Copy(filepath.Join(origDir, "testdata", t.Name(), "project"), app.GetAppRoot())
	require.NoError(t, err)

	t.Cleanup(func() {
		out, err := exec.RunHostCommand(DdevBin, "add-on", "remove", "dependency_recipe")
		assert.NoError(err, "output='%s'", out)
		out, err = exec.RunHostCommand(DdevBin, "add-on", "remove", "depender_recipe")
		assert.NoError(err, "output='%s'", out)
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	// First try of depender_recipe should fail without dependency
	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", filepath.Join(origDir, "testdata", t.Name(), "depender_recipe"))
	require.Error(t, err, "out=%s", out)

	// Now add the dependency and try again
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", filepath.Join(origDir, "testdata", t.Name(), "dependency_recipe"))
	require.NoError(t, err, "out=%s", out)

	// Now depender_recipe should succeed
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", filepath.Join(origDir, "testdata", t.Name(), "depender_recipe"))
	require.NoError(t, err, "out=%s", out)
}

// TestCmdAddonDdevVersionConstraint tests the ddev_version_constraint behavior is correct
func TestCmdAddonDdevVersionConstraint(t *testing.T) {
	if os.Getenv("DDEV_RUN_GET_TESTS") != "true" {
		t.Skip("Skipping because DDEV_RUN_GET_TESTS is not set")
	}

	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	err = copy2.Copy(filepath.Join(origDir, "testdata", t.Name(), "project"), app.GetAppRoot())
	require.NoError(t, err)

	t.Cleanup(func() {
		out, err := exec.RunHostCommand(DdevBin, "add-on", "remove", "invalid_constraint_recipe")
		assert.Error(err, "output='%s'", out)
		out, err = exec.RunHostCommand(DdevBin, "add-on", "remove", "valid_constraint_recipe")
		assert.NoError(err, "output='%s'", out)
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	// Add-on with invalid constraint should not be installed
	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", filepath.Join(origDir, "testdata", t.Name(), "invalid_constraint_recipe"))
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "constraint is not valid")

	// Add-on with valid constraint should be installed
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", filepath.Join(origDir, "testdata", t.Name(), "valid_constraint_recipe"))
	require.NoError(t, err, "out=%s", out)
}

// TestCmdAddonGetWithDotEnv tests that `ddev add-on get` can read .ddev/.env.* files,
// `ddev dotenv set` can write to .ddev/.env.* files,
// env vars are injected in PreInstallActions and PostInstallActions for add-ons,
// env vars are expanded in .ddev/docker-compose.*.yaml files.
func TestCmdAddonGetWithDotEnv(t *testing.T) {
	if os.Getenv("DDEV_RUN_GET_TESTS") != "true" {
		t.Skip("Skipping because DDEV_RUN_GET_TESTS is not set")
	}

	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	err = copy2.Copy(filepath.Join(origDir, "testdata", t.Name(), "project"), app.GetAppRoot())
	require.NoError(t, err)

	t.Cleanup(func() {
		out, err := exec.RunHostCommand(DdevBin, "add-on", "remove", "busybox")
		assert.NoError(err, "output='%s'", out)
		out, err = exec.RunHostCommand(DdevBin, "add-on", "remove", "bare-busybox")
		assert.NoError(err, "output='%s'", out)
		// Remove add-on leftovers in Docker
		out, err = exec.RunHostCommand(DdevBin, "delete", "-Oy", site.Name)
		assert.NoError(err, "output='%s'", out)
		// And register the project again in the global list for other tests
		out, err = exec.RunHostCommand(DdevBin, "config", "--auto")
		assert.NoError(err, "output='%s'", out)
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", filepath.Join(origDir, "testdata", t.Name(), "busybox"))
	require.NoError(t, err, "out=%s", out)
	busyboxEnvFile := filepath.Join(site.Dir, ".ddev/.env.busybox")
	require.NoFileExists(t, busyboxEnvFile, ".ddev/.env.busybox file should not exist at this point")

	out, err = exec.RunHostCommand(DdevBin, "dotenv", "set", ".ddev/.env.busybox", "--busybox-tag=1.36.0", "--pre-install-variable=pre", "--pre-install-variable2=pre2", "--post-install-variable", "post", "-A", "short_flag_VALUE", "--force-run", "-yes", "--force-UPPERCASE=true")
	require.NoError(t, err, "out=%s", out)
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", filepath.Join(origDir, "testdata", t.Name(), "busybox"))
	require.NoError(t, err, "out=%s", out)
	// These variables are used in pre_install_actions and post_install_actions
	require.Contains(t, out, "PRE_INSTALL_VARIABLE=pre")
	require.Contains(t, out, "PRE_INSTALL_VARIABLE2=pre2")
	require.Contains(t, out, "POST_INSTALL_VARIABLE=post")

	// .env.busybox file should exist and contain the expected environment variable
	busyboxEnvFile = filepath.Join(site.Dir, ".ddev/.env.busybox")
	require.FileExists(t, busyboxEnvFile, "unable to find .ddev/.env.busybox file, but it should be here")
	busyboxEnvFileContents, err := os.ReadFile(busyboxEnvFile)
	require.NoError(t, err, "unable to read .ddev/.env.busybox file after add-on install")
	require.Contains(t, string(busyboxEnvFileContents), `BUSYBOX_TAG="1.36.0"`)
	require.Contains(t, string(busyboxEnvFileContents), `PRE_INSTALL_VARIABLE="pre"`)
	require.Contains(t, string(busyboxEnvFileContents), `POST_INSTALL_VARIABLE="post"`)
	// --force-run is converted to empty string
	require.Contains(t, string(busyboxEnvFileContents), `FORCE_RUN=""`)
	// Some flags are ignored because they should not contain uppercase, or be a wrong flag:
	require.NotContains(t, string(busyboxEnvFileContents), `A="short_flag_VALUE"`)
	require.NotContains(t, string(busyboxEnvFileContents), "YES")
	require.NotContains(t, string(busyboxEnvFileContents), "FORCE_UPPERCASE")

	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", filepath.Join(origDir, "testdata", t.Name(), "bare-busybox"))
	require.NoError(t, err, "out=%s", out)
	bareBusyboxEnvFile := filepath.Join(site.Dir, ".ddev/.env.bare-busybox")
	require.NoFileExists(t, bareBusyboxEnvFile, ".ddev/.env.bare-busybox file should not exist at this point")

	out, err = exec.RunHostCommand(DdevBin, "dotenv", "set", ".ddev/.env.bare-busybox", "--bare-busybox-foo=bar")
	require.NoError(t, err, "out=%s", out)
	bareBusyboxEnvFile = filepath.Join(site.Dir, ".ddev/.env.bare-busybox")
	require.FileExists(t, bareBusyboxEnvFile, "unable to find .ddev/.env.bare-busybox file, but it should be here")

	out, err = exec.RunHostCommand(DdevBin, "restart")
	require.NoError(t, err, "unable to ddev restart: %v, output='%s'", err, out)

	// busybox image should be the same version as we specified
	out, err = exec.RunHostCommand(DdevBin, "exec", "-s", "busybox", "sh", "-c", "busybox | head -1")
	require.NoError(t, err, "unable to ddev exec -s busybox sh -c 'busybox | head -1': %v, output='%s'", err, out)
	require.Contains(t, out, "BusyBox v1.36.0")

	// Check that the environment variable are set correctly inside the busybox container
	out, err = exec.RunHostCommand(DdevBin, "exec", "-s", "busybox", "env")
	require.NoError(t, err, "unable to ddev exec -s busybox env: %v, output='%s'", err, out)
	// Busybox has a new tag
	require.Contains(t, out, "BUSYBOX_TAG=1.36.0")
	require.Contains(t, out, "PRE_INSTALL_VARIABLE=pre")
	require.Contains(t, out, "POST_INSTALL_VARIABLE=post")
	require.Contains(t, out, "THIS_VARIABLE_CAN_BE_CHANGED_FROM_ENV=true")
	// Variables from *.example files should not be here
	require.NotContains(t, out, "WEB_EXAMPLE_VARIABLE")
	require.Contains(t, out, "BUSYBOX_EXAMPLE_VARIABLE=notset")
	// Variable from not related .env.redis should not be here
	require.NotContains(t, out, "REDIS_TAG")
	// This variable in environment stanza is set to the default value,
	// until we pass REDIS_TAG from .env.redis to 'docker-compose config'
	require.Contains(t, out, "CAN_READ_FROM_ALL_ENV_FILES=notset")

	// Check that the environment variable are set correctly inside the bare-busybox container
	out, err = exec.RunHostCommand(DdevBin, "exec", "-s", "bare-busybox", "env")
	require.NoError(t, err, "unable to ddev exec -s bare-busybox env: %v, output='%s'", err, out)
	require.Contains(t, out, "BARE_BUSYBOX_FOO=bar")

	// Adding extra .env files here
	err = copy2.Copy(filepath.Join(origDir, "testdata", t.Name(), "env_files"), app.AppConfDir())
	require.NoError(t, err)

	// Update the busybox image in .ddev/.env.busybox
	// And update the value for THIS_VARIABLE_CAN_BE_CHANGED_FROM_ENV
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "set", ".ddev/.env.busybox", "--busybox-tag", "1.36.1", "--this-variable-can-be-changed-from-env=changed")
	require.NoError(t, err, "out=%s", out)

	out, err = exec.RunHostCommand(DdevBin, "restart")
	require.NoError(t, err, "unable to ddev restart: %v, output='%s'", err, out)

	out, err = exec.RunHostCommand(DdevBin, "exec", "-s", "busybox", "sh", "-c", "busybox | head -1")
	require.NoError(t, err, "unable to ddev exec -s busybox sh -c 'busybox | head -1': %v, output='%s'", err, out)
	require.Contains(t, out, "BusyBox v1.36.1")

	// Check that the environment variable are set correctly inside the busybox container
	out, err = exec.RunHostCommand(DdevBin, "exec", "-s", "busybox", "env")
	require.NoError(t, err, "unable to ddev exec -s busybox env: %v, output='%s'", err, out)
	// Variables from .env are passed to all containers
	require.Contains(t, out, "GLOBAL_TEST_VARIABLE=global_test_variable")
	// Busybox has a new tag
	require.Contains(t, out, "BUSYBOX_TAG=1.36.1")
	require.Contains(t, out, "PRE_INSTALL_VARIABLE=pre")
	require.Contains(t, out, "POST_INSTALL_VARIABLE=post")
	// The variable below is already added to the busybox environment stanza.
	require.Contains(t, out, "THIS_VARIABLE_CAN_BE_CHANGED_FROM_ENV=changed")
	require.NotContains(t, out, "THIS_VARIABLE_CAN_BE_CHANGED_FROM_ENV=true")
	// Variables from *.example files should not be here
	require.NotContains(t, out, "WEB_EXAMPLE_VARIABLE")
	require.Contains(t, out, "BUSYBOX_EXAMPLE_VARIABLE=notset")
	// Variable from not related .env.redis should not be here
	require.NotContains(t, out, "REDIS_TAG")
	// But this REDIS_TAG variable from .env.redis can be expanded during 'docker-compose config'
	require.Contains(t, out, "CAN_READ_FROM_ALL_ENV_FILES=7")

	// Check that the environment variable are set correctly inside the bare-busybox container
	out, err = exec.RunHostCommand(DdevBin, "exec", "-s", "bare-busybox", "env")
	require.NoError(t, err, "unable to ddev exec -s bare-busybox env: %v, output='%s'", err, out)
	require.Contains(t, out, "BARE_BUSYBOX_FOO=bar")
	// Variables from .env are passed to all containers
	require.Contains(t, out, "GLOBAL_TEST_VARIABLE=global_test_variable")

	// Check that the environment variable are set correctly inside the web container
	out, err = exec.RunHostCommand(DdevBin, "exec", "env")
	require.NoError(t, err, "unable to ddev exec env: %v, output='%s'", err, out)
	// We set a higher priority for .env.web than for .env
	require.NotContains(t, out, "GLOBAL_TEST_VARIABLE=global_test_variable")
	require.Contains(t, out, "GLOBAL_TEST_VARIABLE=web_test_variable")
	require.Contains(t, out, "WEB_ADDITIONAL_VARIABLE=web_additional_variable")
	// And web container should not have variables from the busybox container
	require.NotContains(t, out, "BUSYBOX_TAG")
	require.NotContains(t, out, "PRE_INSTALL_VARIABLE")
	require.NotContains(t, out, "POST_INSTALL_VARIABLE")
	require.NotContains(t, out, "THIS_VARIABLE_CAN_BE_CHANGED_FROM_ENV")
	require.NotContains(t, out, "BUSYBOX_EXAMPLE_VARIABLE")
	// Variable from *.example files should not be here
	require.NotContains(t, out, "WEB_EXAMPLE_VARIABLE")
	// Variable from not related .env.redis should not be here
	require.NotContains(t, out, "REDIS_TAG")
	// Variable from another service should not be here
	require.NotContains(t, out, "CAN_READ_FROM_ALL_ENV_FILES")

	// Check that the environment variable are set correctly inside the db container
	out, err = exec.RunHostCommand(DdevBin, "exec", "-s", "db", "env")
	require.NoError(t, err, "unable to ddev exec -s db env: %v, output='%s'", err, out)
	// Variables from .env are passed to all containers
	require.Contains(t, out, "GLOBAL_TEST_VARIABLE=global_test_variable")
}
