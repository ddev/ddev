package cmd

import (
	"testing"

	"github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

func TestDevLogsBadArgs(t *testing.T) {
	err := setActiveApp("", "")
	assert := assert.New(t)
	args := []string{"logs"}
	out, err := utils.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "app_name and deploy_name are expected as arguments")
}

// TestDevLogs tests that the Dev logs functionality is working.
func TestDevLogs(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	assert := assert.New(t)
	args := []string{"logs", DevTestApp, DevTestEnv}
	out, err := utils.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Server started")
	assert.Contains(string(out), "GET")
}
