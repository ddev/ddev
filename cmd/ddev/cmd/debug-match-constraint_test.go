package cmd

import (
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

// TestDebugMatchConstraintCmd checks to see match-constraint behaves as expected
// @see https://github.com/Masterminds/semver#checking-version-constraints
func TestDebugMatchConstraintCmd(t *testing.T) {

	constraint := ">= 1.0"
	v, _ := exec.RunHostCommand(DdevBin, "--version")
	v = strings.Trim(v, "\r\n ")
	t.Logf("DdevBin='%s', version from binary='%s', version from versionconstants='%s'", DdevBin, v, constraint)

	out, err := exec.RunHostCommand(DdevBin, "debug", "match-constraint", constraint)
	require.NoError(t, err, "Match constraint should not have errored for %s, out='%s'", constraint, out)

	constraint := "< 1.0"
	out, err = exec.RunHostCommand(DdevBin, "debug", "match-constraint", constraint)
	require.Error(t, err, "Match constraint should have errored for %s, out='%s'", constraint, out)
}
