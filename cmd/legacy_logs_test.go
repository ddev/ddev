package cmd

import (
	"testing"

	drudutils "github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

// TestLegacyLogs tests that the legacy logs functionality is working.
func TestLegacyLogs(t *testing.T) {
	assert := assert.New(t)
	args := []string{"legacy", "logs", "-a", legacyTestApp, "e", legacyTestEnv}
	out, err := drudutils.RunCommand(drudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Server started")
	assert.Contains(string(out), "GET")
}
