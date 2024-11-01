package cmd

import (
	"fmt"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/stretchr/testify/require"
	"regexp"
	"strings"
	"testing"
)

// TestDebugMatchConstraintCmd checks to see match-constraint behaves as expected
// @see https://github.com/Masterminds/semver#checking-version-constraints
func TestDebugMatchConstraintCmd(t *testing.T) {
	out, err := exec.RunHostCommand(DdevBin, "debug", "match-constraint", "-h")
	require.NoError(t, err, "Match constraint should not have errored for help, out='%s'", out)
	require.Contains(t, out, "Check if the currently installed ddev matches the specified version constraint")

	constraint := ">= 1.twentythree"
	out, err = exec.RunHostCommand(DdevBin, "debug", "match-constraint", constraint)
	require.Error(t, err, "Match constraint should have errored for %s, out='%s'", constraint, out)
	require.Contains(t, out, "constraint is not valid")

	constraint = "!= v0.0.0-overridden-by-make"
	out, err = exec.RunHostCommand(DdevBin, "debug", "match-constraint", constraint)
	require.NoError(t, err, "Match constraint should not have errored for %s, out='%s'", constraint, out)

	// Get the real DDEV version from `ddev --version` (versionconstants.DdevVersion won't work here)
	out, err = exec.RunHostCommand(DdevBin, "--version")
	require.NoError(t, err, "DDEV version should not have errored, out='%s'", out)
	versionParts := strings.Split(strings.TrimSpace(out), " ")
	ddevVersion := versionParts[len(versionParts)-1]

	if !regexp.MustCompile(`^v[0-9]+\.`).MatchString(ddevVersion) {
		t.Skip(fmt.Sprintf("Skipping check for semver because DDEV version doesn't start with any valid version tag, it's '%v'", ddevVersion))
	}

	constraint = ">= 1.0"
	out, err = exec.RunHostCommand(DdevBin, "debug", "match-constraint", constraint)
	require.NoError(t, err, "Match constraint should not have errored for %s, out='%s'", constraint, out)

	constraint = "< 1.0"
	out, err = exec.RunHostCommand(DdevBin, "debug", "match-constraint", constraint)
	require.Error(t, err, "Match constraint should have errored for %s, out='%s'", constraint, out)
}
