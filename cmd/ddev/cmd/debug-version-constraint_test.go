package cmd

import (
	"encoding/json"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/versionconstants"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDebugVersionConstraintCmd tests that ddev debug version-constraint works
func TestDebugVersionConstraintCmd(t *testing.T) {
	assert := asrt.New(t)

	versionConstraint := ">= 1.twentythree"
	out, err := exec.RunHostCommand(DdevBin, "debug", "version-constraint", versionConstraint)
	assert.Error(err)
	assert.Contains(out, "constraint is invalid")

	out, err = exec.RunHostCommand(DdevBin, "debug", "version-constraint")
	assert.Error(err)
	assert.Contains(out, "This command only takes one optional argument")

	versionConstraint = "< " + versionconstants.DdevVersion
	out, err = exec.RunHostCommand(DdevBin, "debug", "version-constraint", versionConstraint)
	assert.Error(err)
	assert.Contains(out, "doesn't meet the constraint")

	versionConstraint = ">= " + versionconstants.DdevVersion
	out, err = exec.RunHostCommand(DdevBin, "debug", "version-constraint", versionConstraint)
	assert.NoError(err)
	assert.Equal(out, "true\n")

	versionConstraint = ">= " + versionconstants.DdevVersion
	out, err = exec.RunHostCommand(DdevBin, "debug", "-j", "version-constraint", versionConstraint)
	assert.NoError(err)
	jsonResult := make(map[string]interface{})
	err = json.Unmarshal([]byte(out), &jsonResult)
	require.NoError(t, err, "failed to unmarshal version-constraint '%v'", out)
	rawResult, ok := jsonResult["raw"]
	require.True(t, ok, "raw section wasn't found in version-constraint: %v", out)
	require.Equal(t, "true", rawResult.(string))
}
