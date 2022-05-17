package cmd

import (
	"encoding/json"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/version_constants"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
)

func TestCmdVersion(t *testing.T) {
	assert := asrt.New(t)

	versionData := make(map[string]interface{})

	args := []string{"version", "--json-output"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	err = json.Unmarshal([]byte(out), &versionData)
	require.NoError(t, err)

	raw, ok := versionData["raw"].(map[string]interface{})
	require.True(t, ok, "raw section wasn't found in versioninfo %v", out)

	assert.Equal(version_constants.DdevVersion, raw["DDEV version"])
	assert.Equal(version_constants.WebImg+":"+version_constants.WebTag, raw["web"])
	assert.Equal(version_constants.GetDBImage(nodeps.MariaDB), raw["db"])
	assert.Equal(version_constants.GetDBAImage(), raw["dba"])
	dockerVersion, _ := dockerutil.GetDockerVersion()
	assert.Equal(dockerVersion, raw["docker"])
	composeVersion, _ := dockerutil.GetDockerComposeVersion()
	assert.Equal(composeVersion, raw["docker-compose"])

	assert.Contains(versionData["msg"], version_constants.DdevVersion)
	assert.Contains(versionData["msg"], version_constants.WebImg)
	assert.Contains(versionData["msg"], version_constants.WebTag)
	assert.Contains(versionData["msg"], version_constants.DBImg)
	assert.Contains(versionData["msg"], version_constants.GetDBImage(nodeps.MariaDB, nodeps.MariaDBDefaultVersion))
	assert.Contains(versionData["msg"], version_constants.DBAImg)
	assert.Contains(versionData["msg"], version_constants.DBATag)
	assert.NotEmpty(dockerutil.DockerVersion)
	assert.NotEmpty(globalconfig.DockerComposeVersion)
	assert.Contains(versionData["msg"], dockerutil.DockerVersion)
	assert.Contains(versionData["msg"], globalconfig.DockerComposeVersion)
}
