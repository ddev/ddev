package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestHomeadditions makes sure that extra files added to
// .ddev/homeadditions and ~/.ddev/homeadditions get added into the container's ~/
func TestHomeadditions(t *testing.T) {
	if nodeps.MutagenEnabledDefault || globalconfig.DdevGlobalConfig.MutagenEnabledGlobal || nodeps.NoBindMountsDefault {
		t.Skip("Skipping because this changes homedir and breaks mutagen functionality")
	}
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	testdata := filepath.Join(origDir, "testdata", t.Name())

	tmpHome := testcommon.CreateTmpDir(t.Name() + "tempHome")
	origHome := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		origHome = os.Getenv("USERPROFILE")
	}
	// Change the homedir temporarily
	_ = os.Setenv("HOME", tmpHome)
	_ = os.Setenv("USERPROFILE", tmpHome)

	site := TestSites[0]
	projectHomeadditionsDir := filepath.Join(site.Dir, ".ddev", "homeadditions")

	// We can't use the standard getGlobalDDevDir here because *our* global hasn't changed.
	// It's changed via $HOME for the ddev subprocess
	err := os.MkdirAll(filepath.Join(tmpHome, ".ddev"), 0755)
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
	err = os.Chdir(site.Dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := os.Stat(globalconfig.GetMutagenPath())
		if err == nil {
			out, err := exec.RunHostCommand(DdevBin, "debug", "mutagen", "daemon", "stop")
			assert.NoError(err, "mutagen daemon stop returned %s", string(out))
		}

		err = os.Chdir(origDir)
		assert.NoError(err)
		err = fileutil.PurgeDirectory(projectHomeadditionsDir)
		assert.NoError(err)
		err = os.RemoveAll(tmpHome)
		assert.NoError(err)
		_ = os.Setenv("HOME", origHome)
		_ = os.Setenv("USERPROFILE", origHome)
	})

	// Run ddev start make sure homeadditions example files get populated
	_, err = exec.RunHostCommand(DdevBin, "start")
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
}
