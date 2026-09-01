package cmd

import (
	"encoding/json"
	"testing"

	"github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/versionconstants"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmdVersion(t *testing.T) {
	assert := asrt.New(t)
	versionData := make(map[string]any)

	args := []string{"version", "--json-output"}
	out, err := exec.RunHostCommandSeparateStreams(DdevBin, args...)
	assert.NoError(err)
	err = json.Unmarshal([]byte(out), &versionData)
	require.NoError(t, err, "failed to unmarshal version output '%v'", out)

	raw, ok := versionData["raw"].(map[string]any)
	require.True(t, ok, "raw section wasn't found in versioninfo %v", out)

	assert.Equal(versionconstants.DdevVersion, raw["DDEV version"])
	assert.Equal(docker.GetWebImage(), raw["web"])
	assert.Equal(docker.GetDBImage(nodeps.MariaDB, ""), raw["db"])
	dockerVersion, err := dockerutil.GetDockerVersion()
	require.NoError(t, err)
	assert.Equal(dockerVersion, raw["docker"])
	dockerAPIVersion, err := dockerutil.GetDockerAPIVersion()
	require.NoError(t, err)
	assert.Equal(dockerAPIVersion, raw["docker-api"])
	buildxVersion, err := dockerutil.GetDockerBuildxVersion()
	require.NoError(t, err)
	assert.Equal(buildxVersion, raw["docker-buildx"])

	assert.Contains(versionData["msg"], versionconstants.DdevVersion)
	assert.Contains(versionData["msg"], versionconstants.WebImg)
	assert.Contains(versionData["msg"], docker.ResolveImageTag(versionconstants.WebTag, versionconstants.WebTagBranch))
	assert.Contains(versionData["msg"], versionconstants.DBImg)
	assert.Contains(versionData["msg"], docker.GetDBImage(nodeps.MariaDB, nodeps.MariaDBDefaultVersion))
	assert.Contains(versionData["msg"], dockerVersion)
	assert.Contains(versionData["msg"], dockerAPIVersion)
	assert.Contains(versionData["msg"], buildxVersion)

	// The table shown to the user appends a branch hint to an image whose tag
	// wasn't already resolved to that branch; the raw JSON map used by
	// scripting never has it.
	branchHint := "(" + versionconstants.WebTagBranch + ")"
	if docker.ResolveImageTag(versionconstants.WebTag, versionconstants.WebTagBranch) == versionconstants.WebTagBranch {
		assert.NotContains(versionData["msg"], branchHint)
	} else {
		assert.Contains(versionData["msg"], branchHint)
	}
	_, ok = raw["image-tag-branches"]
	assert.False(ok, "image-tag-branches should not appear in the raw section")
}
