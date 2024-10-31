package cmd

import (
	"github.com/ddev/ddev/pkg/exec"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestDebugMatchConstraintCmd checks to see match-constraint behaves as expected
func TestDebugMatchConstraintCmd(t *testing.T) {
	version, _ := exec.RunHostCommand(DdevBin, "--version")

	out, err := exec.RunHostCommand(DdevBin, "debug", "match-constraint", "> 1.0")
	require.NoError(t, err, "Match constraint should not have errored for > 1.0, version='%s', out='%s'", version, out)

	out, err = exec.RunHostCommand(DdevBin, "debug", "match-constraint", "< 1.0")
	require.Error(t, err, "Match constraint should have errored for < 1.0, version='%s', out='%s'", version, out)
}
