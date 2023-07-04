package cmd

import (
	"encoding/json"
	"testing"

	"github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/versionconstants"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmdVersion(t *testing.T) {
	assert := asrt.New(t)

	versionData := make(map[string]interface{})

	args := []string{"version", "--json-output"}
	out, err := exec.RunHostCommandSeparateStreams(DdevBin, args...)
	assert.NoError(err)
	err = json.Unmarshal([]byte(out), &versionData)
	require.NoError(t, err, "failed to unmarshall version output '%v'", out)

	raw, ok := versionData["raw"].(map[string]interface{})
	require.True(t, ok, "raw section wasn't found in versioninfo %v", out)

	assert.Equal(versionconstants.DdevVersion, raw["DDEV version"])
	assert.Equal(versionconstants.WebImg+":"+versionconstants.WebTag, raw["web"])
	assert.Equal(docker.GetDBImage(nodeps.MariaDB, ""), raw["db"])
	dockerVersion, _ := dockerutil.GetDockerVersion()
	assert.Equal(dockerVersion, raw["docker"])
	composeVersion, _ := dockerutil.GetDockerComposeVersion()
	assert.Equal(composeVersion, raw["docker-compose"])

	assert.Contains(versionData["msg"], versionconstants.DdevVersion)
	assert.Contains(versionData["msg"], versionconstants.WebImg)
	assert.Contains(versionData["msg"], versionconstants.WebTag)
	assert.Contains(versionData["msg"], versionconstants.DBImg)
	assert.Contains(versionData["msg"], docker.GetDBImage(nodeps.MariaDB, nodeps.MariaDBDefaultVersion))
	assert.NotEmpty(dockerutil.DockerVersion)
	assert.NotEmpty(globalconfig.DockerComposeVersion)
	assert.Contains(versionData["msg"], dockerutil.DockerVersion)
	assert.Contains(versionData["msg"], globalconfig.DockerComposeVersion)
}
