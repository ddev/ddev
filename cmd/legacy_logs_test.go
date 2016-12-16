package cmd

import (
	"testing"

	"github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

func TestLegacyLogsBadArgs(t *testing.T) {
	err := setActiveApp("", "")
	assert := assert.New(t)
	args := []string{"dev", "logs"}
	out, err := utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "app_name and deploy_name are expected as arguments")
}

// TestLegacyLogs tests that the legacy logs functionality is working.
func TestLegacyLogs(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	assert := assert.New(t)
	args := []string{"dev", "logs", LegacyTestApp, LegacyTestEnv}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Server started")
	assert.Contains(string(out), "GET")
}
