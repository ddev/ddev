package cmd

import (
	"encoding/json"
	"github.com/ddev/ddev/pkg/nodeps"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
)

// TestDebugCapabilitiesCmd tests that ddev debug capabilities works
func TestDebugCapabilitiesCmd(t *testing.T) {
	assert := asrt.New(t)

	out, err := exec.RunHostCommand(DdevBin, "debug", "capabilities")
	assert.NoError(err)
	assert.Contains(out, "multiple-dockerfiles")

	out, err = exec.RunHostCommandSeparateStreams(DdevBin, "debug", "-j", "capabilities")
	assert.NoError(err)

	jsonCapabilities := make(map[string]interface{})
	err = json.Unmarshal([]byte(out), &jsonCapabilities)
	require.NoError(t, err, "failed to unmarshall json capabilities '%v', out")
	caps, ok := jsonCapabilities["raw"]
	require.True(t, ok, "raw section wasn't found in jsonCapabilities: %v", out)
	sArr := []string{}
	for _, x := range caps.([]interface{}) {
		sArr = append(sArr, x.(string))
	}
	require.True(t, nodeps.ArrayContainsString(sArr, "multiple-dockerfiles"))
}
