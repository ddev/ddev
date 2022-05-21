package cmd

import (
	"encoding/json"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/versionconstants"
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

	assert.Equal(versionconstants.DdevVersion, raw["DDEV version"])
	assert.Equal(versionconstants.WebImg+":"+versionconstants.WebTag, raw["web"])
	assert.Equal(versionconstants.GetDBImage(nodeps.MariaDB), raw["db"])
	assert.Equal(versionconstants.GetDBAImage(), raw["dba"])
	dockerVersion, _ := dockerutil.GetDockerVersion()
	assert.Equal(dockerVersion, raw["docker"])
	composeVersion, _ := dockerutil.GetDockerComposeVersion()
	assert.Equal(composeVersion, raw["docker-compose"])

	assert.Contains(versionData["msg"], versionconstants.DdevVersion)
	assert.Contains(versionData["msg"], versionconstants.WebImg)
	assert.Contains(versionData["msg"], versionconstants.WebTag)
	assert.Contains(versionData["msg"], versionconstants.DBImg)
	assert.Contains(versionData["msg"], versionconstants.GetDBImage(nodeps.MariaDB, nodeps.MariaDBDefaultVersion))
	assert.Contains(versionData["msg"], versionconstants.DBAImg)
	assert.Contains(versionData["msg"], versionconstants.DBATag)
	assert.NotEmpty(dockerutil.DockerVersion)
	assert.NotEmpty(globalconfig.DockerComposeVersion)
	assert.Contains(versionData["msg"], dockerutil.DockerVersion)
	assert.Contains(versionData["msg"], globalconfig.DockerComposeVersion)
}
