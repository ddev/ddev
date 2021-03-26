package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

// TestHomeadditions makes sure that extra files added to
// .ddev/homeadditions and ~/.ddev/homeadditions get added into the container's ~/
func TestHomeadditions(t *testing.T) {
	assert := asrt.New(t)

	pwd, _ := os.Getwd()
	testdata := filepath.Join(pwd, "testdata", t.Name())

	tmpHome := testcommon.CreateTmpDir(t.Name() + "tempHome")
	origHome := os.Getenv("HOME")
	// Change the homedir temporarily
	err := os.Setenv("HOME", tmpHome)
	require.NoError(t, err)

	site := TestSites[0]
	switchDir := TestSites[0].Chdir()
	projectHomeadditionsDir := filepath.Join(site.Dir, ".ddev", "homeadditions")

	// We can't use the standard getGlobalDDevDir here because *our* global hasn't changed.
	// It's changed via $HOME for the ddev subprocess
	err = os.MkdirAll(filepath.Join(tmpHome, ".ddev"), 0755)
	assert.NoError(err)
	tmpHomeGlobalHomeadditionsDir := filepath.Join(tmpHome, ".ddev", "homeadditions")
	err = os.RemoveAll(tmpHomeGlobalHomeadditionsDir)
	assert.NoError(err)
	err = os.RemoveAll(projectHomeadditionsDir)
	assert.NoError(err)
	err = fileutil.CopyDir(filepath.Join(testdata, "global"), tmpHomeGlobalHomeadditionsDir)
	assert.NoError(err)
	err = fileutil.CopyDir(filepath.Join(testdata, "project"), projectHomeadditionsDir)
	assert.NoError(err)

	defer func() {
		_ = fileutil.PurgeDirectory(projectHomeadditionsDir)
		_ = os.RemoveAll(tmpHome)
		_ = os.Setenv("HOME", origHome)
		switchDir()
	}()

	// Simply run "ddev" to make sure homeadditions example files get populated
	_, err = exec.RunCommand(DdevBin, []string{})
	assert.NoError(err)

	for _, f := range []string{"bash_aliases.example", "README.txt"} {
		assert.FileExists(filepath.Join(projectHomeadditionsDir, f))
		assert.FileExists(filepath.Join(tmpHomeGlobalHomeadditionsDir, f))
	}

	app, err := ddevapp.GetActiveApp(site.Name)
	require.NoError(t, err)

	_, err = exec.RunCommand(DdevBin, []string{"start", "-y"})
	assert.NoError(err)

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
}
