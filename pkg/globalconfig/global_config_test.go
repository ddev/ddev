package globalconfig_test

import (
	"context"
	"errors"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/versionconstants"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	globalconfig.EnsureGlobalConfig()
}

// TestGetFreePort checks GetFreePort() to make sure it respects
// ports reserved in DdevGlobalConfig.UsedHostPorts
// and that the port can actually be bound.
func TestGetFreePort(t *testing.T) {
	dockerIP, err := dockerutil.GetDockerIP()
	require.NoError(t, err)

	// Find out a starting port the OS is likely to give us.
	startPort, err := globalconfig.GetFreePort(dockerIP)
	require.NoError(t, err)

	// Put 100 used ports in the UsedHostPorts
	i, err := strconv.Atoi(startPort)
	i = i + 1
	maximum := i + 100
	require.NoError(t, err)
	ports := []string{}
	for ; i < maximum; i++ {
		ports = append(ports, strconv.Itoa(i))
	}
	// Make sure we have a global config set up.
	_ = globalconfig.ReadGlobalConfig()
	err = globalconfig.ReservePorts(t.Name(), ports)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = globalconfig.RemoveProjectInfo(t.Name())
	})

	for range 5 {
		port, err := globalconfig.GetFreePort(dockerIP)
		require.NoError(t, err)
		require.NotContains(t, globalconfig.DdevProjectList["TestGetFreePort"].UsedHostPorts, port)

		// Make sure we can actually use the port.
		dockerCommand := []string{"run", "--rm", "-p" + dockerIP + ":" + port + ":" + port, versionconstants.UtilitiesImage}
		out, err := exec.RunCommand("docker", dockerCommand)

		require.NoError(t, err, "failed to 'docker %v': %v, output='%v'", dockerCommand, err, out)
	}
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

		t.Setenv("HOME", tmpHome)
		t.Setenv("DDEV_XDG_CONFIG_HOME", "")
		t.Setenv("XDG_CONFIG_HOME", "")

		defaultDir := tmpHome + "/.ddev"
		err := os.MkdirAll(defaultDir, 0755)
		require.NoError(t, err)

		err = globalconfig.CheckForMultipleGlobalDdevDirs()
		require.NoError(t, err)
	})

	// Only ~/.config/ddev exists (the Linux fallback), ~/.ddev does not.
	t.Run("OnlyConfigDdevExists", func(t *testing.T) {
		tmpHome := testcommon.CreateTmpDir("TestCheckMultipleDirs_NoConflict")
		defer os.RemoveAll(tmpHome)

		t.Setenv("HOME", tmpHome)
		t.Setenv("DDEV_XDG_CONFIG_HOME", "")
		t.Setenv("XDG_CONFIG_HOME", "")

		xdgDir := tmpHome + "/.config/ddev"
		err := os.MkdirAll(xdgDir, 0755)
		require.NoError(t, err)

		err = globalconfig.CheckForMultipleGlobalDdevDirs()
		require.NoError(t, err)
	})

	// XDG_CONFIG_HOME pointing at the conventional ~/.config on Linux is the
	// recognized fallback, not a leftover, so it must stay silent.
	t.Run("XDGConfigHomeIsConventionalDefault", func(t *testing.T) {
		if !nodeps.IsLinux() {
			t.Skip("~/.config/ddev fallback only applies on Linux")
		}

		tmpHome := testcommon.CreateTmpDir("TestCheckMultipleDirs_XDGDefault")
		defer os.RemoveAll(tmpHome)

		t.Setenv("HOME", tmpHome)
		t.Setenv("DDEV_XDG_CONFIG_HOME", "")
		t.Setenv("XDG_CONFIG_HOME", tmpHome+"/.config")

		xdgDir := tmpHome + "/.config/ddev"
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

		t.Setenv("HOME", tmpHome)
		t.Setenv("DDEV_XDG_CONFIG_HOME", tmpDdevXdg)
		t.Setenv("XDG_CONFIG_HOME", tmpXdg)

		xdgDir := tmpXdg + "/ddev"
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

		t.Setenv("HOME", tmpHome)
		t.Setenv("DDEV_XDG_CONFIG_HOME", "")
		t.Setenv("XDG_CONFIG_HOME", tmpXdg)

		defaultDir := tmpHome + "/.ddev"
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

		t.Setenv("HOME", tmpHome)
		t.Setenv("DDEV_XDG_CONFIG_HOME", "")
		t.Setenv("XDG_CONFIG_HOME", "")

		defaultDir := tmpHome + "/.ddev"
		err := os.MkdirAll(defaultDir, 0755)
		require.NoError(t, err)

		xdgDir := tmpHome + "/.config/ddev"
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

		t.Setenv("HOME", tmpHome)
		t.Setenv("DDEV_XDG_CONFIG_HOME", tmpXdg)
		t.Setenv("XDG_CONFIG_HOME", "")

		defaultDir := tmpHome + "/.ddev"
		err := os.MkdirAll(defaultDir, 0755)
		require.NoError(t, err)

		ddevXdgDir := tmpXdg + "/ddev"
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

		t.Setenv("HOME", tmpHome)
		t.Setenv("DDEV_XDG_CONFIG_HOME", "")
		t.Setenv("XDG_CONFIG_HOME", tmpXdg)

		defaultDir := tmpHome + "/.ddev"
		err := os.MkdirAll(defaultDir, 0755)
		require.NoError(t, err)

		xdgDir := tmpXdg + "/ddev"
		err = os.MkdirAll(xdgDir, 0755)
		require.NoError(t, err)

		err = globalconfig.CheckForMultipleGlobalDdevDirs()
		require.Error(t, err)
		require.Contains(t, err.Error(), "DDEV no longer honors XDG_CONFIG_HOME")
		require.Contains(t, err.Error(), "DDEV_XDG_CONFIG_HOME="+strconv.Quote(tmpXdg))
	})
}
