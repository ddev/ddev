package cmd

import (
	"testing"

	utils "github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

// TestLegacyExecBadArgs run `drud legacy exec` without the proper args
func TestLegacyExecBadArgs(t *testing.T) {
	args := []string{"legacy", "exec", "-a", LegacyTestApp, "-e", LegacyTestEnv}
	out, err := utils.RunCommand(DrudBin, args)
	assert.Error(t, err)
	assert.Contains(t, string(out), "Must pass a command as first argument.")

	args = []string{"legacy", "exec", "pwd"}
	out, err = utils.RunCommand(DrudBin, args)
	assert.Error(t, err)
	assert.Contains(t, string(out), "Must set app flag to dentoe which app you want to work with")
}

// TestLegacyExec run `drud legacy exec pwd` with proper args
func TestLegacyExec(t *testing.T) {
	args := []string{"legacy", "exec", "-a", LegacyTestApp, "-e", LegacyTestEnv, "pwd"}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(t, err)
	assert.Contains(t, string(out), "/var/www/html/docroot")
}
