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

	backupConfig := globalconfig.DdevGlobalConfig
	// Start with no config file
	configFile := globalconfig.GetGlobalConfigPath()
	if fileutil.FileExists(configFile) {
		err := os.Remove(configFile)
		require.NoError(t, err)
	}
	// We need to make sure that the (corrupted, bogus) global config file is removed
	// and then read (empty)
	// nolint: errcheck
	defer func() {
		globalconfig.DdevGlobalConfig = backupConfig
		globalconfig.DdevGlobalConfig.OmitContainersGlobal = nil

		err := os.Remove(configFile)
		if err != nil {
			t.Logf("Unable to remove %v: %v", configFile, err)
		}
		err = globalconfig.ReadGlobalConfig()
		if err != nil {
			t.Logf("Unable to ReadGlobalConfig: %v", err)
		}
	}()

	// Look at initial config
	args := []string{"config", "global"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Global configuration:\ninstrumentation-opt-in=false\nomit-containers=[]\nrouter-bind-all-interfaces=false")

	// Update a config
	args = []string{"config", "global", "--instrumentation-opt-in=false", "--omit-containers=dba,ddev-ssh-agent", "--router-bind-all-interfaces=true"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Global configuration:\ninstrumentation-opt-in=false\nomit-containers=[dba,ddev-ssh-agent]\nrouter-bind-all-interfaces=true")

	err = globalconfig.ReadGlobalConfig()
	assert.NoError(err)
	assert.False(globalconfig.DdevGlobalConfig.InstrumentationOptIn)
	assert.Contains(globalconfig.DdevGlobalConfig.OmitContainersGlobal, "ddev-ssh-agent")
	assert.Contains(globalconfig.DdevGlobalConfig.OmitContainersGlobal, "dba")
	assert.Len(globalconfig.DdevGlobalConfig.OmitContainersGlobal, 2)

	// Even though the global config is going to be deleted, make sure it's sane before leaving
	args = []string{"config", "global", "--omit-containers", ""}
	globalconfig.DdevGlobalConfig.OmitContainersGlobal = nil
	_, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)
}
