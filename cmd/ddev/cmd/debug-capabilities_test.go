package cmd

import (
	"encoding/json"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/drud/ddev/pkg/exec"
)

// TestDebugCapabilitiesCmd tests that ddev debug capabilities works
func TestDebugCapabilitiesCmd(t *testing.T) {
	assert := asrt.New(t)

	out, err := exec.RunHostCommand(DdevBin, "debug", "capabilities")
	assert.NoError(err)
	assert.Contains(out, "multiple-dockerfiles")

	out, err = exec.RunHostCommand(DdevBin, "debug", "capabilities", "--json")
	assert.NoError(err)

	jsonCapabilities := make([]string, 20)
	err = json.Unmarshal([]byte(out), &jsonCapabilities)
	require.NoError(t, err)
	//assert.True(out, "multiple-dockerfiles")
}
