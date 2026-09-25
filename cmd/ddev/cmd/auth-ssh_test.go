package cmd_test

import (
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ddev/ddev/cmd/ddev/cmd"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCmdAuthSSH runs `ddev auth ssh` and checks that it actually worked out.
func TestCmdAuthSSH(t *testing.T) {
	if nodeps.IsAppleSilicon() && dockerutil.IsDockerDesktop() {
		t.Skip("Skipping TestCmdAuthSSH on Apple Silicon because of Docker Desktop failures to connect")
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

	// Run a simple SSH server to act on and get its internal IP address
	_, err = exec.RunCommand("docker", []string{"run", "-d", "--name=test-cmd-ssh-server", "--network=ddev_default", "ddev/test-ssh-server:v1.25.4"})
	assert.NoError(err)
	internalIPAddr, err := exec.RunCommand("docker", []string{"inspect", "-f", "'{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}'", "test-cmd-ssh-server"})
	internalIPAddr = strings.Trim(internalIPAddr, "\r\n\"'")
	assert.NoError(err)

	_ = app.DockerEnv()

	// Before we add the password with ddev auth ssh, we should not be able to access the SSH server
	// Turn off StrictHostChecking because the server can have been run more than once with different
	// identity
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "ssh -o BatchMode=yes root@" + internalIPAddr + " pwd",
	})
	assert.Error(err)

	// Now we add the key with passphrase
	testAuthSSHDir := filepath.Join(origDir, "testdata", "TestCmdAuthSSH")
	err = util.Chmod(filepath.Join(testAuthSSHDir, ".ssh", "id_rsa"), 0600)
	assert.NoError(err)
	sshDir := filepath.Join(testAuthSSHDir, ".ssh")
	out, err := exec.RunCommand("expect", []string{filepath.Join(testAuthSSHDir, "ddevauthssh.expect"), cmd.DdevBin, sshDir, "testkey"})
	require.NoError(t, err)
	require.Contains(t, out, "Identity added:")

	// And at this point we should be able to ssh into the test-cmd-ssh-server
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "ssh -o BatchMode=yes root@" + internalIPAddr + " pwd",
	})
	assert.NoError(err)
	assert.Contains(out, "/root")

	// And try to add the same key, but this time provide the passphrase from stdin
	stdin := strings.NewReader("testkey\n")
	out, err = exec.RunHostCommandWithOptions(cmd.DdevBin, []exec.CmdOption{exec.WithStdin(stdin)}, "auth", "ssh", "-d", sshDir)
	require.NoError(t, err, `expected no error for 'printf "testkey\n" | ddev auth ssh -d %s'`, sshDir)
	require.Contains(t, out, "Identity added:")

	// Check for bad passphrase
	stdin = strings.NewReader("foobar\ntestkey\n")
	out, err = exec.RunHostCommandWithOptions(cmd.DdevBin, []exec.CmdOption{exec.WithStdin(stdin)}, "auth", "ssh", "-d", sshDir)
	require.NoError(t, err, `expected no error for 'printf "foobar\ntestkey\n" | ddev auth ssh -d %s'`, sshDir)
	require.Contains(t, out, "Bad passphrase")
	require.Contains(t, out, "Identity added:")
}

// TestCmdAuthSSHUpstream checks that `ddev auth ssh` lists the keys of an
// upstream agent instead of adding key files, and explains a stopped agent.
func TestCmdAuthSSHUpstream(t *testing.T) {
	sshAgentPath, lookErr := osexec.LookPath("ssh-agent")
	// A socket in a host directory reaches containers only when Docker runs
	// on the host itself, not through a VM file share.
	if runtime.GOOS != "linux" || dockerutil.IsDockerDesktop() || lookErr != nil {
		t.Skip("Skipping: needs Linux with native Docker and ssh-agent")
	}

	origUpstream := globalconfig.DdevGlobalConfig.SSHAgentUpstream
	upstreamDir := testcommon.CreateTmpDir(t.Name())
	upstreamSock := filepath.Join(upstreamDir, "agent.sock")
	keyFile := filepath.Join(upstreamDir, "id_ed25519")
	t.Cleanup(func() {
		_, _ = exec.RunHostCommand("pkill", "-f", "ssh-agent -a "+upstreamSock)
		_, _ = exec.RunHostCommand(cmd.DdevBin, "config", "global", "--ssh-agent-upstream="+origUpstream)
		globalconfig.DdevGlobalConfig.SSHAgentUpstream = origUpstream
		_ = dockerutil.RemoveContainer(ddevapp.SSHAuthName)
		_ = os.RemoveAll(upstreamDir)
	})

	out, err := exec.RunHostCommand("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", "ddev-cmd-upstream-test", "-f", keyFile)
	require.NoError(t, err, out)
	out, err = exec.RunHostCommand(sshAgentPath, "-a", upstreamSock)
	require.NoError(t, err, out)
	out, err = exec.RunHostCommand("bash", "-c", "SSH_AUTH_SOCK="+upstreamSock+" ssh-add "+keyFile)
	require.NoError(t, err, out)

	out, err = exec.RunHostCommand(cmd.DdevBin, "config", "global", "--ssh-agent-upstream="+upstreamSock)
	require.NoError(t, err, out)
	out, err = exec.RunHostCommand(cmd.DdevBin, "auth", "ssh")
	require.NoError(t, err, out)
	require.Contains(t, out, "Containers use the SSH agent at "+upstreamSock)
	require.Contains(t, out, "ddev-cmd-upstream-test")

	_, _ = exec.RunHostCommand("pkill", "-f", "ssh-agent -a "+upstreamSock)
	out, err = exec.RunHostCommand(cmd.DdevBin, "auth", "ssh")
	require.Error(t, err, out)
	require.Contains(t, out, "Make sure that agent is running")
}

// TestCmdAuthSSHStdin checks that `ddev auth ssh -f -` adds a key piped to stdin.
func TestCmdAuthSSHStdin(t *testing.T) {
	origUpstream := globalconfig.DdevGlobalConfig.SSHAgentUpstream
	keyDir := testcommon.CreateTmpDir(t.Name())
	keyFile := filepath.Join(keyDir, "id_ed25519")
	t.Cleanup(func() {
		_, _ = exec.RunHostCommand(cmd.DdevBin, "config", "global", "--ssh-agent-upstream="+origUpstream)
		globalconfig.DdevGlobalConfig.SSHAgentUpstream = origUpstream
		_ = dockerutil.RemoveContainer(ddevapp.SSHAuthName)
		_ = os.RemoveAll(keyDir)
	})
	out, err := exec.RunHostCommand(cmd.DdevBin, "config", "global", "--ssh-agent-upstream=")
	require.NoError(t, err, out)
	out, err = exec.RunHostCommand("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", "ddev-stdin-test", "-f", keyFile)
	require.NoError(t, err, out)

	out, err = exec.RunHostCommand("bash", "-c", `"$0" auth ssh -f - < "$1"`, cmd.DdevBin, keyFile)
	require.NoError(t, err, out)
	require.Contains(t, out, "Successfully added the SSH private key from stdin")
	stdout, stderr, err := dockerutil.Exec(ddevapp.SSHAuthName, "ssh-add -l", "")
	require.NoError(t, err, stderr)
	require.Contains(t, stdout, "ddev-stdin-test")

	out, err = exec.RunHostCommand("bash", "-c", `echo "not a key" | "$0" auth ssh -f -`, cmd.DdevBin)
	require.Error(t, err, out)
	require.Contains(t, out, "stdin does not contain an SSH private key")
}
