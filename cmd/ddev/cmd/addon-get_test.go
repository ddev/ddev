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
