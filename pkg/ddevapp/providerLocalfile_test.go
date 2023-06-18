package ddevapp_test

import (
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
)

// TestLocalfilePull ensures we can pull backups from a flat file for a configured environment.
func TestLocalfilePull(t *testing.T) {
	assert := asrt.New(t)
	var err error

	origDir, _ := os.Getwd()

	tmpDir := testcommon.CreateTmpDir(t.Name())

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	app, err := NewApp(tmpDir, true)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)

		_ = os.Chdir(origDir)
		_ = os.RemoveAll(tmpDir)
	})

	app.Name = t.Name()
	app.Type = nodeps.AppTypeDrupal9
	app.Docroot = "web"
	err = app.Stop(true, false)
	require.NoError(t, err)
	err = app.WriteConfig()
	require.NoError(t, err)

	// This not only shows us the version but also populates the project's
	// .ddev/.global_commands which otherwise doesn't get done until ddev start
	// This matters when --no-bind-mount=true
	out, err := exec.RunHostCommand("ddev", "--version")
	assert.NoError(err)
	t.Logf("ddev --version=%v", out)

	testcommon.ClearDockerEnv()

	err = PopulateExamplesCommandsHomeadditions(app.Name)
	require.NoError(t, err)

	// Build our localfile.yaml from the example file
	s, err := os.ReadFile(app.GetConfigPath("providers/localfile.yaml.example"))
	require.NoError(t, err)
	x := strings.Replace(string(s), "~/Dropbox", path.Join(dockerutil.MassageWindowsHostMountpoint(origDir), "testdata", t.Name()), -1)
	appRoot := dockerutil.MassageWindowsHostMountpoint(app.AppRoot)
	x = strings.Replace(x, "/full/path/to/project/root", appRoot, -1)
	err = os.WriteFile(app.GetConfigPath("providers/localfile.yaml"), []byte(x), 0666)
	assert.NoError(err)
	err = app.WriteConfig()
	require.NoError(t, err)

	provider, err := app.GetProvider("localfile")
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)
	err = app.Pull(provider, false, false, false)
	require.NoError(t, err)

	assert.FileExists(filepath.Join(app.GetHostUploadDirFullPath(), "docs/developers/building-contributing.md"))
	out, _, err = app.Exec(&ExecOpts{
		Cmd:     "echo 'select COUNT(*) from users_field_data where mail=\"margaret.hopper@example.com\";' | mysql -N",
		Service: "db",
	})
	assert.NoError(err)
	assert.True(strings.HasPrefix(out, "1\n"))
}
