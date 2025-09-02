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
	require.True(t, info.Size() == 0)

	require.Contains(t, out, fmt.Sprintf("👍 %s", filepath.Join("extra", "has-ddev-generated.txt")))
	require.NotContains(t, out, fmt.Sprintf("👍 %s", filepath.Join("extra", "no-ddev-generated.txt")))
	require.Regexp(t, regexp.MustCompile(fmt.Sprintf(`NOT overwriting [^ ]*%s`, regexp.QuoteMeta(filepath.Join("extra", "no-ddev-generated.txt")))), out)
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
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	err = copy2.Copy(filepath.Join(origDir, "testdata", t.Name(), "project"), app.GetAppRoot())
	require.NoError(t, err)

	t.Cleanup(func() {
		assert := asrt.New(t)
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
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	t.Cleanup(func() {
		assert := asrt.New(t)
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "mock-redis")
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "mock-redis-commander")
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	// Test that installing an addon with dependencies automatically installs the dependency
	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", filepath.Join(origDir, "testdata", "TestRealWorldDependencies", "mock_redis_commander"))
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
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "addon-x")
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "addon-y")
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "addon-z")
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	// Test that circular dependencies are detected and rejected
	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", filepath.Join(origDir, "testdata", "TestAddonCircular", "addon_x"))
	require.Error(t, err, "Should fail due to circular dependency, out=%s", out)
	require.Contains(t, out, "circular dependency detected", "Should mention circular dependency")
}

// TestAddonGetNoDependenciesFlag tests the --no-dependencies flag
func TestAddonGetNoDependenciesFlag(t *testing.T) {
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

	// Test that --no-dependencies flag prevents automatic installation
	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", "--no-dependencies", filepath.Join(origDir, "testdata", "TestRealWorldDependencies", "mock_redis_commander"))
	require.Error(t, err, "Should fail when --no-dependencies is used without dependency installed, out=%s", out)
	require.Contains(t, out, "declares a dependency on '../mock_redis'", "Should mention missing dependency")

	// Install dependency first
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", filepath.Join(origDir, "testdata", "TestRealWorldDependencies", "mock_redis"))
	require.NoError(t, err, "Should install dependency successfully, out=%s", out)

	// Now --no-dependencies should work
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", "--no-dependencies", filepath.Join(origDir, "testdata", "TestRealWorldDependencies", "mock_redis_commander"))
	require.NoError(t, err, "Should install with --no-dependencies when dependency exists, out=%s", out)
	require.NotContains(t, out, "Installing missing dependency", "Should not automatically install dependencies")
}
