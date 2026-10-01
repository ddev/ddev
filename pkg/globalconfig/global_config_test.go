package globalconfig_test

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	globalconfig.EnsureGlobalConfig()
}

// TestSetProjectAppRoot tests behavior of SetProjectAppRoot
// This also tests RemoveProject
func TestSetProjectAppRoot(t *testing.T) {
	assert := asrt.New(t)

	// Make sure conflicting approot results in error
	// Make sure empty project works
	// Make sure existing project with no approot works

	// Non-existing approot should cause a fail
	err := globalconfig.SetProjectAppRoot(t.Name(), "/nowhere/junk-approot-1")
	assert.Error(err)
	_ = globalconfig.RemoveProjectInfo(t.Name())

	// Create a project in a valid directory
	tmpDir := testcommon.CreateTmpDir(t.Name())

	// Make sure we have valid global config
	_ = globalconfig.ReadGlobalConfig()
	err = globalconfig.SetProjectAppRoot(t.Name(), tmpDir)
	assert.NoError(err)

	t.Cleanup(func() {
		_ = globalconfig.RemoveProjectInfo(t.Name())
		_ = os.RemoveAll(tmpDir)
	})

	project := globalconfig.GetProject(t.Name())
	require.NotNil(t, project)

	// Try to set approot to existing but conflicting approot
	tmpDir2 := testcommon.CreateTmpDir(t.Name())
	// nolint: errcheck
	defer os.RemoveAll(tmpDir2)
	err = globalconfig.SetProjectAppRoot(t.Name(), tmpDir2)
	assert.Error(err)

	// Make sure that the approot didn't accidentally get changed to
	// bad approot
	p2 := globalconfig.GetProject(t.Name())
	assert.Equal(tmpDir, p2.AppRoot)

	err = globalconfig.RemoveProjectInfo(t.Name())
	assert.NoError(err)

	// Make sure after removal the project is gone
	p3 := globalconfig.GetProject(t.Name())
	assert.Nil(p3)

	// ReservePorts will create the project, but without an approot
	err = globalconfig.ReservePorts(t.Name(), []string{})
	assert.NoError(err)
	project = globalconfig.GetProject(t.Name())
	require.NotNil(t, project)
	assert.Empty(project.AppRoot)

	err = globalconfig.SetProjectAppRoot(t.Name(), tmpDir)
	assert.NoError(err)

	project = globalconfig.GetProject(t.Name())
	assert.Equal(tmpDir, project.AppRoot)
}

type internetActiveNetResolverStub struct {
	sleepTime time.Duration
	err       error
}

// LookupIP is a custom version of net.LookupIP that wastes some time and then returns
func (t internetActiveNetResolverStub) LookupIP(ctx context.Context, _, _ string) ([]net.IP, error) {
	select {
	case <-time.After(t.sleepTime):
	case <-ctx.Done():
		return nil, errors.New("context timed out")
	}
	return nil, t.err
}

// internetActiveResetVariables resets the global variables IsInternetActive() uses back to their defaults
func internetActiveResetVariables() {
	globalconfig.IsInternetActiveNetResolver = net.DefaultResolver
	globalconfig.IsInternetActiveAlreadyChecked = false
	globalconfig.IsInternetActiveResult = false
	globalconfig.DdevGlobalConfig.InternetDetectionTimeout = nodeps.InternetDetectionTimeoutDefault
}

// TestIsInternetActiveErrorOccurred tests if IsInternetActive() returns false when LookupIP returns an error
func TestIsInternetActiveErrorOccurred(t *testing.T) {
	internetActiveResetVariables()

	globalconfig.IsInternetActiveNetResolver = internetActiveNetResolverStub{
		sleepTime: 0,
		err:       errors.New("test error"),
	}

	asrt.False(t, globalconfig.IsInternetActive())
}

// TestIsInternetActiveTimeout tests if IsInternetActive() returns false when it times out
func TestIsInternetActiveTimeout(t *testing.T) {
	internetActiveResetVariables()

	globalconfig.IsInternetActiveNetResolver = internetActiveNetResolverStub{
		sleepTime: 4000 * time.Millisecond,
	}

	asrt.False(t, globalconfig.IsInternetActive())
}

// TestIsInternetActiveAlreadyChecked tests if IsInternetActive() returns true when it has already
// been called and returned true on an earlier execution.
func TestIsInternetActiveAlreadyChecked(t *testing.T) {
	internetActiveResetVariables()

	globalconfig.IsInternetActiveAlreadyChecked = true
	globalconfig.IsInternetActiveResult = true

	asrt.True(t, globalconfig.IsInternetActive())
}

