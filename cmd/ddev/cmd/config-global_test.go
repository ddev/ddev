package cmd

import (
	"fmt"
	"os"
	"testing"

	configTypes "github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/globalconfig/types"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	t.Cleanup(func() {
		// Even though the global config is going to be deleted, make sure it's sane before leaving
		args := []string{"config", "global", "--omit-containers", "", "--disable-http2=false", "--performance-mode-reset", "--simple-formatting=false", "--table-style=default", `--required-docker-compose-version=""`, `--use-docker-compose-from-path=false`, `--xdebug-ide-location`, "", `--router=traefik`, `--traefik-monitor-port=10999`}
		globalconfig.DdevGlobalConfig.OmitContainersGlobal = nil
		out, err := exec.RunHostCommand(DdevBin, args...)
		assert.NoError(err, "error running ddev config global; output=%s", out)
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
	})

	// Look at initial config
	args := []string{"config", "global"}
	out, err := exec.RunCommand(DdevBin, args)
	require.NoError(t, err)
	assert.NoError(err, "error running ddev config global; output=%s", out)
	assert.Contains(out, "instrumentation-opt-in=false\nomit-containers=[]")
	assert.Contains(out, `web-environment=[]`)
	assert.Contains(out, fmt.Sprintf("performance-mode=%s", configTypes.GetPerformanceModeDefault()))
	assert.Contains(out, "router-bind-all-interfaces=false")
	assert.Contains(out, "internet-detection-timeout-ms=3000")
	assert.Contains(out, "disable-http2=false")
	assert.Contains(out, "use-letsencrypt=false\nletsencrypt-email=\n")
	assert.Contains(out, "table-style=default\nsimple-formatting=false")
	assert.Contains(out, "auto-restart-containers=false\nuse-hardened-images=false\n")
	assert.Contains(out, "fail-on-hook-fail=false")
	assert.Contains(out, fmt.Sprintf("required-docker-compose-version=%s\nuse-docker-compose-from-path=false", globalconfig.DdevGlobalConfig.RequiredDockerComposeVersion))
	assert.Contains(out, "project-tld=\nxdebug-ide-location=")
	assert.Contains(out, "router=traefik")
	assert.Contains(out, "wsl2-no-windows-hosts-mgt=false")
	assert.Contains(out, "router-http-port=80\nrouter-https-port=443")
	assert.Contains(out, "traefik-monitor-port=10999")

	// Update a config
	// Don't include no-bind-mounts because global testing
	// will turn it on and break this
	args = []string{"config", "global", "--project-tld=ddev.test", "--instrumentation-opt-in=false", "--omit-containers=ddev-ssh-agent", "--performance-mode=mutagen", "--router-bind-all-interfaces=true", "--internet-detection-timeout-ms=850", "--table-style=bright", "--simple-formatting=true", "--auto-restart-containers=true", "--use-hardened-images=true", "--fail-on-hook-fail=true", `--web-environment="SOMEENV=some+val"`, `--xdebug-ide-location=container`, `--router=nginx-proxy`, `--router-http-port=8081`, `--router-https-port=8882`, `--traefik-monitor-port=11999`}
	out, err = exec.RunCommand(DdevBin, args)
	require.NoError(t, err)
	assert.NoError(err, "error running ddev config global; output=%s", out)
	assert.Contains(out, "instrumentation-opt-in=false")
	assert.Contains(out, "omit-containers=[ddev-ssh-agent]")
	assert.Contains(out, `web-environment=["SOMEENV=some+val"]`)
	assert.Contains(out, fmt.Sprintf("performance-mode=%s", configTypes.PerformanceModeMutagen))
	assert.Contains(out, "router-bind-all-interfaces=true")
	assert.Contains(out, "internet-detection-timeout-ms=850")
	assert.Contains(out, "disable-http2=false")
	assert.Contains(out, "use-letsencrypt=false\nletsencrypt-email=\n")
	assert.Contains(out, "table-style=bright\nsimple-formatting=true")
	assert.Contains(out, "auto-restart-containers=true\nuse-hardened-images=true\n")
	assert.Contains(out, "fail-on-hook-fail=true")
	assert.Contains(out, fmt.Sprintf("required-docker-compose-version=%s\nuse-docker-compose-from-path=false", globalconfig.DdevGlobalConfig.RequiredDockerComposeVersion))
	assert.Contains(out, "project-tld=")
	assert.Contains(out, "router=nginx-proxy")
	assert.Contains(out, "wsl2-no-windows-hosts-mgt=false")

	assert.Contains(string(out), "xdebug-ide-location=container")
	assert.Contains(out, "wsl2-no-windows-hosts-mgt=false")
	assert.Contains(out, "router-http-port=8081\nrouter-https-port=8882")
	assert.Contains(out, "router=nginx-proxy")
	assert.Contains(out, "traefik-monitor-port=11999")

	globalconfig.EnsureGlobalConfig()
	assert.False(globalconfig.DdevGlobalConfig.InstrumentationOptIn)
	assert.Contains(globalconfig.DdevGlobalConfig.OmitContainersGlobal, "ddev-ssh-agent")
	assert.True(globalconfig.DdevGlobalConfig.IsMutagenEnabled())
	assert.False(globalconfig.DdevGlobalConfig.IsNFSMountEnabled())
	assert.Len(globalconfig.DdevGlobalConfig.OmitContainersGlobal, 1)
	assert.Equal("ddev.test", globalconfig.DdevGlobalConfig.ProjectTldGlobal)
	assert.True(globalconfig.DdevGlobalConfig.UseHardenedImages)
	assert.True(globalconfig.DdevGlobalConfig.FailOnHookFailGlobal)
	assert.True(globalconfig.DdevGlobalConfig.SimpleFormatting)
	assert.Equal("bright", globalconfig.DdevGlobalConfig.TableStyle)
	assert.Equal("container", globalconfig.DdevGlobalConfig.XdebugIDELocation)
	assert.Equal(types.RouterTypeNginxProxy, globalconfig.DdevGlobalConfig.Router)
	assert.Equal("8081", globalconfig.DdevGlobalConfig.RouterHTTPPort)
	assert.Equal("8882", globalconfig.DdevGlobalConfig.RouterHTTPSPort)

	// Test that variables can be appended to the web environment
	args = []string{"config", "global", `--web-environment-add="FOO=bar"`}
	out, err = exec.RunCommand(DdevBin, args)
	require.NoError(t, err)
	assert.Contains(string(out), "web-environment=[\"FOO=bar\",\"SOMEENV=some+val\"]")

	// Test that NFS can be enabled
	args = []string{"config", "global", "--performance-mode=nfs"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)

	assert.Contains(string(out), "performance-mode=nfs")

	err = globalconfig.ReadGlobalConfig()
	assert.NoError(err)

	assert.False(globalconfig.DdevGlobalConfig.IsMutagenEnabled())
	assert.True(globalconfig.DdevGlobalConfig.IsNFSMountEnabled())
}
