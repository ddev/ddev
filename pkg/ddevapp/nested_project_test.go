package ddevapp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

// TestNestedProject validates that a nested project is used only when it's registered, or
// when nothing registered contains it.
func TestNestedProject(t *testing.T) {
	parentDir := testcommon.CreateTmpDir(t.Name())
	childDir := filepath.Join(parentDir, "child")
	for _, dir := range []string{parentDir, childDir} {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ddev"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".ddev", "config.yaml"), nil, 0644))
	}
	parentName := filepath.Base(parentDir)
	childName := parentName + "-child"
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(childDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(origDir))
		testcommon.CleanupDir(parentDir)
		delete(globalconfig.DdevProjectList, parentName)
		delete(globalconfig.DdevProjectList, childName)
	})

	// With neither of them registered, the nearest one wins as it always has.
	got, err := ddevapp.GetActiveAppRoot("")
	require.NoError(t, err)
	require.Equal(t, childDir, got)

	// A registered project outside the nested one wins, and says so.
	globalconfig.DdevProjectList[parentName] = &globalconfig.ProjectInfo{AppRoot: parentDir}
	getWarning := util.CaptureUserErr()
	got, err = ddevapp.GetActiveAppRoot("")
	warning := getWarning()
	require.NoError(t, err)
	require.Equal(t, parentDir, got)
	require.Contains(t, warning, childDir)

	// `ddev config` still configures the directory it's in, so the nested project can be
	// registered without the `ddev start` prompt.
	got, err = ddevapp.CheckForConf(childDir)
	require.NoError(t, err)
	require.Equal(t, childDir, got)

	// Registering the nested project pins it, without a warning.
	globalconfig.DdevProjectList[childName] = &globalconfig.ProjectInfo{AppRoot: childDir}
	getWarning = util.CaptureUserErr()
	got, err = ddevapp.GetActiveAppRoot("")
	warning = getWarning()
	require.NoError(t, err)
	require.Equal(t, childDir, got)
	require.Empty(t, warning)
}

// TestNestedProjectBareStartDetection validates the plumbing cmd/ddev/cmd relies on to tell
// a `ddev start`/`ddev add` with flags apart from a truly bare one: IsBareStartInvocation,
// once overridden, offers the opt-in prompt instead of warning; and until it's overridden
// (as during push.go/pull.go's own init(), which resolves the active project even earlier
// than cmd/ddev/cmd wires this up), IsBareStartInvocationKnown keeps a "start"/"add" os.Args
// from warning on a guess that might be wrong.
func TestNestedProjectBareStartDetection(t *testing.T) {
	t.Setenv("DDEV_NONINTERACTIVE", "true")

	parentDir := testcommon.CreateTmpDir(t.Name())
	childDir := filepath.Join(parentDir, "child")
	for _, dir := range []string{parentDir, childDir} {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ddev"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".ddev", "config.yaml"), nil, 0644))
	}
	parentName := filepath.Base(parentDir)
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(childDir))
	globalconfig.DdevProjectList[parentName] = &globalconfig.ProjectInfo{AppRoot: parentDir}

	origArgs := os.Args
	origIsBareStart := ddevapp.IsBareStartInvocation
	origKnown := ddevapp.IsBareStartInvocationKnown
	origTokens := ddevapp.BareStartTokens
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(origDir))
		testcommon.CleanupDir(parentDir)
		delete(globalconfig.DdevProjectList, parentName)
		os.Args = origArgs
		ddevapp.IsBareStartInvocation = origIsBareStart
		ddevapp.IsBareStartInvocationKnown = origKnown
		ddevapp.BareStartTokens = origTokens
	})

	// `ddev start -y`: once cmd/ddev/cmd's Cobra-aware check (simulated here) recognizes this
	// as bare start despite the flag, it offers the prompt rather than warning.
	os.Args = []string{"ddev", "start", "-y"}
	ddevapp.IsBareStartInvocation = func() bool { return true }
	ddevapp.IsBareStartInvocationKnown = true
	getWarning := util.CaptureUserErr()
	got, err := ddevapp.GetActiveAppRoot("")
	warning := getWarning()
	require.NoError(t, err)
	require.Equal(t, parentDir, got)
	require.Empty(t, warning, "a recognized bare start should prompt, not warn")

	// Before that recognition is wired up, the default os.Args check can't tell the flag
	// apart from a bare start and says "false" - IsBareStartInvocationKnown holds off the
	// warning rather than contradict the correct prompt that's about to follow.
	ddevapp.IsBareStartInvocation = origIsBareStart
	ddevapp.IsBareStartInvocationKnown = false
	getWarning = util.CaptureUserErr()
	got, err = ddevapp.GetActiveAppRoot("")
	warning = getWarning()
	require.NoError(t, err)
	require.Equal(t, parentDir, got)
	require.Empty(t, warning, "an unresolved start/add token shouldn't warn before Cobra can tell flags apart from a bare invocation")

	// `ddev add myproject`: once wired up, a check that correctly says this isn't bare (a
	// project name was given) warns as usual.
	os.Args = []string{"ddev", "add", "myproject"}
	ddevapp.IsBareStartInvocation = func() bool { return false }
	ddevapp.IsBareStartInvocationKnown = true
	getWarning = util.CaptureUserErr()
	got, err = ddevapp.GetActiveAppRoot("")
	warning = getWarning()
	require.NoError(t, err)
	require.Equal(t, parentDir, got)
	require.Contains(t, warning, childDir)
}
