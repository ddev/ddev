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
		// Even though the global config is going to be deleted, make sure it's sane before leaving
		args := []string{"config", "global", "--omit-containers", "", "--nfs-mount-enabled=true", "--disable-http2=false"}
		globalconfig.DdevGlobalConfig.OmitContainersGlobal = nil
		_, err := exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		globalconfig.DdevGlobalConfig = backupConfig
		globalconfig.DdevGlobalConfig.OmitContainersGlobal = nil

		err = os.Remove(configFile)
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
	assert.Contains(string(out), "Global configuration:\ninstrumentation-opt-in=false\nomit-containers=[]\nweb-environment=[]\nnfs-mount-enabled=false\nrouter-bind-all-interfaces=false\ninternet-detection-timeout-ms=750\ndisable-http2=false\nuse-letsencrypt=false\nletsencrypt-email=\nauto-restart-containers=false\nuse-hardened-images=false\nfail-on-hook-fail=false")

	// Update a config
	args = []string{"config", "global", "--instrumentation-opt-in=false", "--omit-containers=dba,ddev-ssh-agent", "--nfs-mount-enabled=true", "--router-bind-all-interfaces=true", "--internet-detection-timeout-ms=850", "--use-letsencrypt", "--letsencrypt-email=nobody@example.com", "--auto-restart-containers=true", "--use-hardened-images=true", "--fail-on-hook-fail=true", `--disable-http2`, `--web-environment="SOMEENV=some+val"`}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Global configuration:\ninstrumentation-opt-in=false\nomit-containers=[dba,ddev-ssh-agent]\nweb-environment=[\"SOMEENV=some+val\"]\nnfs-mount-enabled=true\nrouter-bind-all-interfaces=true\ninternet-detection-timeout-ms=850\ndisable-http2=true\nuse-letsencrypt=true\nletsencrypt-email=nobody@example.com\nauto-restart-containers=true\nuse-hardened-images=true\nfail-on-hook-fail=true")

	err = globalconfig.ReadGlobalConfig()
	assert.NoError(err)
	assert.False(globalconfig.DdevGlobalConfig.InstrumentationOptIn)
	assert.Contains(globalconfig.DdevGlobalConfig.OmitContainersGlobal, "ddev-ssh-agent")
	assert.Contains(globalconfig.DdevGlobalConfig.OmitContainersGlobal, "dba")
	assert.True(globalconfig.DdevGlobalConfig.NFSMountEnabledGlobal)
	assert.Len(globalconfig.DdevGlobalConfig.OmitContainersGlobal, 2)
	assert.Equal("nobody@example.com", globalconfig.DdevGlobalConfig.LetsEncryptEmail)
	assert.True(globalconfig.DdevGlobalConfig.UseLetsEncrypt)
	assert.True(globalconfig.DdevGlobalConfig.UseHardenedImages)
	assert.True(globalconfig.DdevGlobalConfig.FailOnHookFailGlobal)
	assert.True(globalconfig.DdevGlobalConfig.DisableHTTP2)
}
