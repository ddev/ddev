package cmd

import (
	"fmt"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
)

// TestDebugMatchConstraintCmd checks to see match-constraint behaves as expected
// @see https://github.com/Masterminds/semver#checking-version-constraints
func TestDebugMatchConstraintCmd(t *testing.T) {

	versionRegex := regexp.MustCompile(`^v[0-9]+\.`)
	if !versionRegex.MatchString(versionconstants.DdevVersion) {
		t.Skip(fmt.Sprintf("Skipping because ddev version doesn't start with any valid version tag, it's '%v'", versionconstants.DdevVersion))
	}

	constraint := ">= 1.0"
	out, err := exec.RunHostCommand(DdevBin, "debug", "match-constraint", constraint)
	require.NoError(t, err, "Match constraint should not have errored for %s, out='%s'", constraint, out)

	constraint = "< 1.0"
	out, err = exec.RunHostCommand(DdevBin, "debug", "match-constraint", constraint)
	require.Error(t, err, "Match constraint should have errored for %s, out='%s'", constraint, out)
}
