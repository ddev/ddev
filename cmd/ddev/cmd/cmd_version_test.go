package cmd

import (
	"encoding/json"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/version"
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

	assert.Equal(version.DdevVersion, raw["DDEV-Local version"])
	assert.Equal(version.WebImg+":"+version.WebTag, raw["web"])
	assert.Equal(version.GetDBImage(nodeps.MariaDB), raw["db"])
	assert.Equal(version.GetDBAImage(), raw["dba"])
	dockerVersion, _ := version.GetDockerVersion()
	assert.Equal(dockerVersion, raw["docker"])
	composeVersion, _ := version.GetDockerComposeVersion()
	assert.Equal(composeVersion, raw["docker-compose"])

	assert.Contains(versionData["msg"], version.DdevVersion)
	assert.Contains(versionData["msg"], version.WebImg)
	assert.Contains(versionData["msg"], version.WebTag)
	assert.Contains(versionData["msg"], version.DBImg)
	assert.Contains(versionData["msg"], version.GetDBImage(nodeps.MariaDB, nodeps.MariaDBDefaultVersion))
	assert.Contains(versionData["msg"], version.DBAImg)
	assert.Contains(versionData["msg"], version.DBATag)
	assert.NotEmpty(version.DockerVersion)
	assert.NotEmpty(version.DockerComposeVersion)
	assert.Contains(versionData["msg"], version.DockerVersion)
	assert.Contains(versionData["msg"], version.DockerComposeVersion)
}
