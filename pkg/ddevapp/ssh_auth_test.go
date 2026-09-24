package ddevapp_test

import (
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ddev/ddev/cmd/ddev/cmd"
	"github.com/ddev/ddev/pkg/ddevapp"
	ddevImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSSHAuth tests basic SSH authentication
func TestSSHAuth(t *testing.T) {
	if dockerutil.IsColima() {
		t.Skip("Skipping on Colima")
	}
	assert := asrt.New(t)
	origDir, _ := os.Getwd()
	app := &ddevapp.DdevApp{}

	runTime := util.TimeTrackC(t.Name())

	//  Add a docker-compose service that has SSH server and mounted authorized_keys
	site := TestSites[0]
	// If running this with GOTEST_SHORT we have to create the directory, tarball etc.
	if site.Dir == "" || !fileutil.FileExists(site.Dir) {
		err := site.Prepare()
		if err != nil {
			t.Fatalf("Prepare() failed on TestSite.Prepare() site=%s, err=%v", site.Name, err)
		}
	}

	testcommon.ClearDockerEnv()

	err := app.Init(site.Dir)
	require.NoError(t, err, "app.Init() failed on site %s in dir %s, err=%v", site.Name, site.Dir, err)
	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(app.GetConfigPath(".ssh"))
		_ = os.RemoveAll(app.GetConfigPath("docker-compose.sshserver.yaml"))
	})
	srcDdev := filepath.Join(origDir, "testdata", t.Name(), ".ddev")
	err = fileutil.CopyDir(filepath.Join(srcDdev, ".ssh"), app.GetConfigPath(".ssh"))
	require.NoError(t, err)
	err = util.Chmod(app.GetConfigPath(".ssh"), 0700)
	require.NoError(t, err)
	err = util.Chmod(app.GetConfigPath(".ssh/authorized_keys"), 0600)
	require.NoError(t, err)
	err = util.Chmod(app.GetConfigPath(".ssh/id_rsa"), 0600)
	require.NoError(t, err)
	err = fileutil.CopyFile(filepath.Join(srcDdev, "docker-compose.sshserver.yaml"), app.GetConfigPath("docker-compose.sshserver.yaml"))
	require.NoError(t, err)

	// Start with the testsite stopped (and everything stopped)
	err = app.Stop(true, false)
	assert.NoError(err)

	// Make absolutely sure the ssh-agent is created from scratch.
	err = ddevapp.RemoveSSHAgentContainer()
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	err = app.EnsureSSHAgentContainer()
	require.NoError(t, err)

	// Try a simple ssh (with no auth set up), it should fail with "Permission denied"
	_, stderr, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "rm -f /home/.ssh-agent/known_hosts; ssh -o BatchMode=yes root@test-ssh-server pwd",
	})

	assert.Error(err)
	assert.Contains(stderr, "Permission denied")

	// Add the passphrase-protected key the same way `ddev auth ssh` does, driving the
	// prompt with expect. The key is added by explicit path because bare `ssh-add` would
	// abort with "No user found with uid" - the host user has no account in the image.
	expectScript := `
set timeout 10
spawn ssh-add id_rsa
expect {
    "Enter passphrase" { send "testkey\n" }
    eof { puts "FAIL: ssh-add exited before prompting for a passphrase"; exit 2 }
    timeout { puts "FAIL: timed out waiting for passphrase prompt"; exit 2 }
}
expect {
    "Identity added" {}
    eof { puts "FAIL: ssh-add exited without adding the key"; exit 3 }
    timeout { puts "FAIL: timed out waiting for the key to be added"; exit 3 }
}
expect eof
`
	expectCmd := "expect -c '" + expectScript + "'"
	uid, gid, _ := dockerutil.GetContainerUser()
	if dockerutil.IsDockerRootless() {
		uid, gid = "0", "0"
	}
	sshKeyPath := app.GetConfigPath(".ssh")

	// No `-t`: expect drives the passphrase prompt over its own pty, so we don't need
	// a TTY on the container and can capture the output to assert on it.
	args := []string{"run", "--rm", "--volumes-from=" + ddevapp.SSHAuthName, "-v", sshKeyPath + ":/tmp/sshtmp", "-u", uid + ":" + gid}
	containerCmd := "docker"
	if dockerutil.IsPodmanRootless() {
		containerCmd = "podman"
	}
	// keep-id is only correct on Linux Podman rootless; on macOS the VM's
	// subuid/subgid range cannot map host GIDs so keep-id must not be used.
	// Using it here would mismatch the userns of the ddev-ssh-agent container
	// (which also omits keep-id on macOS), causing "Permission denied" on the socket.
	if dockerutil.UseKeepID() {
		args = append(args, "--userns=keep-id")
	}
	args = append(args, "--entrypoint", "bash", ddevImages.GetSSHAuthImage(), "-c", cmd.GetAuthSSHCmd(expectCmd))

	out, err := exec.RunHostCommand(containerCmd, args...)
	require.NoError(t, err, "ssh-add via expect failed; output:\n%s", out)
	require.Contains(t, out, "Identity added", "ssh-add did not report success; output:\n%s", out)

	// Try SSH, should succeed
	stdout, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "rm -f /home/.ssh-agent/known_hosts; ssh root@test-ssh-server pwd",
	})
	stdout = strings.Trim(stdout, "\r\n")
	assert.Equal("/root", stdout)
	assert.NoError(err)

	err = app.Stop(true, false)
	assert.NoError(err)

	// Now start it up again; we shouldn't need to add the key this time
	err = app.Start()
	require.NoError(t, err)

	// Try SSH, should succeed
	stdout, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "rm -f /home/.ssh-agent/known_hosts; ssh root@test-ssh-server pwd",
	})
	stdout = strings.Trim(stdout, "\r\n")
	assert.Equal("/root", stdout)
	assert.NoError(err)

	err = app.Stop(true, false)
	assert.NoError(err)

	runTime()
}

