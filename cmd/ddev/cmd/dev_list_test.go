package cmd

import (
	"testing"

	"github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

func TestDevList(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	args := []string{"list"}
	out, err := utils.RunCommand(DdevBin, args)
	assert.NoError(t, err)
	assert.Contains(t, string(out), "found")
	assert.Contains(t, string(out), DevTestApp)
	assert.Contains(t, string(out), DevTestEnv)
	assert.Contains(t, string(out), "running")
}
