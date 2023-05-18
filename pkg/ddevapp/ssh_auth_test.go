package ddevapp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/stretchr/testify/require"

	asrt "github.com/stretchr/testify/assert"
)

// TestSSHAuth tests basic ssh authentication
func TestSSHAuth(t *testing.T) {
	assert := asrt.New(t)
	origDir, _ := os.Getwd()
	app := &ddevapp.DdevApp{}

	runTime := util.TimeTrackC(t.Name())

	//  Add a docker-compose service that has ssh server and mounted authorized_keys
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
	err = os.Chmod(app.GetConfigPath(".ssh"), 0700)
	require.NoError(t, err)
	err = os.Chmod(app.GetConfigPath(".ssh/authorized_keys"), 0600)
	require.NoError(t, err)
	err = os.Chmod(app.GetConfigPath(".ssh/id_rsa"), 0600)
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
		Cmd:     "ssh -o BatchMode=yes -o StrictHostKeyChecking=false root@test-ssh-server pwd",
	})

	assert.Error(err)
	assert.Contains(stderr, "Permission denied")

	// Add password/key to auth. This is an unfortunate perversion of using docker run directly, copied from
	// ddev auth ssh command, and with an expect script to provide the passphrase.
	uidStr, _, username := util.GetContainerUIDGid()
	sshKeyPath := app.GetConfigPath(".ssh")
	sshKeyPath = dockerutil.MassageWindowsHostMountpoint(sshKeyPath)

	err = exec.RunInteractiveCommand("docker", []string{"run", "-t", "--rm", "--volumes-from=" + ddevapp.SSHAuthName, "-v", sshKeyPath + ":/home/" + username + "/.ssh", "-u", uidStr, versionconstants.SSHAuthImage + ":" + versionconstants.SSHAuthTag + "-built", "//test.expect.passphrase"})
	require.NoError(t, err)

	// Try ssh, should succeed
	stdout, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "ssh -o StrictHostKeyChecking=false root@test-ssh-server pwd",
	})
	stdout = strings.Trim(stdout, "\r\n")
	assert.Equal(stdout, "/root")
	assert.NoError(err)

	err = app.Stop(true, false)
	assert.NoError(err)

	// Now start it up again; we shouldn't need to add the key this time
	err = app.Start()
	require.NoError(t, err)

	// Try ssh, should succeed
	stdout, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "ssh -o StrictHostKeyChecking=false root@test-ssh-server pwd",
	})
	stdout = strings.Trim(stdout, "\r\n")
	assert.Equal(stdout, "/root")
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

	// Remove the ddev-ssh-agent, since the start code simply checks to see if it's
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
