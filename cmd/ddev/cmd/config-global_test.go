package cmd

import (
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

// Run with various flags
// Try to create errors
// Validate that what it spits out is what's there

func TestCmdGlobalConfig(t *testing.T) {
	assert := asrt.New(t)

	// Start with no config file
	configFile := globalconfig.GetGlobalConfigPath()
	if fileutil.FileExists(configFile) {
		err := os.Remove(configFile)
		require.NoError(t, err)
	}
	defer os.Remove(configFile)

	// Look at initial config
	args := []string{"config", "global"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Global configuration:\ninstrumentation-opt-in=false\nomit-containers=[]")

	// Update a config
	args = []string{"config", "global", "--instrumentation-opt-in=false", "--omit-containers=dba,ddev-ssh-agent"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Global configuration:\ninstrumentation-opt-in=false\nomit-containers=[dba,ddev-ssh-agent]")

	found, err := fileutil.FgrepStringInFile(configFile, "\ninstrumentation_opt_in: false")
	assert.NoError(err)
	assert.True(found)
	found, err = fileutil.FgrepStringInFile(configFile, "\n- dba")
	assert.NoError(err)
	assert.True(found)
	found, err = fileutil.FgrepStringInFile(configFile, "\n- ddev-ssh-agent")
	assert.NoError(err)
	assert.True(found)
}
