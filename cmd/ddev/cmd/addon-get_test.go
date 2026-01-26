package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/globalconfig"
	copy2 "github.com/otiai10/copy"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCmdAddonComplex tests advanced usages
func TestCmdAddonComplex(t *testing.T) {
	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	err = copy2.Copy(filepath.Join(origDir, "testdata", t.Name(), "project"), app.GetAppRoot())
	require.NoError(t, err)

	t.Cleanup(func() {
		assert := asrt.New(t)
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
	require.Equal(t, "web99", app.Docroot)
	require.Equal(t, "mariadb", app.Database.Type)
	require.Equal(t, "10.7", app.Database.Version)
	require.Equal(t, "8.1", app.PHPVersion)

	// Make sure that environment variable interpolation happened. If it did, we'll have the one file
	// we're looking for.
	require.FileExists(t, app.GetConfigPath(fmt.Sprintf("junk_%s_%s.txt", runtime.GOOS, runtime.GOARCH)))
	info, err := os.Stat(app.GetConfigPath("extra/no-ddev-generated.txt"))
	require.NoError(t, err, "stat of no-ddev-generated.txt failed")
	// With empty file handling, the empty file is overwritten with addon content (39 bytes)
	require.True(t, info.Size() == 39, "empty file should be overwritten with addon content")

	require.Contains(t, out, fmt.Sprintf("👍 %s", filepath.Join("extra", "has-ddev-generated.txt")))
	// With empty file handling, no-ddev-generated.txt is also installed (empty files are overwritten)
	require.Contains(t, out, fmt.Sprintf("👍 %s", filepath.Join("extra", "no-ddev-generated.txt")))
}

// TestCmdAddonComplex tests advanced usages
func TestCmdAddonActionsOutput(t *testing.T) {
	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	t.Cleanup(func() {
		assert := asrt.New(t)
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", filepath.Join(origDir, "testdata", t.Name(), "recipe"))
	require.NoError(t, err, "out=%s", out)

	// The first action outputs nothing but creates a file.
	require.FileExists(t, app.GetConfigPath("test_cmd_addon_actions_no_output.txt"))

	// Check that we see the "Executing post-install actions:" header
	require.Contains(t, out, "Executing post-install actions:")

	// The second action outputs something and should be present
	require.Contains(t, out, "action 2 with output and no #ddev-description")

	// The third action should be success with output and created a file.
	require.Contains(t, out, "👍 Action 3 with #ddev-description and output")
	require.Contains(t, out, "test_cmd_addon_actions_output.txt created")
	require.FileExists(t, app.GetConfigPath("test_cmd_addon_actions_output.txt"))

	// The fourth action should also be success with output
	require.Contains(t, out, "👍 Action 4 that errs if .ddev/test_cmd_addon_actions_output_error.txt is present")
	require.Contains(t, out, "test_cmd_addon_actions_output_error.txt not found!")

	// The fifth action has an exit statement that should normally be an error, but because of '#ddev-warning-exit-code'
	// the rest of the actions continue to run normally.
	// It also creates a file.
	require.FileExists(t, app.GetConfigPath("test_cmd_addon_actions_no_output_warning.txt"))

	// The sixth action is also a warning but with output and no description
	require.Contains(t, out, "action 6 with output, #ddev-warning-exit-code and no #ddev-description")

	// The seventh action creates a file but has a '#ddev-description'.
	require.Contains(t, out, "👍 Action 7 with #ddev-description and no output")
	require.FileExists(t, app.GetConfigPath("test_cmd_addon_actions_description.txt"))

	// The eighth action is both a warning and has a description but has no output.
	require.Contains(t, out, "⚠️ Action 8 with #ddev-warning-exit-code and #ddev-description and no output")

	// The ninth action is both a warning, has a description and has output.
	require.Contains(t, out, "⚠️ Action 9 with #ddev-warning-exit-code and #ddev-description and some output")
	require.Contains(t, out, "This is a warning!!!")
	// This action also has an echo after the exit code. We know that will never be output, but we
	// can check for it anyway.
	require.NotContains(t, out, "This line that comes after an exit should never be output")

	// The last action has an output to wrap things up
	require.Contains(t, out, "👍 Action 10 is our final action doing nothing")

	// We now want to make sure it fails when it has to and with the proper output
	_, err = os.Create(app.GetConfigPath("test_cmd_addon_actions_output_error.txt"))
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", filepath.Join(origDir, "testdata", t.Name(), "recipe"))
	require.Error(t, err, "out=%s", out)

	// The fourth action should have erred with output.
	require.Contains(t, out, "👎 Action 4 that errs if .ddev/test_cmd_addon_actions_output_error.txt is present")
	require.Contains(t, out, "test_cmd_addon_actions_output_error.txt found!")

	// We should never have reached further actions.
	require.NotContains(t, out, "👍 Action 7 with #ddev-description and no output")
}

// TestCmdAddonDependencies tests the dependency behavior is correct
func TestCmdAddonDependencies(t *testing.T) {
	origDir, _ := os.Getwd()
	t.Setenv("DDEV_ADDON_TEST_DIR", filepath.Join(origDir, "testdata", "test-addons"))
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	err = copy2.Copy(filepath.Join(origDir, "testdata", t.Name(), "project"), app.GetAppRoot())
	require.NoError(t, err)

	t.Cleanup(func() {
		assert := asrt.New(t)
		out, err := exec.RunHostCommand(DdevBin, "add-on", "remove", "depender_recipe")
		assert.NoError(err, "output='%s'", out)
		out, err = exec.RunHostCommand(DdevBin, "add-on", "remove", "dependency_recipe")
		assert.NoError(err, "output='%s'", out)
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	// Install depender_recipe - should succeed and auto-install test/dependency_recipe
	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", filepath.Join(origDir, "testdata", t.Name(), "depender_recipe"))
	require.NoError(t, err, "out=%s", out)
	require.Contains(t, out, "Installing missing dependency: test/dependency_recipe", "Should auto-install test dependency")
	require.Contains(t, out, "Successfully installed dependency_recipe from directory", "Should successfully install from test fixture")
}

// TestCmdAddonDdevVersionConstraint tests the ddev_version_constraint behavior is correct
func TestCmdAddonDdevVersionConstraint(t *testing.T) {
	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	err = copy2.Copy(filepath.Join(origDir, "testdata", t.Name(), "project"), app.GetAppRoot())
	require.NoError(t, err)

	t.Cleanup(func() {
		assert := asrt.New(t)
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
	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	err = copy2.Copy(filepath.Join(origDir, "testdata", t.Name(), "project"), app.GetAppRoot())
	require.NoError(t, err)

	t.Cleanup(func() {
		assert := asrt.New(t)
		out, err := exec.RunHostCommand(DdevBin, "stop", site.Name)
		assert.NoError(err, "output='%s'", out)
		out, err = exec.RunHostCommand(DdevBin, "add-on", "remove", "busybox")
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

	out, err = exec.RunHostCommand(DdevBin, "dotenv", "set", ".ddev/.env.busybox", "--busybox-tag=1.36.0", "--pre-install-variable=pre", "--pre-install-variable2=pre2", "--post-install-variable", "post", "--force-run", "--dollar-sign", `$dollar_variable`)
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
	require.Contains(t, string(busyboxEnvFileContents), `DOLLAR_SIGN="\$dollar_variable"`)
	// --force-run is converted to empty string
	require.Contains(t, string(busyboxEnvFileContents), `FORCE_RUN=""`)

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
	require.Contains(t, out, `DOLLAR_SIGN=$dollar_variable`)
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
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "set", ".ddev/.env.busybox", "--busybox-tag", "1.36.1", "--this-variable-can-be-changed-from-env=changed", "--dollar-sign", `$dollar_variable_override`)
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
	require.Contains(t, out, `DOLLAR_SIGN=$dollar_variable_override`)
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

// TestAddonGetWithDependencies tests static dependency installation
func TestAddonGetWithDependencies(t *testing.T) {
	origDir, _ := os.Getwd()
	t.Setenv("DDEV_ADDON_TEST_DIR", filepath.Join(origDir, "testdata", "test-addons"))
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	t.Cleanup(func() {
		assert := asrt.New(t)
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "mock-redis-commander")
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "mock-redis")
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	// Test that installing an addon with dependencies automatically installs the dependency
	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", filepath.Join(origDir, "testdata", t.Name(), "mock_redis_commander"))
	require.NoError(t, err, "Should successfully install addon with dependencies, out=%s", out)

	// Verify both addons are installed
	require.Contains(t, out, "Successfully installed mock-redis from directory")
	require.Contains(t, out, "Redis service configured successfully") // This should be in the pre-install output
	require.Contains(t, out, "Redis Commander service configured successfully")

	// Verify files were created
	require.FileExists(t, app.GetConfigPath("docker-compose.redis.yaml"))
	require.FileExists(t, app.GetConfigPath("docker-compose.redis-commander.yaml"))
}

// TestAddonGetCircularDependencies tests circular dependency detection
func TestAddonGetCircularDependencies(t *testing.T) {
	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert := asrt.New(t)
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "test-dummy-self-referencing-addon")
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	// Test that circular dependencies are detected and rejected using real self-referencing add-on
	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/test-dummy-self-referencing-addon")
	require.Error(t, err, "Should fail due to circular dependency, out=%s", out)
	require.Contains(t, out, "circular dependency detected", "Should mention circular dependency")
}

// TestAddonGetSkipDepsFlag tests the --skip-deps flag
func TestAddonGetSkipDepsFlag(t *testing.T) {
	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert := asrt.New(t)
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "mock-redis")
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "mock-redis-commander")
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	// Test that --skip-deps flag prevents automatic installation but does not fail
	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", "--skip-deps", filepath.Join(origDir, "testdata", t.Name(), "mock_redis_commander"))
	require.NoError(t, err, "Should not fail when --skip-deps is used, out=%s", out)
	require.NotContains(t, out, "Installing missing dependency", "Should not automatically install dependencies")

	// Install dependency first
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", filepath.Join(origDir, "testdata", t.Name(), "mock_redis"))
	require.NoError(t, err, "Should install dependency successfully, out=%s", out)

	// Now --skip-deps should also work (dependency present, but still skipped)
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", "--skip-deps", filepath.Join(origDir, "testdata", t.Name(), "mock_redis_commander"))
	require.NoError(t, err, "Should install with --skip-deps when dependency exists, out=%s", out)
	require.NotContains(t, out, "Installing missing dependency", "Should not automatically install dependencies")
}

// TestAddonGetRuntimeDependencies tests runtime dependency file parsing
func TestAddonGetRuntimeDependencies(t *testing.T) {
	origDir, _ := os.Getwd()
	t.Setenv("DDEV_ADDON_TEST_DIR", filepath.Join(origDir, "testdata", "test-addons"))
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	// Ensure clean state before testing
	_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "runtime-deps-addon")
	_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "mock-redis")
	_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "mock-redis-commander")

	t.Cleanup(func() {
		assert := asrt.New(t)
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "runtime-deps-addon")
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "mock-redis")
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "mock-redis-commander")
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	// Test that runtime dependencies are discovered and installed
	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", "--verbose", filepath.Join(origDir, "testdata", t.Name(), "runtime_deps_addon"))
	require.NoError(t, err, "Should successfully install addon with runtime dependencies, out=%s", out)

	// Verify runtime dependency installation
	require.Contains(t, out, "Installing runtime dependencies")
	require.Contains(t, out, "Successfully installed mock-redis from directory")
	require.Contains(t, out, "Successfully installed mock-redis-commander from directory")
	require.Contains(t, out, "Runtime dependencies addon configured successfully")

	// Verify files were created by all addons
	require.FileExists(t, app.GetConfigPath("docker-compose.redis.yaml"))
	require.FileExists(t, app.GetConfigPath("docker-compose.redis-commander.yaml"))
	require.FileExists(t, app.GetConfigPath("config.runtime-deps-addon.yaml"))

	// Verify the runtime deps file was cleaned up
	runtimeDepsFile := app.GetConfigPath(".runtime-deps-runtime-deps-addon")
	require.NoFileExists(t, runtimeDepsFile, "Runtime dependencies file should be cleaned up after processing")
}

// TestAddonGetPostInstallRuntimeDependencies tests runtime dependency file parsing when created during post-install actions
func TestAddonGetPostInstallRuntimeDependencies(t *testing.T) {
	origDir, _ := os.Getwd()
	t.Setenv("DDEV_ADDON_TEST_DIR", filepath.Join(origDir, "testdata", "test-addons"))
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	// Ensure clean state before testing
	_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "post-install-runtime-deps-addon")
	_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "mock-redis")
	_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "mock-redis-commander")

	t.Cleanup(func() {
		assert := asrt.New(t)
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "post-install-runtime-deps-addon")
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "mock-redis")
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "mock-redis-commander")
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	// Test that runtime dependencies created during post-install actions are discovered and installed
	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", "--verbose", filepath.Join(origDir, "testdata", t.Name(), "post_install_runtime_deps_addon"))
	require.NoError(t, err, "Should successfully install addon with post-install runtime dependencies, out=%s", out)

	// Verify runtime dependency installation
	require.Contains(t, out, "Installing runtime dependencies")
	require.Contains(t, out, "Successfully installed mock-redis from directory")
	require.Contains(t, out, "Successfully installed mock-redis-commander from directory")
	require.Contains(t, out, "Post-install runtime dependencies file created")

	// Verify files were created by all addons
	require.FileExists(t, app.GetConfigPath("docker-compose.redis.yaml"))
	require.FileExists(t, app.GetConfigPath("docker-compose.redis-commander.yaml"))
	require.FileExists(t, app.GetConfigPath("config.post-install-runtime-deps-addon.yaml"))

	// Verify the runtime deps file was cleaned up
	runtimeDepsFile := app.GetConfigPath(".runtime-deps-post-install-runtime-deps-addon")
	require.NoFileExists(t, runtimeDepsFile, "Runtime dependencies file should be cleaned up after processing")
}