// TestIsInternetActive tests if IsInternetActive() returns true, when the LookupIP call goes well
// and if it properly sets the globals so it won't execute the LookupIP again.
func TestIsInternetActive(t *testing.T) {
	internetActiveResetVariables()

	globalconfig.IsInternetActiveNetResolver = internetActiveNetResolverStub{
		sleepTime: 0,
	}

	// should return true
	asrt.True(t, globalconfig.IsInternetActive())
	// should have set the IsInternetActiveAlreadyChecked to true
	asrt.True(t, globalconfig.IsInternetActiveAlreadyChecked)
	// result should still be true
	asrt.True(t, globalconfig.IsInternetActiveResult)
	// and calling it again, should also still be true
	asrt.True(t, globalconfig.IsInternetActive())
}

// TestCheckForMultipleGlobalDdevDirs tests CheckForMultipleGlobalDdevDirs behavior
func TestCheckForMultipleGlobalDdevDirs(t *testing.T) {
	// ===== Happy path: a single recognized location is in use, no warning =====

	// Only the default ~/.ddev exists.
	t.Run("OnlyDefaultExists", func(t *testing.T) {
		tmpHome := testcommon.CreateTmpDir("TestCheckMultipleDirs_Default")
		defer os.RemoveAll(tmpHome)

		testcommon.SetTestHome(t, tmpHome)
		t.Setenv("DDEV_XDG_CONFIG_HOME", "")

		defaultDir := filepath.Join(tmpHome, ".ddev")
		err := os.MkdirAll(defaultDir, 0755)
		require.NoError(t, err)

		err = globalconfig.CheckForMultipleGlobalDdevDirs()
		require.NoError(t, err)
	})

	// Only ~/.config/ddev exists (the Linux fallback), ~/.ddev does not.
	t.Run("OnlyConfigDdevExists", func(t *testing.T) {
		tmpHome := testcommon.CreateTmpDir("TestCheckMultipleDirs_NoConflict")
		defer os.RemoveAll(tmpHome)

		testcommon.SetTestHome(t, tmpHome)
		t.Setenv("DDEV_XDG_CONFIG_HOME", "")

		xdgDir := filepath.Join(tmpHome, ".config", "ddev")
		err := os.MkdirAll(xdgDir, 0755)
		require.NoError(t, err)

		err = globalconfig.CheckForMultipleGlobalDdevDirs()
		require.NoError(t, err)
	})

	// XDG_CONFIG_HOME pointing at ~/.config on Linux
	// doesn't trigger a warning because we recognize ~/.config/ddev
	t.Run("XDGConfigHomeIsConventionalDefault", func(t *testing.T) {
		if !nodeps.IsLinux() {
			t.Skip("~/.config/ddev fallback only applies on Linux")
		}

		tmpHome := testcommon.CreateTmpDir("TestCheckMultipleDirs_XDGDefault")
		defer os.RemoveAll(tmpHome)

		testcommon.SetTestHome(t, tmpHome)
		t.Setenv("DDEV_XDG_CONFIG_HOME", "")
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))

		xdgDir := filepath.Join(tmpHome, ".config", "ddev")
		err := os.MkdirAll(xdgDir, 0755)
		require.NoError(t, err)

		err = globalconfig.CheckForMultipleGlobalDdevDirs()
		require.NoError(t, err)
	})

	// DDEV_XDG_CONFIG_HOME takes precedence over XDG_CONFIG_HOME, so a leftover
	// XDG_CONFIG_HOME/ddev must not be reported when both are set.
	t.Run("DDEVXDGConfigHomeSuppressesXDGLeftover", func(t *testing.T) {
		tmpHome := testcommon.CreateTmpDir("TestCheckMultipleDirs_BothHome")
		defer os.RemoveAll(tmpHome)

		tmpDdevXdg := testcommon.CreateTmpDir("TestCheckMultipleDirs_BothDDEVXDG")
		defer os.RemoveAll(tmpDdevXdg)

		tmpXdg := testcommon.CreateTmpDir("TestCheckMultipleDirs_BothXDG")
		defer os.RemoveAll(tmpXdg)

		testcommon.SetTestHome(t, tmpHome)
		t.Setenv("DDEV_XDG_CONFIG_HOME", tmpDdevXdg)
		t.Setenv("XDG_CONFIG_HOME", tmpXdg)

		xdgDir := filepath.Join(tmpXdg, "ddev")
		err := os.MkdirAll(xdgDir, 0755)
		require.NoError(t, err)

		err = globalconfig.CheckForMultipleGlobalDdevDirs()
		require.NoError(t, err)
	})

	// XDG_CONFIG_HOME is set but has no ddev directory under it, so there is
	// nothing to recover and no warning.
	t.Run("XDGConfigHomeWithoutDdevDir", func(t *testing.T) {
		tmpHome := testcommon.CreateTmpDir("TestCheckMultipleDirs_XDGEmpty")
		defer os.RemoveAll(tmpHome)

		tmpXdg := testcommon.CreateTmpDir("TestCheckMultipleDirs_XDGNoDdev")
		defer os.RemoveAll(tmpXdg)

		testcommon.SetTestHome(t, tmpHome)
		t.Setenv("DDEV_XDG_CONFIG_HOME", "")
		t.Setenv("XDG_CONFIG_HOME", tmpXdg)

		defaultDir := filepath.Join(tmpHome, ".ddev")
		err := os.MkdirAll(defaultDir, 0755)
		require.NoError(t, err)

		err = globalconfig.CheckForMultipleGlobalDdevDirs()
		require.NoError(t, err)
	})

	// ===== Error path: config exists in a location other than the one in use =====

	// Both Linux standard locations exist; ~/.ddev wins by precedence and the
	// message must explain that either can be removed, not name only one.
	t.Run("ConflictBothExist", func(t *testing.T) {
		if !nodeps.IsLinux() {
			t.Skip("This test only runs on Linux")
		}

		tmpHome := testcommon.CreateTmpDir("TestCheckMultipleDirs_Conflict")
		defer os.RemoveAll(tmpHome)

		testcommon.SetTestHome(t, tmpHome)
		t.Setenv("DDEV_XDG_CONFIG_HOME", "")

		defaultDir := filepath.Join(tmpHome, ".ddev")
		err := os.MkdirAll(defaultDir, 0755)
		require.NoError(t, err)

		xdgDir := filepath.Join(tmpHome, ".config", "ddev")
		err = os.MkdirAll(xdgDir, 0755)
		require.NoError(t, err)

		err = globalconfig.CheckForMultipleGlobalDdevDirs()
		require.Error(t, err)
		require.Contains(t, err.Error(), "multiple global DDEV configurations found")
		require.Contains(t, err.Error(), "takes precedence")
		require.Contains(t, err.Error(), "To use "+strconv.Quote(xdgDir)+" instead, remove "+strconv.Quote(defaultDir))
	})

	// DDEV_XDG_CONFIG_HOME is in use while a stale ~/.ddev exists (all platforms).
	t.Run("DDEVXDGConfigHomeSet", func(t *testing.T) {
		tmpHome := testcommon.CreateTmpDir("TestCheckMultipleDirs_DDEVXDGHome")
		defer os.RemoveAll(tmpHome)

		tmpXdg := testcommon.CreateTmpDir("TestCheckMultipleDirs_DDEVXDG")
		defer os.RemoveAll(tmpXdg)

		testcommon.SetTestHome(t, tmpHome)
		t.Setenv("DDEV_XDG_CONFIG_HOME", tmpXdg)

		defaultDir := filepath.Join(tmpHome, ".ddev")
		err := os.MkdirAll(defaultDir, 0755)
		require.NoError(t, err)

		ddevXdgDir := filepath.Join(tmpXdg, "ddev")
		err = os.MkdirAll(ddevXdgDir, 0755)
		require.NoError(t, err)

		err = globalconfig.CheckForMultipleGlobalDdevDirs()
		require.Error(t, err)
		require.Contains(t, err.Error(), "multiple global DDEV configurations found")
	})

	// Leftover XDG_CONFIG_HOME/ddev that DDEV no longer honors (any platform, when
	// it is not the ~/.config/ddev fallback on Linux). The warning must tell the
	// user to set DDEV_XDG_CONFIG_HOME or remove it.
	t.Run("XDGConfigHomeLeftover", func(t *testing.T) {
		tmpHome := testcommon.CreateTmpDir("TestCheckMultipleDirs_XDGHome")
		defer os.RemoveAll(tmpHome)

		tmpXdg := testcommon.CreateTmpDir("TestCheckMultipleDirs_XDG")
		defer os.RemoveAll(tmpXdg)

		testcommon.SetTestHome(t, tmpHome)
		t.Setenv("DDEV_XDG_CONFIG_HOME", "")
		t.Setenv("XDG_CONFIG_HOME", tmpXdg)

		defaultDir := filepath.Join(tmpHome, ".ddev")
		err := os.MkdirAll(defaultDir, 0755)
		require.NoError(t, err)

		xdgDir := filepath.Join(tmpXdg, "ddev")
		err = os.MkdirAll(xdgDir, 0755)
		require.NoError(t, err)

		err = globalconfig.CheckForMultipleGlobalDdevDirs()
		require.Error(t, err)
		require.Contains(t, err.Error(), "DDEV no longer honors XDG_CONFIG_HOME")
		require.Contains(t, err.Error(), "DDEV_XDG_CONFIG_HOME="+strconv.Quote(tmpXdg))
	})
}

