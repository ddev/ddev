package ddevapp_test

import (
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
)

// TestFlatfilePull ensures we can pull backups from a flat file for a configured environment.
func TestFlatfilePull(t *testing.T) {
	assert := asrt.New(t)
	var err error

	testDir, _ := os.Getwd()

	siteDir := testcommon.CreateTmpDir(t.Name())

	err = os.Chdir(siteDir)
	assert.NoError(err)
	app, err := NewApp(siteDir, true, "")
	assert.NoError(err)
	app.Name = t.Name()
	app.Type = nodeps.AppTypeDrupal9
	err = app.Stop(true, false)
	require.NoError(t, err)
	err = app.WriteConfig()
	require.NoError(t, err)

	testcommon.ClearDockerEnv()

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)

		_ = os.Chdir(testDir)
		err = os.RemoveAll(siteDir)
		assert.NoError(err)
	})

	_, err = exec.Command(DdevBin).CombinedOutput()
	require.NoError(t, err)

	// Build our Flatfile.yaml from the example file
	s, err := ioutil.ReadFile(app.GetConfigPath("providers/flatfile.yaml.example"))
	require.NoError(t, err)
	x := strings.Replace(string(s), "~/Dropbox", path.Join(dockerutil.MassageWindowsHostMountpoint(testDir), "testdata", t.Name()), -1)
	appRoot := dockerutil.MassageWindowsHostMountpoint(app.AppRoot)
	x = strings.Replace(x, "/full/path/to/project/root", appRoot, -1)
	err = ioutil.WriteFile(app.GetConfigPath("providers/flatfile.yaml"), []byte(x), 0666)
	assert.NoError(err)
	app.Provider = "flatfile"
	err = app.WriteConfig()
	require.NoError(t, err)

	provider, err := app.GetProvider(app.Provider)
	require.NoError(t, err)
	err = app.Start()
	require.NoError(t, err)
	err = app.Pull(provider, &PullOptions{})
	assert.NoError(err)

	assert.FileExists(filepath.Join(app.AppRoot, app.Docroot, app.GetUploadDir(), "docs/developers/building-contributing.md"))
	out, _, err := app.Exec(&ExecOpts{
		Cmd:     "echo 'select COUNT(*) from users_field_data where mail=\"margaret.hopper@example.com\";' | mysql -N",
		Service: "db",
	})
	assert.NoError(err)
	assert.True(strings.HasPrefix(out, "1\n"))
}
