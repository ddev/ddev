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

	for _, site := range LegacyTestSites {
		args := []string{"dev", "rm", site[0], site[1]}
		out, err := utils.RunCommand(DrudBin, args)
		assert.NoError(t, err)
		format := fmt.Sprintf
		assert.Contains(t, string(out), format("Stopping legacy-%s-%s-web ... done", site[0], site[1]))
		assert.Contains(t, string(out), format("Stopping legacy-%s-%s-db ... done", site[0], site[1]))
		assert.Contains(t, string(out), format("Removing legacy-%s-%s-web ... done", site[0], site[1]))
		assert.Contains(t, string(out), format("Removing legacy-%s-%s-db ... done", site[0], site[1]))
	}
}