// TestGlobalConfigHooksRoundTrip tests that hooks in global_config.yaml survive being read and rewritten.
func TestGlobalConfigHooksRoundTrip(t *testing.T) {
	origConfig := globalconfig.DdevGlobalConfig
	tmpHome := testcommon.CreateTmpDir("TestGlobalConfigHooksRoundTrip")
	testcommon.SetTestHome(t, tmpHome)
	t.Setenv("DDEV_XDG_CONFIG_HOME", "")
	t.Cleanup(func() {
		globalconfig.DdevGlobalConfig = origConfig
		_ = os.RemoveAll(tmpHome)
	})

	globalDir := filepath.Join(tmpHome, ".ddev")
	require.NoError(t, os.MkdirAll(globalDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "global_config.yaml"), []byte("hooks:\n  post-start:\n    - exec: echo hi\n      service: db\n    - exec-host: echo host\n"), 0644))

	globalconfig.DdevGlobalConfig = globalconfig.New()
	require.NoError(t, globalconfig.ReadGlobalConfig())
	expected := map[string][]map[string]any{"post-start": {{"exec": "echo hi", "service": "db"}, {"exec-host": "echo host"}}}
	require.Equal(t, expected, globalconfig.DdevGlobalConfig.Hooks)

	require.NoError(t, globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig))
	globalconfig.DdevGlobalConfig = globalconfig.New()
	require.NoError(t, globalconfig.ReadGlobalConfig())
	require.Equal(t, expected, globalconfig.DdevGlobalConfig.Hooks)
}

