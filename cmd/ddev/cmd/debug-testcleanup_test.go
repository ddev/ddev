package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDebugTestCleanupCmd ensures the debug testcleanup only removes diagnostic projects prefixed with 'tryddevproject-'
func TestDebugTestCleanupCmd(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	debugAppName := "tryddevproject-" + t.Name()
	nonDebugAppName := TestSites[0].Name

	testDir := testcommon.CreateTmpDir(debugAppName)
	err := os.Chdir(testDir)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(testDir)
		assert.NoError(err)
	})

	out, err := exec.RunHostCommand(DdevBin, "config", "--project-name", debugAppName)
	require.NoError(t, err, "Failed to run ddev config: %s", out)

	out, err = exec.RunHostCommand(DdevBin, "debug", "testcleanup")
	require.NoError(t, err, "Failed to run ddev debug testcleanup: %s", out)

	assert.NotContains(out, fmt.Sprintf("Project %s was deleted", nonDebugAppName))
	assert.Contains(out, fmt.Sprintf("Project %s was deleted", debugAppName))
	assert.Contains(out, "Finished cleaning ddev diagnostic projects")

	out, err = exec.RunCommand(DdevBin, []string{"describe", debugAppName})
	assert.Error(err, "Expected an error when describing a deleted project")
	assert.Contains(out, "could not find requested project")
}
