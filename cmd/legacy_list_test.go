package cmd

import (
	"testing"

	"github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

func TestLegacyList(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	args := []string{"dev", "list"}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(t, err)
	assert.Contains(t, string(out), "found")
	assert.Contains(t, string(out), LegacyTestApp)
	assert.Contains(t, string(out), LegacyTestEnv)
	assert.Contains(t, string(out), "running")
}
