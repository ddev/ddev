package ddevapp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInactiveProjects(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	app := &ddevapp.DdevApp{}

	site := TestSites[0]

	t.Cleanup(func() {
		err := os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
	})
	// Make sure we have one app in existence
	err := app.Init(site.Dir)
	require.NoError(t, err)
	err = app.Start()
	require.NoError(t, err)

	// Stop all sites
	_, err = exec.RunCommand(DdevBin, []string{"stop", "--all"})
	require.NoError(t, err)

	apps, err := ddevapp.GetInactiveProjects()
	require.NoError(t, err)

	assert.Greater(len(apps), 0)
}

func TestExtractProjectNames(t *testing.T) {
	var apps []*ddevapp.DdevApp

	assert := asrt.New(t)

	apps = append(apps, &ddevapp.DdevApp{Name: "bar"})
	apps = append(apps, &ddevapp.DdevApp{Name: "foo"})
	apps = append(apps, &ddevapp.DdevApp{Name: "zoz"})

	names := ddevapp.ExtractProjectNames(apps)
	expectedNames := []string{"bar", "foo", "zoz"}

	assert.Equal(expectedNames, names)
}

// mkDdevConfig creates an empty .ddev/config.yaml under dir.
func mkDdevConfig(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ddev"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".ddev", "config.yaml"), []byte(""), 0644))
}

// registerProject registers root as a known project for the test, so
// FindActiveProjectRoot treats it as configured.
func registerProject(t *testing.T, root string) {
	t.Helper()
	if globalconfig.DdevProjectList == nil {
		globalconfig.DdevProjectList = map[string]*globalconfig.ProjectInfo{}
	}
	globalconfig.DdevProjectList[root] = &globalconfig.ProjectInfo{AppRoot: root}
	t.Cleanup(func() { delete(globalconfig.DdevProjectList, root) })
}

// TestCheckForConf validates that CheckForConf returns the nearest project root.
func TestCheckForConf(t *testing.T) {
	parentDir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(parentDir)
	childDir := filepath.Join(parentDir, "child")
	require.NoError(t, os.MkdirAll(childDir, 0755))

	_, err := ddevapp.CheckForConf(childDir)
	require.Error(t, err)

	mkDdevConfig(t, parentDir)
	got, err := ddevapp.CheckForConf(childDir)
	require.NoError(t, err)
	require.Equal(t, parentDir, got)

	mkDdevConfig(t, childDir)
	got, err = ddevapp.CheckForConf(childDir)
	require.NoError(t, err)
	require.Equal(t, childDir, got)
}

// TestFindActiveProjectRoot validates that a nested project resolves to the nearest
// registered ancestor, skipping a stray nested .ddev/, and warns when it does.
func TestFindActiveProjectRoot(t *testing.T) {
	tmpDir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(tmpDir)
	top := filepath.Join(tmpDir, "top")
	middle := filepath.Join(top, "middle")
	leaf := filepath.Join(middle, "leaf")
	workDir := filepath.Join(leaf, "work")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// No project found anywhere.
	_, err := ddevapp.FindActiveProjectRoot(workDir)
	require.Error(t, err)

	// Single, not-yet-configured project at the top: keep nearest (unchanged behavior).
	mkDdevConfig(t, top)
	got, err := ddevapp.FindActiveProjectRoot(workDir)
	require.NoError(t, err)
	require.Equal(t, top, got)

	// Only the top is registered: the stray leaf and middle are skipped for the top.
	mkDdevConfig(t, middle)
	mkDdevConfig(t, leaf)
	registerProject(t, top)

	// `ddev config` registers the nested project, so it doesn't warn. Check this before
	// the message below is emitted, since WarningOnce dedupes by message.
	origArgs := os.Args
	os.Args = []string{"ddev", "config"}
	getWarning := util.CaptureUserErr()
	got, err = ddevapp.FindActiveProjectRoot(workDir)
	warning := getWarning()
	os.Args = origArgs
	require.NoError(t, err)
	require.Equal(t, top, got)
	require.NotContains(t, warning, "nested project")

	// Any other command warns about the nearest one.
	getWarning = util.CaptureUserErr()
	got, err = ddevapp.FindActiveProjectRoot(workDir)
	warning = getWarning()
	require.NoError(t, err)
	require.Equal(t, top, got)
	require.Contains(t, warning, "nested project")
	require.Contains(t, warning, leaf)

	// Registering the leaf pins it: the nearest registered project wins, no warning.
	registerProject(t, leaf)
	getWarning = util.CaptureUserErr()
	got, err = ddevapp.FindActiveProjectRoot(workDir)
	warning = getWarning()
	require.NoError(t, err)
	require.Equal(t, leaf, got)
	require.NotContains(t, warning, "nested project")
}

// TestGetActiveAppRootNested verifies the active project resolves to the configured
// parent when a stray, unconfigured .ddev/ is nested inside it.
func TestGetActiveAppRootNested(t *testing.T) {
	parentDir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(parentDir)
	childDir := filepath.Join(parentDir, "child")
	require.NoError(t, os.MkdirAll(childDir, 0755))
	mkDdevConfig(t, parentDir)
	mkDdevConfig(t, childDir)

	// Only the parent is a configured project; the child is a stray nested .ddev/.
	// Register the resolved path because GetActiveAppRoot resolves cwd via os.Getwd().
	resolvedParent, err := filepath.EvalSymlinks(parentDir)
	require.NoError(t, err)
	registerProject(t, resolvedParent)

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(childDir))

	got, err := ddevapp.GetActiveAppRoot("")
	require.NoError(t, err)
	gotResolved, err := filepath.EvalSymlinks(got)
	require.NoError(t, err)
	require.Equal(t, resolvedParent, gotResolved)
}

// TestGetRelativeWorkingDirectory validates GetRelativeWorkingDirectory
func TestGetRelativeWorkingDirectory(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	app := &ddevapp.DdevApp{}

	site := TestSites[0]

	t.Cleanup(func() {
		err := os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		_ = os.RemoveAll("one")
	})
	// Make sure we have one app in existence
	err := app.Init(site.Dir)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(app.AppRoot, "one", "two", "three", "four"), 0755)
	require.NoError(t, err)

	testCases := []string{filepath.Join("one"), filepath.Join("one", "two"), filepath.Join("one", "two", "three"), filepath.Join("one", "two", "three", "four")}
	for _, c := range testCases {
		err = os.Chdir(filepath.Join(app.AppRoot, c))
		require.NoError(t, err)
		relDir := app.GetRelativeWorkingDirectory()
		assert.Equal(filepath.ToSlash(c), filepath.ToSlash(relDir))
	}
}
