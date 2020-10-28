package cmd_test

import (
	"github.com/drud/ddev/cmd/ddev/cmd"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
)

// TestCmdAuthSSH runs `ddev auth ssh` and checks that it actually worked out.
func TestCmdAuthSSH(t *testing.T) {
	assert := asrt.New(t)
	if !util.IsCommandAvailable("expect") {
		t.Skip("Skipping TestCmdAuthSSH because expect scripting tool is not available")
	}

	testDir, _ := os.Getwd()
	defer cmd.TestSites[0].Chdir()()

	_, err := exec.RunCommand(cmd.DdevBin, []string{"start"})
	require.NoError(t, err)
	// nolint: errcheck
	defer exec.RunCommand(cmd.DdevBin, []string{"stop", "--remove-data", "--omit-snapshot"})

	// Delete any existing identities from ddev-ssh-agent
	_, err = exec.RunCommand("docker", []string{"exec", "ddev-ssh-agent", "ssh-add", "-D"})
	assert.NoError(err)

	// Run a simple ssh server to act on and get its internal IP address
	_, err = exec.RunCommand("docker", []string{"run", "-d", "--name=test-cmd-ssh-server", "--network=ddev_default", "drud/test-ssh-server:v1.16.0"})
	assert.NoError(err)
	//nolint: errcheck
	defer dockerutil.RemoveContainer("test-cmd-ssh-server", 10)

	internalIPAddr, err := exec.RunCommand("docker", []string{"inspect", "-f", "'{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}'", "test-cmd-ssh-server"})
	internalIPAddr = strings.Trim(internalIPAddr, "\n\"'")
	assert.NoError(err)

	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	app.DockerEnv()

	// Before we add the password with ddev auth ssh, we should not be able to access the ssh server
	// Turn off StrictHostChecking because the server can have been run more than once with different
	// identity
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "ssh -o BatchMode=yes -o StrictHostKeyChecking=false root@" + internalIPAddr + " pwd",
	})
	assert.Error(err)

	// Now we add the key with passphrase
	testAuthSSHDir := filepath.Join(testDir, "testdata", "TestCmdAuthSSH")
	err = os.Chmod(filepath.Join(testAuthSSHDir, ".ssh", "id_rsa"), 0600)
	assert.NoError(err)
	sshDir := filepath.Join(testAuthSSHDir, ".ssh")
	out, err := exec.RunCommand("expect", []string{filepath.Join(testAuthSSHDir, "ddevauthssh.expect"), cmd.DdevBin, sshDir, "testkey"})
	assert.NoError(err)
	assert.Contains(string(out), "Identity added:")

	// And at this point we should be able to ssh into the test-cmd-ssh-server
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "ssh -o BatchMode=yes -o StrictHostKeyChecking=false root@" + internalIPAddr + " pwd",
	})
	assert.NoError(err)
	assert.Contains(out, "/root")

}
