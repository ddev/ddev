package ddevapp_test

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
	assert2 "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

// TestMailpit does a basic test of mailpit.
func TestMailpit(t *testing.T) {
	assert := assert2.New(t)

	testcommon.ClearDockerEnv()

	origDir, _ := os.Getwd()
	testDir := testcommon.CreateTmpDir(t.Name())

	app, err := ddevapp.NewApp(testDir, false)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.RemoveAll(testDir)
		assert.NoError(err)
	})

	_ = os.RemoveAll(testDir)
	err = fileutil.CopyDir(filepath.Join(origDir, "testdata", t.Name()), testDir)
	require.NoError(t, err)

	err = app.WriteConfig()
	require.NoError(t, err)

	err = globalconfig.ReadGlobalConfig()
	require.NoError(t, err)

	err = app.Start()
	assert.NoError(err)

	stdout, stderr, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "composer install",
	})
	require.NoError(t, err)
	assert.Contains(stderr, "No composer.lock file present. Updating dependencies", "stdout='%s' stderr='%s'", stdout, stderr)

	err = app.MutagenSyncFlush()
	require.NoError(t, err)

	stdout, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "php send_email.php",
	})
	require.NoError(t, err)
	assert.Contains(stdout, "Message sent!")
}
