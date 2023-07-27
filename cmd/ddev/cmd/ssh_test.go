package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCmdSSH runs `ddev ssh` on basic apps, including with a dot and a dash in them
func TestCmdSSH(t *testing.T) {
	if nodeps.IsAppleSilicon() {
		t.Skip("Skipping TestCmdSSH on Mac M1 because of useless Docker Desktop failures to connect")
	}
	assert := asrt.New(t)

	// Create a temporary directory and change to it for the duration of this test.
	testDir := testcommon.CreateTmpDir(t.Name())
	origDir, _ := os.Getwd()

	err := os.Chdir(testDir)
	require.NoError(t, err)
	app, err := ddevapp.NewApp(testDir, true)
	assert.NoError(err)
	// Projects with dots and dashes in name have been problematic at times, so use that
	app.Name = t.Name() + "." + "betweendots" + "-" + "x"
	err = app.WriteConfig()
	require.NoError(t, err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.RemoveAll(testDir)
		assert.NoError(err)
	})

	err = fileutil.AppendStringToFile("index.php", `
<?php
	mysqli_report(MYSQLI_REPORT_ERROR | MYSQLI_REPORT_STRICT);
	$mysqli = new mysqli('db', 'db', 'db', 'db');
	printf("Success accessing database... %s\n", $mysqli->host_info);
	`)
	require.NoError(t, err)
	err = app.Start()
	require.NoError(t, err)

	_, err = testcommon.EnsureLocalHTTPContent(t, app.GetPrimaryURL(), "Success accessing database")
	assert.NoError(err)

	stdout, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "pwd",
	})
	require.NoError(t, err)
	assert.Equal("/var/www/html\n", stdout)

	b := util.FindBashPath()
	out, err := exec.RunHostCommand(b, "-c", fmt.Sprintf("echo pwd | %s ssh", DdevBin))
	assert.NoError(err)
	assert.Equal("/var/www/html\n", out)

}