// TestAddonGetWithPRFlag tests the --project flag functionality
func TestCmdAddonProjectFlag(t *testing.T) {
	if !github.HasGitHubToken() {
		t.Skip("Skipping because DDEV_GITHUB_TOKEN is not set")
	}
	origDdevDebug := os.Getenv("DDEV_DEBUG")
	_ = os.Unsetenv("DDEV_DEBUG")
	assert := asrt.New(t)

	site := TestSites[0]
	// Explicitly don't chdir to the project

	t.Cleanup(func() {
		_, err := exec.RunHostCommand(DdevBin, "add-on", "remove", "redis", "--project", site.Name)
		assert.NoError(err)
		_ = os.RemoveAll(filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/web/global-touched"))
		_ = os.Setenv("DDEV_DEBUG", origDdevDebug)
	})

	// Install the add-on using the `--project` flag
	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis", "--project", site.Name, "--json-output")
	require.NoError(t, err, "failed ddev add-on get ddev/ddev-redis --project %s --json-output: %v (output='%s')", site.Name, err, out)

	redisManifest := getManifestFromLogs(t, out)
	require.NoError(t, err)

	installedOutput, err := exec.RunHostCommand(DdevBin, "add-on", "list", "--installed", "--project", site.Name, "--json-output")
	require.NoError(t, err, "failed ddev add-on list --installed --project %s --json-output: %v (output='%s')", site.Name, err, installedOutput)
	installedManifests := getManifestMapFromLogs(t, installedOutput)

	require.NotEmptyf(t, redisManifest["Version"], "redis manifest is empty: %v", redisManifest)
	assert.Equal(redisManifest["Version"], installedManifests["redis"]["Version"])

	// Remove the add-on using the `--project` flag
	out, err = exec.RunHostCommand(DdevBin, "add-on", "remove", "ddev/ddev-redis", "--project", site.Name)
	require.NoError(t, err, "unable to ddev add-on remove ddev/ddev-redis --project %s: %v, output='%s'", site.Name, err, out)

	// Now make sure we put it back so it can be removed in cleanup
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis", "--project", site.Name)
	assert.NoError(err, "unable to ddev add-on get ddev/ddev-redis --project %s: %v, output='%s'", site.Name, err, out)
}

// TestAddonGetWithPRFlag tests the --pr flag functionality
func TestAddonGetWithPRFlag(t *testing.T) {
	if !github.HasGitHubToken() {
		t.Skip("Skipping because DDEV_GITHUB_TOKEN is not set")
	}
	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert := asrt.New(t)
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "ddev-redis")
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	// Test that --pr flag with invalid value fails
	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis", "--pr", "0")
	require.Error(t, err, "Should fail with --pr=0, out=%s", out)
	require.Contains(t, out, "--pr flag requires a positive integer value")

	// Test that --pr flag with negative value fails
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis", "--pr", "-1")
	require.Error(t, err, "Should fail with --pr=-1, out=%s", out)
	require.Contains(t, out, "--pr flag requires a positive integer value")

	// Test that --pr flag with valid value works (PR #54 exists for ddev-redis)
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis", "--pr", "54")
	require.NoError(t, err, "Should succeed with --pr=54, out=%s", out)
	require.Contains(t, out, "Installing ddev/ddev-redis:pr-54")
}

