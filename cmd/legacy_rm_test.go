package cmd

import (
	"fmt"
	"testing"

	drudutils "github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

// TestLegacyRm runs `drud legacy rm` on the test apps
func TestLegacyRm(t *testing.T) {
	args := []string{"legacy", "rm", "-a", legacyTestApp}
	out, err := drudutils.RunCommand(drudBin, args)
	assert.NoError(t, err)
	format := fmt.Sprintf
	assert.Contains(t, string(out), format("Stopping legacy-%s-%s-web ... done", legacyTestApp, legacyTestEnv))
	assert.Contains(t, string(out), format("Stopping legacy-%s-%s-db ... done", legacyTestApp, legacyTestEnv))
	assert.Contains(t, string(out), format("Removing legacy-%s-%s-web ... done", legacyTestApp, legacyTestEnv))
	assert.Contains(t, string(out), format("Removing legacy-%s-%s-db ... done", legacyTestApp, legacyTestEnv))
}