// TestSshAuthConfigOverride tests that the ~/.ddev/.ssh-auth-compose-compose.yaml can be overridden
// with ~/.ddev/ssh-auth-compose.*.yaml
func TestSshAuthConfigOverride(t *testing.T) {
	assert := asrt.New(t)
	pwd, _ := os.Getwd()
	testDir := testcommon.CreateTmpDir(t.Name())
	_ = os.Chdir(testDir)
	overrideYaml := filepath.Join(globalconfig.GetGlobalDdevDir(), "ssh-auth-compose.override.yaml")

	// Remove the ddev-ssh-agent, since the start code checks to see if it's
	// running and doesn't restart it if it's running
	_ = dockerutil.RemoveContainer("ddev-ssh-agent")

	testcommon.ClearDockerEnv()

	app, err := ddevapp.NewApp(testDir, true)
	assert.NoError(err)
	err = app.WriteConfig()
	assert.NoError(err)
	err = fileutil.CopyFile(filepath.Join(pwd, "testdata", t.Name(), "ssh-auth-compose.override.yaml"), overrideYaml)
	assert.NoError(err)

	answer := fileutil.RandomFilenameBase()
	t.Setenv("ANSWER", answer)
	assert.NoError(err)
	assert.NoError(err)
	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(pwd)
		assert.NoError(err)
		_ = os.RemoveAll(testDir)
		_ = os.Remove(overrideYaml)
	})

	err = app.Start()
	assert.NoError(err)

	stdout, _, err := dockerutil.Exec("ddev-ssh-agent", "bash -c 'echo $ANSWER'", "")
	assert.Equal(answer+"\n", stdout)
}

// TestSSHAgentUpstream checks that ssh_agent_upstream turns ddev-ssh-agent
// into a relay to an existing agent, and that clearing it restores the default.
func TestSSHAgentUpstream(t *testing.T) {
	origUpstream := globalconfig.DdevGlobalConfig.SSHAgentUpstream
	t.Cleanup(func() {
		globalconfig.DdevGlobalConfig.SSHAgentUpstream = origUpstream
		_ = dockerutil.RemoveContainer(ddevapp.SSHAuthName)
	})
	app := &ddevapp.DdevApp{}

	globalconfig.DdevGlobalConfig.SSHAgentUpstream = "relative/agent.sock"
	_, err := ddevapp.SSHAgentUpstreamSocket()
	require.Error(t, err)
	_ = dockerutil.RemoveContainer(ddevapp.SSHAuthName)
	err = app.EnsureSSHAgentContainer()
	require.NoError(t, err)
	stdout, _, err := dockerutil.Exec(ddevapp.SSHAuthName, "killall -0 ssh-agent", "")
	require.NoError(t, err, stdout)

	upstreamDir := testcommon.CreateTmpDir(t.Name())
	t.Cleanup(func() {
		_ = os.RemoveAll(upstreamDir)
	})
	upstreamSock := filepath.Join(upstreamDir, "agent.sock")
	globalconfig.DdevGlobalConfig.SSHAgentUpstream = upstreamSock
	sock, err := ddevapp.SSHAgentUpstreamSocket()
	require.NoError(t, err)
	require.Equal(t, upstreamSock, sock)

	_, err = app.CreateSSHAuthComposeFile()
	require.NoError(t, err)
	rendered, err := fileutil.ReadFileIntoString(ddevapp.SSHAuthComposeYAMLPath())
	require.NoError(t, err)
	require.Contains(t, rendered, upstreamDir+":/upstream")
	require.Contains(t, rendered, "UNIX-CONNECT:/upstream/agent.sock")
	require.Contains(t, rendered, "killall -0 socat")

	// A socket in a host directory reaches containers only when Docker runs
	// on the host itself, not through a VM file share.
	sshAgentPath, lookErr := osexec.LookPath("ssh-agent")
	if runtime.GOOS != "linux" || dockerutil.IsDockerDesktop() || lookErr != nil {
		t.Log("Skipping live relay check: needs Linux with native Docker and ssh-agent")
		return
	}
	out, err := exec.RunHostCommand(sshAgentPath, "-a", upstreamSock)
	require.NoError(t, err, out)
	t.Cleanup(func() {
		_, _ = exec.RunHostCommand("bash", "-c", "SSH_AUTH_SOCK="+upstreamSock+" ssh-agent -k || pkill -f 'ssh-agent -a "+upstreamSock+"'")
	})
	keyFile := filepath.Join(upstreamDir, "id_ed25519")
	out, err = exec.RunHostCommand("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", "ddev-upstream-test", "-f", keyFile)
	require.NoError(t, err, out)
	out, err = exec.RunHostCommand("bash", "-c", "SSH_AUTH_SOCK="+upstreamSock+" ssh-add "+keyFile)
	require.NoError(t, err, out)

	err = app.EnsureSSHAgentContainer()
	require.NoError(t, err)
	stdout, stderr, err := dockerutil.Exec(ddevapp.SSHAuthName, "ssh-add -l", "")
	require.NoError(t, err, stderr)
	require.Contains(t, stdout, "ddev-upstream-test")

	globalconfig.DdevGlobalConfig.SSHAgentUpstream = ""
	err = app.EnsureSSHAgentContainer()
	require.NoError(t, err)
	stdout, _, _ = dockerutil.Exec(ddevapp.SSHAuthName, "ssh-add -l", "")
	require.NotContains(t, stdout, "ddev-upstream-test")
}
