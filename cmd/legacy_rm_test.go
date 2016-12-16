package cmd

import (
	"fmt"
	"testing"

	"github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

// TestLegacyRm runs `drud legacy rm` on the test apps
func TestLegacyRm(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	args := []string{"dev", "rm", LegacyTestApp, LegacyTestEnv}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(t, err)
	format := fmt.Sprintf
	assert.Contains(t, string(out), format("Stopping legacy-%s-%s-web ... done", LegacyTestApp, LegacyTestEnv))
	assert.Contains(t, string(out), format("Stopping legacy-%s-%s-db ... done", LegacyTestApp, LegacyTestEnv))
	assert.Contains(t, string(out), format("Removing legacy-%s-%s-web ... done", LegacyTestApp, LegacyTestEnv))
	assert.Contains(t, string(out), format("Removing legacy-%s-%s-db ... done", LegacyTestApp, LegacyTestEnv))
}
