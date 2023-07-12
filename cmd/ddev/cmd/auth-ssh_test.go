package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/cmd/ddev/cmd"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"

	"github.com/ddev/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
)

// TestCmdAuthSSH runs `ddev auth ssh` and checks that it actually worked out.
func TestCmdAuthSSH(t *testing.T) {
	if nodeps.IsAppleSilicon() {
		t.Skip("Skipping TestCmdAuthSSH on Mac M1 because of useless Docker Desktop failures to connect")
	}

	assert := asrt.New(t)
	if !util.IsCommandAvailable("expect") {
		t.Skip("Skipping TestCmdAuthSSH because expect scripting tool is not available")
	}

	origDir, _ := os.Getwd()
	err := os.Chdir(cmd.TestSites[0].Dir)
	require.NoError(t, err)

	app, err := ddevapp.NewApp("", false)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = dockerutil.RemoveContainer("test-cmd-ssh-server")
		assert.NoError(err)
	})

	// Delete any existing identities from ddev-ssh-agent
	_, err = exec.RunCommand("docker", []string{"exec", "ddev-ssh-agent", "ssh-add", "-D"})
	assert.NoError(err)

	// Run a simple ssh server to act on and get its internal IP address
	_, err = exec.RunCommand("docker", []string{"run", "-d", "--name=test-cmd-ssh-server", "--network=ddev_default", "ddev/test-ssh-server:v1.22.0"})
	assert.NoError(err)
	internalIPAddr, err := exec.RunCommand("docker", []string{"inspect", "-f", "'{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}'", "test-cmd-ssh-server"})
	internalIPAddr = strings.Trim(internalIPAddr, "\r\n\"'")
	assert.NoError(err)

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
	testAuthSSHDir := filepath.Join(origDir, "testdata", "TestCmdAuthSSH")
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