// TestGlobalConfigOverrides tests that global_config.*.yaml files are merged over
// global_config.yaml in lexical order, and that saving the global config does not
// copy their values into global_config.yaml.
func TestGlobalConfigOverrides(t *testing.T) {
	origConfig := globalconfig.DdevGlobalConfig
	tmpHome := testcommon.CreateTmpDir("TestGlobalConfigOverrides")
	testcommon.SetTestHome(t, tmpHome)
	t.Setenv("DDEV_XDG_CONFIG_HOME", "")
	t.Cleanup(func() {
		globalconfig.DdevGlobalConfig = origConfig
		_ = os.RemoveAll(tmpHome)
	})

	globalDir := filepath.Join(tmpHome, ".ddev")
	require.NoError(t, os.MkdirAll(globalDir, 0755))
	mainFile := filepath.Join(globalDir, "global_config.yaml")
	require.NoError(t, os.WriteFile(mainFile, []byte("project_tld: main.test\nomit_containers: [ddev-ssh-agent]\nweb_environment: [A=main]\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "global_config.a.yaml"), []byte("project_tld: a.test\nomit_containers: [ddev-router]\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "global_config.b.yaml"), []byte("project_tld: b.test\n"), 0644))

	globalconfig.DdevGlobalConfig = globalconfig.New()
	require.NoError(t, globalconfig.ReadGlobalConfig())
	require.Equal(t, "b.test", globalconfig.DdevGlobalConfig.ProjectTldGlobal)
	require.ElementsMatch(t, []string{"ddev-ssh-agent", "ddev-router"}, globalconfig.DdevGlobalConfig.OmitContainersGlobal)
	require.Equal(t, []string{"A=main"}, globalconfig.DdevGlobalConfig.WebEnvironment)

	// A change to a field no override sets is saved; the overridden values are not.
	globalconfig.DdevGlobalConfig.WebEnvironment = []string{"A=changed"}
	require.NoError(t, globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig))
	written, err := os.ReadFile(mainFile)
	require.NoError(t, err)
	require.Contains(t, string(written), "project_tld: main.test")
	require.NotContains(t, string(written), "b.test")
	require.Contains(t, string(written), "omit_containers: [ddev-ssh-agent]")
	require.Contains(t, string(written), "A=changed")

	// With override_config: true, a list in the override replaces the one from global_config.yaml.
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "global_config.b.yaml"), []byte("override_config: true\nomit_containers: [ddev-router]\n"), 0644))
	require.NoError(t, os.Remove(filepath.Join(globalDir, "global_config.a.yaml")))
	globalconfig.DdevGlobalConfig = globalconfig.New()
	require.NoError(t, globalconfig.ReadGlobalConfig())
	require.Equal(t, []string{"ddev-router"}, globalconfig.DdevGlobalConfig.OmitContainersGlobal)
}
