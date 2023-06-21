package dockerutil_test

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/dockerutil"
	exec2 "github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var DdevBin = "ddev"

// TestDockerComposeDownload
func TestDockerComposeDownload(t *testing.T) {
	assert := asrt.New(t)
	var err error

	if os.Getenv("DDEV_BINARY_FULLPATH") != "" {
		DdevBin = os.Getenv("DDEV_BINARY_FULLPATH")
	}

	tmpHome := testcommon.CreateTmpDir(t.Name() + "tempHome")
	// Unusual case where we need to alter the RequiredDockerComposeVersion
	// just so we can make sure the one in PATH is different.
	origRequiredComposeVersion := globalconfig.DdevGlobalConfig.RequiredDockerComposeVersion

	// Change the homedir temporarily
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	t.Cleanup(func() {
		_, err := os.Stat(globalconfig.GetMutagenPath())
		if err == nil {
			ddevapp.StopMutagenDaemon()
		}

		err = os.RemoveAll(tmpHome)
		assert.NoError(err)
		globalconfig.DdevGlobalConfig.RequiredDockerComposeVersion = origRequiredComposeVersion
		// Reset the cached DockerComposeVersion so it doesn't come into play again
		globalconfig.DockerComposeVersion = ""
		globalconfig.DdevGlobalConfig.UseDockerComposeFromPath = false
	})

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

	for _, v := range []string{"v2.18.0"} {
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

	// Test using docker-compose from path.
	// Make sure our Required/Expected DockerComposeVersion is not something we'd find on the machine
	globalconfig.DockerComposeVersion = ""
	globalconfig.DdevGlobalConfig.RequiredDockerComposeVersion = "v2.5.1"
	globalconfig.DdevGlobalConfig.UseDockerComposeFromPath = true
	activeVersion, err := dockerutil.GetLiveDockerComposeVersion()
	assert.NoError(err)
	path, err := exec.LookPath("docker-compose")
	assert.NoError(err)
	out, err := exec2.RunHostCommand(path, "version", "--short")
	assert.NoError(err)
	parsedFoundVersion := strings.Trim(string(out), "\r\n")
	if !strings.HasPrefix(parsedFoundVersion, "v") {
		parsedFoundVersion = "v" + parsedFoundVersion
	}
	assert.Equal(parsedFoundVersion, activeVersion)
	t.Logf("parsedFoundVersion=%s activeVersion=%s", parsedFoundVersion, activeVersion)
}
