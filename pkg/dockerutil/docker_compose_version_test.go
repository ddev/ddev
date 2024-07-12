package dockerutil_test

import (
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
