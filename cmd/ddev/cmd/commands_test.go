package cmd

import (
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	osexec "os/exec"
	"path/filepath"
	"testing"
)

// TestCmdCustomCommands does basic checks to make sure custom commands work OK.
func TestCmdCustomCommands(t *testing.T) {
	assert := asrt.New(t)

	pwd, _ := os.Getwd()
	testCmdCustomCommandsDir := filepath.Join(pwd, "testdata", t.Name(), "commands")

	site := TestSites[0]
	switchDir := TestSites[0].Chdir()
	defer func() {
		_ = fileutil.PurgeDirectory(filepath.Join(site.Dir, ".ddev", "commands"))
		switchDir()
	}()

	commandsDir := filepath.Join(site.Dir, ".ddev", "commands")
	assert.FileExists(filepath.Join(commandsDir, "db", "mysql"))
	assert.FileExists(filepath.Join(commandsDir, "host", "mysqlworkbench.example"))
	out, err := exec.RunCommand(DdevBin, []string{})
	assert.NoError(err)
	assert.Contains(out, "mysql client in web container")

	// Test the `ddev mysql` command with stdin
	inputFile := filepath.Join(testCmdCustomCommandsDir, "..", "select99.sql")
	f, err := os.Open(inputFile)
	require.NoError(t, err)
	// nolint: errcheck
	defer f.Close()
	command := osexec.Command(DdevBin, "mysql")
	command.Stdin = f
	byteOut, err := command.CombinedOutput()
	require.NoError(t, err)
	assert.Equal("99\n99\n", string(byteOut))

	// Test ddev mysql -uroot -proot mysql
	command = osexec.Command("bash", "-c", "echo 'SHOW TABLES;' | "+DdevBin+" mysql --user=root --password=root --database=mysql")
	byteOut, err = command.CombinedOutput()
	assert.NoError(err, "byteOut=%v", string(byteOut))
	assert.Contains(string(byteOut), `Tables_in_mysql
column_stats
columns_priv`)

	err = os.RemoveAll(commandsDir)
	assert.NoError(err)

	// Now copy a web and a host command and make sure they show up and execute properly
	err = fileutil.CopyDir(testCmdCustomCommandsDir, commandsDir)
	assert.NoError(err)

	out, err = exec.RunCommand(DdevBin, []string{})
	assert.NoError(err)
	assert.Contains(out, "Test host command (custom host container command)")
	assert.Contains(out, "Test web command (custom web container command)")
	out, err = exec.RunCommand(DdevBin, []string{"testhostcmd", "hostarg1", "hostarg2", "--hostflag1"})
	assert.NoError(err)
	hostname, _ := os.Hostname()
	assert.Contains(out, "Test Host Command was executed with args=hostarg1 hostarg2 --hostflag1 on host "+hostname)

	out, err = exec.RunCommand(DdevBin, []string{"testwebcmd", "webarg1", "webarg2", "--webflag1"})
	assert.NoError(err)
	assert.Contains(out, "Test Web Command was executed with args=webarg1 webarg2 --webflag1 on host "+site.Name+"-web")
}
