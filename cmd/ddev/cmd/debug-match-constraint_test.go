package cmd

import (
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestDebugMatchConstraintCmd checks to see match-constraint behaves as expected
// @see https://github.com/Masterminds/semver#checking-version-constraints
func TestDebugMatchConstraintCmd(t *testing.T) {

	constraint := "= " + versionconstants.DdevVersion
	out, _ := exec.RunHostCommand(DdevBin, "--version")
	util.Debug("DdevBin=%s, version=%s", DdevBin, out)

	out, err := exec.RunHostCommand(DdevBin, "debug", "match-constraint", constraint)
	require.NoError(t, err, "Match constraint should not have errored for %s, out='%s'", constraint, out)

	constraint = "!= " + versionconstants.DdevVersion
	out, err = exec.RunHostCommand(DdevBin, "debug", "match-constraint", constraint)
	require.Error(t, err, "Match constraint should have errored for %s, out='%s'", constraint, out)
}
