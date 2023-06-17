package ddevapp_test

import (
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
)

// TestGitPull ensures we can pull backups from a git repository
func TestGitPull(t *testing.T) {
	assert := asrt.New(t)
	var err error

	testDir, _ := os.Getwd()

	siteDir := testcommon.CreateTmpDir(t.Name())

	err = os.Chdir(siteDir)
	assert.NoError(err)
	app, err := NewApp(siteDir, true)
	assert.NoError(err)
	app.Name = t.Name()
	app.Type = nodeps.AppTypeDrupal9
	app.Docroot = "web"
	err = app.Stop(true, false)
	require.NoError(t, err)
	err = app.WriteConfig()
	require.NoError(t, err)

	testcommon.ClearDockerEnv()

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)

		_ = os.Chdir(testDir)
		_ = os.RemoveAll(siteDir)
		home, _ := homedir.Dir()
		_ = os.RemoveAll(filepath.Join(home, "tmp", "ddev-pull-git-test-repo"))
	})

	err = PopulateExamplesCommandsHomeadditions(app.Name)
	require.NoError(t, err)

	// Build our git.yaml from the example file
	s, err := os.ReadFile(app.GetConfigPath("providers/git.yaml.example"))
	require.NoError(t, err)
	err = os.WriteFile(app.GetConfigPath("providers/git.yaml"), []byte(s), 0666)
	assert.NoError(err)
	err = app.WriteConfig()
	require.NoError(t, err)

	provider, err := app.GetProvider("git")
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)
	err = app.Pull(provider, false, false, false)
	assert.NoError(err)

	assert.FileExists(filepath.Join(app.GetHostUploadDirFullPath(), "tmp/veggie-pasta-bake-hero-umami.jpg"))
	out, _, err := app.Exec(&ExecOpts{
		Cmd:     "echo 'select COUNT(*) from users_field_data where mail=\"margaret.hopper@example.com\";' | mysql -N",
		Service: "db",
	})
	assert.NoError(err)
	assert.True(strings.HasPrefix(out, "1\n"))
}
