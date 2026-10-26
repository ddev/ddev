package globalconfig_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
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

// TestReadProjectListDropsRemovedProjects checks that rereading the project
// list forgets projects another process removed from the file.
func TestReadProjectListDropsRemovedProjects(t *testing.T) {
	origList := globalconfig.DdevProjectList
	t.Setenv("DDEV_XDG_CONFIG_HOME", t.TempDir())
	t.Cleanup(func() { globalconfig.DdevProjectList = origList })

	globalconfig.DdevProjectList = map[string]*globalconfig.ProjectInfo{
		"kept":    {AppRoot: "/kept"},
		"removed": {AppRoot: "/removed"},
	}
	require.NoError(t, os.WriteFile(globalconfig.GetProjectListPath(), []byte("kept:\n  approot: /kept\n"), 0644))

	require.NoError(t, globalconfig.ReadProjectList())
	require.Equal(t, map[string]*globalconfig.ProjectInfo{"kept": {AppRoot: "/kept"}}, globalconfig.DdevProjectList)
}

// TestUpdateProjectListKeepsOtherProcessChanges checks that a change is made
// to the list on disk, not to this process's startup copy.
func TestUpdateProjectListKeepsOtherProcessChanges(t *testing.T) {
	origList := globalconfig.DdevProjectList
	t.Setenv("DDEV_XDG_CONFIG_HOME", t.TempDir())
	t.Cleanup(func() { globalconfig.DdevProjectList = origList })
	appRoot := t.TempDir()

	require.NoError(t, os.WriteFile(globalconfig.GetProjectListPath(), []byte("a:\n  approot: /a\n"), 0644))
	require.NoError(t, globalconfig.ReadProjectList())
	require.NoError(t, os.WriteFile(globalconfig.GetProjectListPath(), []byte("a:\n  approot: /a\nb:\n  approot: /b\n"), 0644))

	require.NoError(t, globalconfig.SetProjectAppRoot("c", appRoot))
	require.NoError(t, globalconfig.ReadProjectList())
	require.Equal(t, map[string]*globalconfig.ProjectInfo{
		"a": {AppRoot: "/a"},
		"b": {AppRoot: "/b"},
		"c": {AppRoot: appRoot},
	}, globalconfig.DdevProjectList)

	require.NoError(t, os.WriteFile(globalconfig.GetProjectListPath(), []byte("a:\n  approot: /a\nc:\n  approot: "+appRoot+"\nd:\n  approot: /d\n"), 0644))
	require.NoError(t, globalconfig.RemoveProjectInfo("a"))
	require.NoError(t, globalconfig.ReadProjectList())
	require.Equal(t, map[string]*globalconfig.ProjectInfo{
		"c": {AppRoot: appRoot},
		"d": {AppRoot: "/d"},
	}, globalconfig.DdevProjectList)
}

// TestConcurrentProjectListWriters registers projects from several processes
// at once and checks that none is lost.
func TestConcurrentProjectListWriters(t *testing.T) {
	const writers, perWriter = 4, 10
	if prefix := os.Getenv("DDEV_TEST_PROJECT_LIST_WRITER"); prefix != "" {
		appRoot := t.TempDir()
		for i := range perWriter {
			require.NoError(t, globalconfig.UpdateProject(prefix+strconv.Itoa(i), appRoot, []string{}))
		}
		return
	}

	origList := globalconfig.DdevProjectList
	xdg := t.TempDir()
	t.Setenv("DDEV_XDG_CONFIG_HOME", xdg)
	t.Cleanup(func() { globalconfig.DdevProjectList = origList })
	require.NoError(t, os.WriteFile(globalconfig.GetProjectListPath(), nil, 0644))

	errs := make(chan error, writers)
	for w := range writers {
		go func() {
			cmd := exec.Command(os.Args[0], "-test.run=^TestConcurrentProjectListWriters$")
			cmd.Env = append(os.Environ(), "DDEV_XDG_CONFIG_HOME="+xdg, "DDEV_TEST_PROJECT_LIST_WRITER=w"+strconv.Itoa(w)+"-")
			out, err := cmd.CombinedOutput()
			if err != nil {
				err = fmt.Errorf("writer %d: %v\n%s", w, err, out)
			}
			errs <- err
		}()
	}
	for range writers {
		require.NoError(t, <-errs)
	}

	require.NoError(t, globalconfig.ReadProjectList())
	require.Len(t, globalconfig.DdevProjectList, writers*perWriter)
}

// TestProjectListEmptyName checks that an entry with an empty name, as
// `ddev config` used to write with omit_project_name_by_default, is neither
// written nor kept.
func TestProjectListEmptyName(t *testing.T) {
	origList := globalconfig.DdevProjectList
	t.Setenv("DDEV_XDG_CONFIG_HOME", t.TempDir())
	t.Cleanup(func() { globalconfig.DdevProjectList = origList })
	appRoot := t.TempDir()

	require.Error(t, globalconfig.SetProjectAppRoot("", appRoot))
	require.Error(t, globalconfig.ReservePorts("", []string{}))

	require.NoError(t, os.WriteFile(globalconfig.GetProjectListPath(), []byte(`"":\n  approot: `+appRoot+"\n"), 0644))
	require.NoError(t, globalconfig.ReadProjectList())
	require.Empty(t, globalconfig.DdevProjectList)

	require.NoError(t, globalconfig.SetProjectAppRoot("real", appRoot))
	content, err := os.ReadFile(globalconfig.GetProjectListPath())
	require.NoError(t, err)
	require.NotContains(t, string(content), `""`)
}