// TestAddonGetWithVersionFlag tests the --version flag functionality
func TestAddonGetWithVersionFlag(t *testing.T) {
	if !github.HasGitHubToken() {
		t.Skip("Skipping because DDEV_GITHUB_TOKEN is not set")
	}
	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert := asrt.New(t)
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "ddev-redis")
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	// Test that --version flag with empty value fails
	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis", "--version", "")
	require.Error(t, err, "Should fail with --version='', out=%s", out)
	require.Contains(t, out, "--version flag requires a non-empty value")

	// Test that --version flag with valid tag works
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis", "--version", "v2.2.0")
	require.NoError(t, err, "Should succeed with --version=v2.2.0, out=%s", out)
	require.Contains(t, out, "Installing ddev/ddev-redis:v2.2.0")

	// Clean up before next test
	_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "ddev-redis")

	// Test that --version flag with branch name works
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis", "--version", "main")
	require.NoError(t, err, "Should succeed with --version=main, out=%s", out)
	require.Contains(t, out, "Installing ddev/ddev-redis:main")

	// Clean up before next test
	_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "ddev-redis")

	// Test that --version flag with commit SHA works
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis", "--version", "b50ac77")
	require.NoError(t, err, "Should succeed with --version=b50ac77, out=%s", out)
	require.Contains(t, out, "Installing ddev/ddev-redis:b50ac77")
}

// TestAddonGetWithDefaultBranchFlag tests the --default-branch flag functionality
func TestAddonGetWithDefaultBranchFlag(t *testing.T) {
	if !github.HasGitHubToken() {
		t.Skip("Skipping because DDEV_GITHUB_TOKEN is not set")
	}
	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert := asrt.New(t)
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "ddev-redis")
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	// Test that --default-branch flag works and installs from default branch
	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis", "--default-branch")
	require.NoError(t, err, "Should succeed with --default-branch, out=%s", out)
	// Should install from main branch (default_branch for ddev-redis)
	require.Contains(t, out, "Installing ddev/ddev-redis:main")
}
