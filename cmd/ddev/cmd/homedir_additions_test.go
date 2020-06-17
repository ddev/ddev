package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

// TestHomedirAdditions makes sure that extra files added to
// .ddev/homeadditions and ~/.ddev/homeadditions get added into the container's ~/
func TestHomedirAdditions(t *testing.T) {
	assert := asrt.New(t)

	pwd, _ := os.Getwd()
	testdata := filepath.Join(pwd, "testdata", t.Name())

	site := TestSites[0]
	switchDir := TestSites[0].Chdir()
	projectHomeadditionsDir := filepath.Join(site.Dir, ".ddev", "homeadditions")
	globalHomeadditionsDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "homeadditions")
	defer func() {
		_ = fileutil.PurgeDirectory(projectHomeadditionsDir)
		// Note that this erases the global homeadditions directory on the
		// machine it's run on.
		_ = fileutil.PurgeDirectory(globalHomeadditionsDir)
		switchDir()
	}()

	// Simply run "ddev" to make sure homeadditions example files get populated
	_, err := exec.RunCommand(DdevBin, []string{})
	assert.NoError(err)

	assert.FileExists(filepath.Join(projectHomeadditionsDir, "bash_aliases.example"))

	// Copy testdata scripts into project homeadditionsdir
	for _, script := range []string{".myscript.sh", ".projectscript.sh"} {
		err = fileutil.CopyFile(filepath.Join(testdata, "project", script), filepath.Join(projectHomeadditionsDir, script))
		assert.NoError(err)
		err = os.Chmod(filepath.Join(testdata, "project", script), 0777)
		assert.NoError(err)
	}

	// Copy testdata scripts into global homeadditionsdir
	for _, script := range []string{".myscript.sh", ".globalscript.sh"} {
		err = fileutil.CopyFile(filepath.Join(testdata, "global", script), filepath.Join(globalHomeadditionsDir, script))
		assert.NoError(err)
		err = os.Chmod(filepath.Join(globalHomeadditionsDir, script), 0777)
		assert.NoError(err)
	}

	app, err := ddevapp.GetActiveApp(site.Name)
	require.NoError(t, err)

	_, err = exec.RunCommand(DdevBin, []string{"start"})
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
		stdout, _, err := app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Cmd:     fmt.Sprintf("~/.%sscript.sh", script),
		})
		assert.NoError(err)
		assert.Contains(stdout, fmt.Sprintf("this is .%sscript.sh", script))
	}
}
