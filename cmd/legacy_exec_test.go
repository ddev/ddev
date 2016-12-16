package cmd

import (
	"testing"

	"github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

// TestLegacyExecBadArgs run `drud legacy exec` without the proper args
func TestLegacyExecBadArgs(t *testing.T) {
	assert := assert.New(t)
	args := []string{"dev", "exec", LegacyTestApp, LegacyTestEnv}
	out, err := utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Invalid arguments detected.")

	args = []string{"dev", "exec", "pwd"}
	out, err = utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "app_name and deploy_name are expected as arguments")

	// Try with an invalid number of args
	args = []string{"dev", "exec", LegacyTestApp, "pwd"}
	out, err = utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Invalid arguments detected")
}

// TestLegacyExec run `drud legacy exec pwd` with proper args
func TestLegacyExec(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}

	// Run an exec by passing in TestApp + TestEnv
	assert := assert.New(t)

	args := []string{"dev", "exec", LegacyTestApp, LegacyTestEnv, "pwd"}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "/var/www/html/docroot")

	// Try again with active app set.
	err = setActiveApp(LegacyTestApp, LegacyTestEnv)
	assert.NoError(err)
	args = []string{"dev", "exec", LegacyTestApp, LegacyTestEnv, "pwd"}
	out, err = utils.RunCommand(DrudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "/var/www/html/docroot")
}
