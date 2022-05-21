package dockerutil_test

import (
	"github.com/drud/ddev/pkg/dockerutil"
	exec2 "github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

var DdevBin = "ddev"

// TestDockerComposeDownload
func TestDockerComposeDownload(t *testing.T) {
	assert := asrt.New(t)
	var err error

	if os.Getenv("DDEV_BINARY_FULLPATH") != "" {
		DdevBin = os.Getenv("DDEV_BINARY_FULLPATH")
	}

	origHome := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		origHome = os.Getenv("USERPROFILE")
	}
	tmpHome := testcommon.CreateTmpDir(t.Name() + "tempHome")
	// Unusual case where we need to alter the RequiredDockerComposeVersion
	// just so we can make sure the one in PATH is different.
	origRequiredComposeVersion := globalconfig.RequiredDockerComposeVersion

	// Change the homedir temporarily
	_ = os.Setenv("HOME", tmpHome)
	_ = os.Setenv("USERPROFILE", tmpHome)

	// Make sure we have the .ddev/bin dir we need to verify against
	//err = fileutil.CopyDir(filepath.Join(origHome, ".ddev/bin"), filepath.Join(tmpHome, ".ddev/bin"))
	//require.NoError(t, err)

	t.Cleanup(func() {
		_, err := os.Stat(globalconfig.GetMutagenPath())
		if err == nil {
			out, err := exec2.RunHostCommand(DdevBin, "debug", "mutagen", "daemon", "stop")
			assert.NoError(err, "mutagen daemon stop returned %s", string(out))
		}

		err = os.RemoveAll(tmpHome)
		assert.NoError(err)
		_ = os.Setenv("HOME", origHome)
		globalconfig.RequiredDockerComposeVersion = origRequiredComposeVersion
	})

	// Download the normal required version specified in code
	globalconfig.DockerComposeVersion = ""

	downloaded, err := dockerutil.DownloadDockerComposeIfNeeded()
	assert.NoError(err)
	assert.True(downloaded)
	v, err := dockerutil.GetLiveDockerComposeVersion()
	assert.NoError(err)
	assert.Equal(globalconfig.GetRequiredDockerComposeVersion(), v)

	// Make sure it doesn't download a second time
	downloaded, err = dockerutil.DownloadDockerComposeIfNeeded()
	assert.NoError(err)
	assert.False(downloaded)

	for _, v := range []string{"v2.0.1", "v1.29.0"} {
		globalconfig.DockerComposeVersion = ""
		// Skip v1 tests on arm64, as they aren't provided
		if strings.HasPrefix(v, "v1") && runtime.GOARCH == "arm64" {
			continue
		}
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
	globalconfig.RequiredDockerComposeVersion = "v2.1.0"
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
