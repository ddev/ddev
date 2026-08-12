package cmd

import (
	"fmt"
	"os"
	"strings"
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
	if nodeps.IsEnvFalse("DDEV_RUN_TEST_ANYWAY") && nodeps.IsAppleSilicon() {
		t.Skip("Skipping TestCmdSSH on Apple Silicon because of intermittent failures to connect")
	}
	assert := asrt.New(t)
	t.Setenv("DDEV_DEBUG", "")

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
		_ = os.RemoveAll(testDir)
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

	testcommon.AssertLocalHTTPContent(t, app.GetPrimaryURL(), "Success accessing database",
		testcommon.WithMessagef("project web container should be able to reach the database"),
	)

	stdout, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "pwd",
	})
	require.NoError(t, err)
	assert.Equal("/var/www/html\n", stdout)

	b := util.FindBashPath()
	out, err := exec.RunHostCommand(b, "-c", fmt.Sprintf("echo pwd | %s ssh", DdevBin))
	require.True(t, strings.HasPrefix(out, "/var/www/html\n"), "output should start with /var/www/html but is actually '%s'", out)

	t.Setenv("TERM", "xterm-256color")
	t.Setenv("COLORTERM", "truecolor")

	// Here the output is captured, so there is no terminal and nothing to
	// forward. The container has to keep the TERM of its image.
	out, err = exec.RunHostCommand(b, "-c", fmt.Sprintf("echo 'echo MARK TERM=$TERM COLORTERM=$COLORTERM' | %s ssh", DdevBin))
	require.NoError(t, err)
	require.NotContains(t, out, "TERM=xterm-256color", "without a terminal ddev ssh should not forward TERM, got '%s'", out)
	require.NotContains(t, out, "COLORTERM=truecolor", "without a terminal ddev ssh should not forward COLORTERM, got '%s'", out)

	// On a terminal the shell in the container has to see the terminal of the
	// host, otherwise it keeps the TERM of the image and shows only 16 colors.
	if nodeps.IsWindows() {
		t.Skip("skipping the terminal part of this test on Windows, it has no script(1)")
	}
	shellInput := `printf 'echo MARK TERM=$TERM COLORTERM=$COLORTERM\nexit\n'`
	out, err = exec.RunHostCommand(b, "-c", fmt.Sprintf("%s | %s", shellInput, ptyCommand(DdevBin+" ssh")))
	require.NoError(t, err)
	require.Contains(t, out, "MARK TERM=xterm-256color COLORTERM=truecolor", "on a terminal ddev ssh should forward TERM and COLORTERM, got '%s'", out)
}

// ptyCommand wraps a command so that script(1) runs it on a terminal. The
// arguments of script(1) are different on macOS and on Linux.
func ptyCommand(command string) string {
	if nodeps.IsMacOS() {
		return fmt.Sprintf("script -q /dev/null %s", command)
	}
	return fmt.Sprintf("script -q -c '%s' /dev/null", command)
}
