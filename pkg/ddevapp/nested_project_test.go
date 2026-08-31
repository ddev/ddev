package ddevapp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
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
	ddevapp.ResetNestedProjectState()
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(origDir))
		testcommon.CleanupDir(parentDir)
		delete(globalconfig.DdevProjectList, parentName)
		delete(globalconfig.DdevProjectList, childName)
		ddevapp.ResetNestedProjectState()
	})

	// With neither of them registered, the nearest one wins as it always has.
	got, err := ddevapp.GetActiveAppRoot("")
	require.NoError(t, err)
	require.Equal(t, childDir, got)

	// A registered project outside the nested one wins.
	globalconfig.DdevProjectList[parentName] = &globalconfig.ProjectInfo{AppRoot: parentDir}
	got, err = ddevapp.GetActiveAppRoot("")
	require.NoError(t, err)
	require.Equal(t, parentDir, got)

	// `ddev config` still configures the directory it's in, so the nested project can be
	// registered without the `ddev start` prompt.
	got, err = ddevapp.CheckForConf(childDir)
	require.NoError(t, err)
	require.Equal(t, childDir, got)

	// Registering the nested project pins it.
	globalconfig.DdevProjectList[childName] = &globalconfig.ProjectInfo{AppRoot: childDir}
	got, err = ddevapp.GetActiveAppRoot("")
	require.NoError(t, err)
	require.Equal(t, childDir, got)
}

// TestNestedProjectPrompt validates that after PromptToUseNestedProject(), which project
// gets used follows the answer - here the "no" default, since DDEV_NONINTERACTIVE can't ask.
func TestNestedProjectPrompt(t *testing.T) {
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
	ddevapp.ResetNestedProjectState()
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(origDir))
		testcommon.CleanupDir(parentDir)
		delete(globalconfig.DdevProjectList, parentName)
		ddevapp.ResetNestedProjectState()
	})

	ddevapp.PromptToUseNestedProject(false)
	got, err := ddevapp.GetActiveAppRoot("")
	require.NoError(t, err)
	require.Equal(t, parentDir, got, "declining should leave the outer project in place")

	// `-y` doesn't ask, and doesn't take it as a yes either.
	ddevapp.ResetNestedProjectState()
	ddevapp.PromptToUseNestedProject(true)
	got, err = ddevapp.GetActiveAppRoot("")
	require.NoError(t, err)
	require.Equal(t, parentDir, got)
}
