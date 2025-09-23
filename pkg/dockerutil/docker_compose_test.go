package dockerutil_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var DdevBin = "ddev"

func init() {
	// Make sets DDEV_BINARY_FULLPATH when building the executable
	if os.Getenv("DDEV_BINARY_FULLPATH") != "" {
		DdevBin = os.Getenv("DDEV_BINARY_FULLPATH")
	}
	if os.Getenv("DDEV_TEST_NO_BIND_MOUNTS") == "true" {
		globalconfig.DdevGlobalConfig.NoBindMounts = true
	}

	globalconfig.EnsureGlobalConfig()
}

// TestDockerComposeDownload verifies that we can download a particular docker-compose version
func TestDockerComposeDownload(t *testing.T) {
	assert := asrt.New(t)
	var err error

	_, err = dockerutil.DownloadDockerComposeIfNeeded()
	require.NoError(t, err)

	tmpXdgConfigHomeDir := testcommon.CopyGlobalDdevDir(t)

	t.Cleanup(func() {
		testcommon.ResetGlobalDdevDir(t, tmpXdgConfigHomeDir)
	})

	// Remove previous binary
	previousDockerCompose, _ := globalconfig.GetDockerComposePath()
	_ = os.RemoveAll(previousDockerCompose)

	// Download the normal required version specified in code
	globalconfig.DockerComposeVersion = ""

	downloaded, err := dockerutil.DownloadDockerComposeIfNeeded()
	require.NoError(t, err)
	require.True(t, downloaded)
	v, err := dockerutil.GetLiveDockerComposeVersion()
	assert.NoError(err)
	assert.Equal(globalconfig.GetRequiredDockerComposeVersion(), v)

	// Make sure it doesn't download a second time
	downloaded, err = dockerutil.DownloadDockerComposeIfNeeded()
	assert.NoError(err)
	assert.False(downloaded)

	for _, v := range []string{"v2.32.4"} {
		globalconfig.DockerComposeVersion = ""
		globalconfig.DdevGlobalConfig.RequiredDockerComposeVersion = v
		downloaded, err = dockerutil.DownloadDockerComposeIfNeeded()
		require.NoError(t, err)
		assert.True(downloaded)
		// We have to reset version.DockerComposeVersion so it will actually check
		// instead of using cached value.
		globalconfig.DockerComposeVersion = ""
		activeVersion, err := dockerutil.GetLiveDockerComposeVersion()
		assert.NoError(err)
		assert.Equal(globalconfig.GetRequiredDockerComposeVersion(), activeVersion)
	}
}

// TestComposeCmd tests execution of docker-compose commands.
func TestComposeCmd(t *testing.T) {
	assert := asrt.New(t)

	composeFiles := []string{filepath.Join("testdata", "docker-compose.yml")}

	stdout, stderr, err := dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
		ComposeFiles: composeFiles,
		Action:       []string{"config", "--services"},
	})
	assert.NoError(err)
	assert.Contains(stdout, "web")
	assert.Contains(stdout, "db")
	assert.Contains(stderr, "Defaulting to a blank string")

	composeFiles = append(composeFiles, filepath.Join("testdata", "docker-compose.override.yml"))

	stdout, stderr, err = dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
		ComposeFiles: composeFiles,
		Action:       []string{"config", "--services"},
	})
	assert.NoError(err)
	assert.Contains(stdout, "web")
	assert.Contains(stdout, "db")
	assert.Contains(stdout, "foo")
	assert.Contains(stderr, "Defaulting to a blank string")

	composeFiles = []string{"invalid.yml"}
	_, _, err = dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
		ComposeFiles: composeFiles,
		Action:       []string{"config", "--services"},
	})
	assert.Error(err)
}

// TestComposeWithStreams tests execution of docker-compose commands with streams
func TestComposeWithStreams(t *testing.T) {
	assert := asrt.New(t)

	container, _ := dockerutil.FindContainerByName(t.Name())
	if container != nil {
		_ = dockerutil.RemoveContainer(container.ID)
	}

	// Use the current actual web container for this, so replace in base docker-compose file
	composeBase := filepath.Join("testdata", "TestComposeWithStreams", "test-compose-with-streams.yaml")
	tmpDir := testcommon.CreateTmpDir(t.Name())
	realComposeFile := filepath.Join(tmpDir, "replaced-compose-with-streams.yaml")

	err := fileutil.ReplaceStringInFile("TEST-COMPOSE-WITH-STREAMS-IMAGE", versionconstants.WebImg+":"+versionconstants.WebTag, composeBase, realComposeFile)
	assert.NoError(err)

	composeFiles := []string{realComposeFile}

	t.Cleanup(func() {
		_, _, err = dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
			ComposeFiles: composeFiles,
			Action:       []string{"down"},
		})
		assert.NoError(err)
	})

	_, _, err = dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
		ComposeFiles: composeFiles,
		Action:       []string{"up", "-d"},
	})
	require.NoError(t, err)

	_, err = dockerutil.ContainerWait(60, map[string]string{
		"com.ddev.site-name":        t.Name(),
		"com.docker.compose.oneoff": "False",
	})
	if err != nil {
		logout, _ := exec.RunCommand("docker", []string{"logs", t.Name()})
		inspectOut, _ := exec.RunCommandPipe("sh", []string{"-c", fmt.Sprintf("docker inspect %s|jq -r '.[0].State.Health.Log'", t.Name())})
		t.Fatalf("FAIL: TestComposeWithStreams failed to ContainerWait for container: %v, logs\n========= container logs ======\n%s\n======= end logs =======\n==== health log =====\ninspectOut\n%s\n========", err, logout, inspectOut)
	}

	// Point stdout to os.Stdout and do simple ps -ef in web container
	stdout := util.CaptureStdOut()
	err = dockerutil.ComposeWithStreams(&dockerutil.ComposeCmdOpts{
		ComposeFiles: composeFiles,
		Action:       []string{"exec", "-T", "web", "ps", "-ef"},
	}, os.Stdin, os.Stdout, os.Stderr)
	assert.NoError(err)
	output := stdout()
	assert.Contains(output, "supervisord")

	// Reverse stdout and stderr and create an error and normal stdout. We should see only the error captured in stdout
	stdout = util.CaptureStdOut()
	err = dockerutil.ComposeWithStreams(&dockerutil.ComposeCmdOpts{
		ComposeFiles: composeFiles,
		Action:       []string{"exec", "-T", "web", "ls", "-d", "xx", "/var/run/apache2"},
	}, os.Stdin, os.Stderr, os.Stdout)
	assert.Error(err)
	output = stdout()
	assert.Contains(output, "ls: cannot access 'xx': No such file or directory")

	// Flip stdout and stderr and create an error and normal stdout. We should see only the success captured in stdout
	stdout = util.CaptureStdOut()
	err = dockerutil.ComposeWithStreams(&dockerutil.ComposeCmdOpts{
		ComposeFiles: composeFiles,
		Action:       []string{"exec", "-T", "web", "ls", "-d", "xx", "/var/run/apache2"},
	}, os.Stdin, os.Stdout, os.Stderr)
	assert.Error(err)
	output = stdout()
	assert.Contains(output, "/var/run/apache2", output)
}
