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
