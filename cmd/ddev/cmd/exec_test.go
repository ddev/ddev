package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"

	"github.com/ddev/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
)

// TestCmdExecBadArgs run `ddev exec` without the proper args
func TestCmdExecBadArgs(t *testing.T) {
	// Change to the first DevTestSite for the duration of this test.
	defer TestSites[0].Chdir()()
	assert := asrt.New(t)

	args := []string{"exec"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Usage:")
}

// TestCmdExec runs a number of exec commands to verify behavior
func TestCmdExec(t *testing.T) {
	assert := asrt.New(t)
	origDir, err := os.Getwd()
	require.NoError(t, err)
	origDdevDebug := os.Getenv("DDEV_DEBUG")
	_ = os.Unsetenv("DDEV_DEBUG")

	site := TestSites[0]
	err = os.Chdir(site.Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.Setenv("DDEV_DEBUG", origDdevDebug)
	})
	app, err := ddevapp.GetActiveApp(site.Name)
	assert.NoError(err)

	// Test default invocation
	out, err := exec.RunHostCommand(DdevBin, "exec", "pwd")
	assert.NoError(err)
	assert.Contains(out, "/var/www/html")

	// Test specifying service
	out, err = exec.RunHostCommand(DdevBin, "-s", "db", "exec", "pwd")
	assert.NoError(err)
	assert.Contains(out, "/")

	// Test specifying working directory
	out, err = exec.RunHostCommand(DdevBin, "exec", "-d", "/bin", "pwd")
	assert.NoError(err)
	assert.Contains(out, "/bin")

	// Test specifying service and working directory
	out, err = exec.RunHostCommand(DdevBin, "exec", "-s", "db", "-d", "/var", "pwd")
	assert.NoError(err)
	assert.Contains(out, "/var")

	// Test with raw cmd
	out, err = exec.RunHostCommand(DdevBin, "exec", "--raw", "--", "ls", "/usr/local")
	assert.NoError(err)
	assert.True(strings.HasPrefix(out, "bin\netc\ngames\ninclude"), `expected '%v' to start with '%s'`, out, `bin\netc\ngames\ninclude`)

	// Test with raw cmd and bash
	out, err = exec.RunHostCommand(DdevBin, "exec", "--raw", "bash", "-c", "ls /usr/local")
	assert.NoError(err)
	assert.True(strings.HasPrefix(out, "bin\netc\ngames\ninclude"), `expected '%v' to start with '%s'`, out, `bin\netc\games\include`)

	// Test sudo
	out, err = exec.RunHostCommand(DdevBin, "exec", "sudo", "whoami")
	assert.NoError(err)
	assert.Contains(out, "root")

	// Test that an nonexistent working directory generates an error
	out, err = exec.RunHostCommand(DdevBin, "exec", "-d", "/does/not/exist", "pwd")
	assert.Error(err)
	assert.Contains(out, "no such file or directory")

	_, err = exec.RunHostCommand(DdevBin, "exec", "ls >/var/www/html/TestCmdExec-${OSTYPE}.txt")
	assert.NoError(err)
	err = app.MutagenSyncFlush()
	assert.NoError(err)
	assert.FileExists(filepath.Join(site.Dir, "TestCmdExec-linux-gnu.txt"))

	_, err = exec.RunHostCommand(DdevBin, "exec", "ls >/dev/null && touch /var/www/html/TestCmdExec-touch-all-in-one.txt")
	assert.NoError(err)

	// We have created it in the container, must flush before looking for it on host again
	err = app.MutagenSyncFlush()
	assert.NoError(err)

	assert.FileExists(filepath.Join(site.Dir, "TestCmdExec-touch-all-in-one.txt"))

	// Checking how '&&' works in Bash
	bash := util.FindBashPath()
	out, err = exec.RunHostCommand(bash, "-c", fmt.Sprintf("%s exec 'true && pwd'", DdevBin))
	assert.NoError(err)
	assert.Equal("/var/www/html", strings.TrimSpace(out))

	out, err = exec.RunHostCommand(bash, "-c", fmt.Sprintf("%s exec true && pwd", DdevBin))
	assert.NoError(err)
	assert.NotEqual("/var/www/html", strings.TrimSpace(out))

	// Create a bash script for testing
	_, err = exec.RunHostCommand(DdevBin, "exec", "echo 'for i; do echo $i; done' > echo_args.sh")
	assert.NoError(err)
	err = app.MutagenSyncFlush()
	assert.NoError(err)
	assert.FileExists(filepath.Join(site.Dir, "echo_args.sh"))

	// Arguments with spaces should not be split, quotes should be preserved
	out, err = exec.RunHostCommand(DdevBin, "exec", "bash", "echo_args.sh", "string with \"quotes\" and spaces", "another 'quoted' string")
	assert.NoError(err)
	assert.Equal("string with \"quotes\" and spaces\nanother 'quoted' string", strings.TrimSpace(out))

	// Bash expansion should work
	out, err = exec.RunHostCommand(DdevBin, "exec", "echo", "$IS_DDEV_PROJECT")
	assert.NoError(err)
	assert.Equal("true", strings.TrimSpace(out))

	// Bash expansion should work (using arg with spaces)
	out, err = exec.RunHostCommand(DdevBin, "exec", "echo", "$IS_DDEV_PROJECT\n${DDEV_NON_EXISTING_TEST_VARIABLE:-foobar}")
	assert.NoError(err)
	assert.Equal("true\nfoobar", strings.TrimSpace(out))

	// Bash expansion doesn't work with --raw, it's expected
	out, err = exec.RunHostCommand(DdevBin, "exec", "--raw", "echo", "$IS_DDEV_PROJECT")
	assert.NoError(err)
	// Expect a literal string here instead of a variable
	// because this is not being run in Bash, i.e., not `docker-compose exec bash -c "command"`,
	// but `docker-compose exec "command"`
	assert.Equal("$IS_DDEV_PROJECT", strings.TrimSpace(out))

	// Bash expansion doesn't work with --raw, it's expected (using arg with spaces)
	out, err = exec.RunHostCommand(DdevBin, "exec", "--raw", "echo", "$IS_DDEV_PROJECT\n${DDEV_NON_EXISTING_TEST_VARIABLE:-foobar}")
	assert.NoError(err)
	// Expect a literal string here instead of a variable
	// because this is not being run in Bash, i.e., not `docker-compose exec bash -c "command"`,
	// but `docker-compose exec "command"`
	assert.Equal("$IS_DDEV_PROJECT\n${DDEV_NON_EXISTING_TEST_VARIABLE:-foobar}", strings.TrimSpace(out))

	bashPath := util.FindBashPath()
	// Make sure we can pipe things into ddev exec and have them work in stdin inside container
	filename := t.Name() + "_junk.txt"
	_, err = exec.RunHostCommand(bashPath, "-c", fmt.Sprintf(`printf "This file was piped into ddev exec" | %s exec "cat >/var/www/html/%s"`, DdevBin, filename))
	assert.NoError(err)
	err = app.MutagenSyncFlush()
	assert.NoError(err)
	require.FileExists(t, filepath.Join(site.Dir, filename))

	content, err := os.ReadFile(filepath.Join(site.Dir, filename))
	assert.NoError(err)
	assert.Equal("This file was piped into ddev exec", string(content))

	// Make sure that redirection of output from ddev exec works
	f, err := os.CreateTemp("", "")
	err = f.Close()
	assert.NoError(err)
	defer os.Remove(f.Name()) // nolint: errcheck

	bashTempName := f.Name()
	_, err = exec.RunHostCommand(bashPath, "-c", fmt.Sprintf("%s exec ls -l //usr/local/bin/composer >%s", DdevBin, bashTempName))

	out, err = fileutil.ReadFileIntoString(f.Name())
	assert.NoError(err)
	assert.Contains(out, "/usr/local/bin/composer")
}
