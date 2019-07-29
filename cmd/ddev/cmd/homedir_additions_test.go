package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

// TestHomedirAdditions makes sure that extra files added to
// .ddev/homeadditions get added into the container's ~/
func TestHomedirAdditions(t *testing.T) {
	assert := asrt.New(t)

	pwd, _ := os.Getwd()

	site := TestSites[0]
	switchDir := TestSites[0].Chdir()
	defer func() {
		_ = fileutil.PurgeDirectory(filepath.Join(site.Dir, ".ddev", "homeadditions"))
		switchDir()
	}()

	err := fileutil.CopyDir(filepath.Join(pwd, "testdata", t.Name(), "assets"), filepath.Join(site.Dir, "assets"))
	assert.NoError(err)
	_, err = exec.RunCommand(DdevBin, []string{})
	assert.NoError(err)

	homeadditionsDir := filepath.Join(site.Dir, ".ddev", "homeadditions")
	assert.FileExists(filepath.Join(homeadditionsDir, "bash_aliases.example"))

	err = fileutil.CopyFile(filepath.Join(pwd, "testdata", t.Name(), ".myscript.sh"), filepath.Join(homeadditionsDir, ".myscript.sh"))
	assert.NoError(err)
	err = os.Chmod(filepath.Join(homeadditionsDir, ".myscript.sh"), 0777)
	assert.NoError(err)

	// There was a bug in upstream packr2 where the ".ddev/assets" directory was getting
	// populated from the project's "assets" directory.
	assert.False(fileutil.FileExists(filepath.Join(site.Dir, ".ddev", "assets")))

	app, err := ddevapp.GetActiveApp(site.Name)
	require.NoError(t, err)

	_, err = exec.RunCommand(DdevBin, []string{"start"})
	assert.NoError(err)

	stdout, stderr, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "~/.myscript.sh",
	})
	assert.NoError(err)
	_ = stderr
	assert.Contains(stdout, "this is myscript.sh")
}
