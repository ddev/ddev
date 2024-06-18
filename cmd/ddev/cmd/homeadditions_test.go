package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHomeadditions makes sure that extra files added to
// .ddev/homeadditions and ~/.ddev/homeadditions get added into the container's ~/
func TestHomeadditions(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	testdata := filepath.Join(origDir, "testdata", t.Name())

	tmpXdgConfigHomeDir := testcommon.CopyGlobalDdevDir(t)

	tmpHomeGlobalHomeadditionsDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "homeadditions")
	err := os.RemoveAll(tmpHomeGlobalHomeadditionsDir)
	assert.NoError(err)
	site := TestSites[0]
	projectHomeadditionsDir := filepath.Join(site.Dir, ".ddev", "homeadditions")
	err = os.RemoveAll(projectHomeadditionsDir)
	assert.NoError(err)
	err = fileutil.CopyDir(filepath.Join(testdata, "global"), tmpHomeGlobalHomeadditionsDir)
	assert.NoError(err)
	err = fileutil.CopyDir(filepath.Join(testdata, "project"), projectHomeadditionsDir)
	assert.NoError(err)
	err = os.Chdir(site.Dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(projectHomeadditionsDir)
		testcommon.ResetGlobalDdevDir(t, tmpXdgConfigHomeDir)
	})

	// Before we can symlink global, need to make sure anything is already gone
	err = os.RemoveAll(filepath.Join(tmpHomeGlobalHomeadditionsDir, "realglobaltarget.txt"))
	assert.NoError(err)
	err = os.RemoveAll(filepath.Join(projectHomeadditionsDir, "realprojecttarget.txt"))
	assert.NoError(err)

	// Symlink the project file
	err = os.Symlink(filepath.Join(origDir, "testdata", t.Name(), "project/realprojecttarget.txt"), filepath.Join(projectHomeadditionsDir, "realprojecttarget.txt"))
	require.NoError(t, err)
	// Symlink the global file
	err = os.Symlink(filepath.Join(origDir, "testdata", t.Name(), "global/realglobaltarget.txt"), filepath.Join(tmpHomeGlobalHomeadditionsDir, "realglobaltarget.txt"))
	require.NoError(t, err)
	// Run DDEV start make sure homeadditions example files get populated
	_, err = exec.RunHostCommand(DdevBin, "restart")
	assert.NoError(err)

	for _, f := range []string{"bash_aliases.example", "README.txt"} {
		assert.FileExists(filepath.Join(projectHomeadditionsDir, f))
		assert.FileExists(filepath.Join(tmpHomeGlobalHomeadditionsDir, f))
	}

	app, err := ddevapp.GetActiveApp(site.Name)
	require.NoError(t, err)

	// Make sure that even though there was a global and a project-level .myscript.sh
	// the project-level one should win.

	stdout, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "~/.myscript.sh",
	})
	assert.NoError(err)
	assert.Contains(stdout, "this is project .myscript.sh")

	for _, script := range []string{"global", "project"} {
		stdout, _, err = app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Cmd:     fmt.Sprintf("~/.%sscript.sh", script),
		})
		assert.NoError(err)
		assert.Contains(stdout, fmt.Sprintf("this is .%sscript.sh", script))
	}
	for _, f := range []string{"realglobaltarget.txt", "realprojecttarget.txt"} {
		_, _, err = app.Exec(&ddevapp.ExecOpts{
			Cmd: `ls ~/` + f,
		})
		assert.NoError(err)
	}
}
