package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/stretchr/testify/require"
)

// TestDebugVersionConstraintCmd tests that ddev debug version-constraint works
func TestDebugVersionConstraintCmd(t *testing.T) {
	versionConstraint := ">= 1.twentythree"
	out, err := exec.RunHostCommand(DdevBin, "debug", "version-constraint", versionConstraint)
	require.Error(t, err)
	require.Contains(t, out, "constraint is invalid")

	out, err = exec.RunHostCommand(DdevBin, "debug", "version-constraint")
	require.Error(t, err)
	require.Contains(t, out, "This command only takes one optional argument")

	if !strings.HasPrefix(versionconstants.DdevVersion, "v1.") {
		t.Skip(fmt.Sprintf("Skipping because ddev version doesn't start with 'v1.', it's '%v'", versionconstants.DdevVersion))
	}

	versionConstraint = "< " + versionconstants.DdevVersion
	out, err = exec.RunHostCommand(DdevBin, "debug", "version-constraint", versionConstraint)
	require.Error(t, err)
	require.Contains(t, out, "doesn't meet the constraint")

	versionConstraint = ">= " + versionconstants.DdevVersion
	out, err = exec.RunHostCommand(DdevBin, "debug", "version-constraint", versionConstraint)
	require.NoError(t, err)
	require.Equal(t, out, "true\n")

	versionConstraint = ">= " + versionconstants.DdevVersion
	out, err = exec.RunHostCommand(DdevBin, "debug", "-j", "version-constraint", versionConstraint)
	require.NoError(t, err)
	jsonResult := make(map[string]interface{})
	err = json.Unmarshal([]byte(out), &jsonResult)
	require.NoError(t, err, "failed to unmarshal version-constraint '%v'", out)
	rawResult, ok := jsonResult["raw"]
	require.True(t, ok, "raw section wasn't found in version-constraint: %v", out)
	require.Equal(t, "true", rawResult.(string))
}
