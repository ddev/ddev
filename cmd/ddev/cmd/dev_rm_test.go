package cmd

import (
	"fmt"
	"testing"

	"github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

// TestDevRm runs `drud legacy rm` on the test apps
func TestDevRm(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}

	for _, site := range DevTestSites {
		args := []string{"rm", site[0], site[1]}
		out, err := utils.RunCommand(DdevBin, args)
		assert.NoError(t, err)
		format := fmt.Sprintf
		assert.Contains(t, string(out), format("Stopping legacy-%s-%s-web ... done", site[0], site[1]))
		assert.Contains(t, string(out), format("Stopping legacy-%s-%s-db ... done", site[0], site[1]))
		assert.Contains(t, string(out), format("Removing legacy-%s-%s-web ... done", site[0], site[1]))
		assert.Contains(t, string(out), format("Removing legacy-%s-%s-db ... done", site[0], site[1]))
	}
}
