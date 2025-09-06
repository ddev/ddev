package dockerutil_test

import (
	"testing"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckCompose tests detection of docker-compose.
func TestCheckCompose(t *testing.T) {
	assert := asrt.New(t)

	globalconfig.DockerComposeVersion = ""
	composeErr := dockerutil.CheckDockerCompose()
	if composeErr != nil {
		out, err := exec.RunHostCommand(DdevBin, "config", "global")
		require.NoError(t, err)
		ddevVersion, err := exec.RunHostCommand(DdevBin, "version")
		require.NoError(t, err)
		assert.NoError(composeErr, "RequiredDockerComposeVersion=%s global config=%s ddevVersion=%s", globalconfig.DdevGlobalConfig.RequiredDockerComposeVersion, out, ddevVersion)
	}
}