// TestUpdateGlobalConfigKeepsOtherProcessChanges checks that a field change
// is made to global_config.yaml as it is on disk, not to the startup copy.
func TestUpdateGlobalConfigKeepsOtherProcessChanges(t *testing.T) {
	origConfig := globalconfig.DdevGlobalConfig
	t.Setenv("DDEV_XDG_CONFIG_HOME", t.TempDir())
	t.Cleanup(func() { globalconfig.DdevGlobalConfig = origConfig })

	globalconfig.DdevGlobalConfig = globalconfig.New()
	require.NoError(t, globalconfig.ReadGlobalConfig())

	other := globalconfig.New()
	other.TableStyle = "bright"
	require.NoError(t, globalconfig.WriteGlobalConfig(other))

	require.NoError(t, globalconfig.UpdateGlobalConfig(func(config *globalconfig.GlobalConfig) {
		config.XdebugIDELocation = "wsl2"
	}))

	globalconfig.DdevGlobalConfig = globalconfig.New()
	require.NoError(t, globalconfig.ReadGlobalConfig())
	require.Equal(t, "bright", globalconfig.DdevGlobalConfig.TableStyle)
	require.Equal(t, "wsl2", globalconfig.DdevGlobalConfig.XdebugIDELocation)
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

	// `ddev config global` reads twice before writing; override keys missing
	// from global_config.yaml must still stay out of it.
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "global_config.b.yaml"), []byte("project_tld: b.test\ndeveloper_mode: true\n"), 0644))
	globalconfig.DdevGlobalConfig = globalconfig.New()
	require.NoError(t, globalconfig.ReadGlobalConfig())
	require.NoError(t, globalconfig.ReadGlobalConfig())
	require.True(t, globalconfig.DdevGlobalConfig.DeveloperMode)
	globalconfig.DdevGlobalConfig.TableStyle = "bold"
	require.NoError(t, globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig))
	written, err = os.ReadFile(mainFile)
	require.NoError(t, err)
	require.NotRegexp(t, `(?m)^developer_mode:`, string(written))
	require.NotContains(t, string(written), "b.test")

	// Changing a list an override extends saves only the change, not the override's items.
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "global_config.b.yaml"), []byte("web_environment: [FROM_OVERRIDE=1]\n"), 0644))
	globalconfig.DdevGlobalConfig = globalconfig.New()
	require.NoError(t, globalconfig.ReadGlobalConfig())
	require.Equal(t, []string{"A=changed", "FROM_OVERRIDE=1"}, globalconfig.DdevGlobalConfig.WebEnvironment)
	globalconfig.DdevGlobalConfig.WebEnvironment = []string{"FROM_OVERRIDE=1", "MINE=1"}
	require.NoError(t, globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig))
	written, err = os.ReadFile(mainFile)
	require.NoError(t, err)
	require.Contains(t, string(written), "web_environment:\n    - MINE=1\n")
	require.NotContains(t, string(written), "FROM_OVERRIDE")

	// With override_config: true, a list in the override replaces the one from global_config.yaml.
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "global_config.b.yaml"), []byte("override_config: true\nomit_containers: [ddev-router]\n"), 0644))
	require.NoError(t, os.Remove(filepath.Join(globalDir, "global_config.a.yaml")))
	globalconfig.DdevGlobalConfig = globalconfig.New()
	require.NoError(t, globalconfig.ReadGlobalConfig())
	require.Equal(t, []string{"ddev-router"}, globalconfig.DdevGlobalConfig.OmitContainersGlobal)
}

// TestReadProjectListUnparseable checks that an unparseable project list is
// set aside instead of blocking every command.
func TestReadProjectListUnparseable(t *testing.T) {
	origProjectList := globalconfig.DdevProjectList
	t.Cleanup(func() {
		globalconfig.DdevProjectList = origProjectList
	})
	t.Setenv("DDEV_XDG_CONFIG_HOME", t.TempDir())

	projectListPath := globalconfig.GetProjectListPath()
	garbage := []byte("{{{ not: [yaml\n")
	require.NoError(t, os.WriteFile(projectListPath, garbage, 0644))

	require.NoError(t, globalconfig.ReadProjectList())
	require.Empty(t, globalconfig.DdevProjectList)

	saved, err := os.ReadFile(projectListPath + ".bad")
	require.NoError(t, err)
	require.Equal(t, garbage, saved)

	require.NoError(t, globalconfig.ReadProjectList())
	require.NotNil(t, globalconfig.DdevProjectList)
}
