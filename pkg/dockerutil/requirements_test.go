package dockerutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckCompose tests detection of docker-compose.
func TestCheckCompose(t *testing.T) {
	assert := asrt.New(t)

	globalconfig.DockerComposeVersion = ""
	composeErr := dockerutil.CheckDockerCompose()
	if composeErr != nil {
		out, err := exec.RunHostCommand(DdevBin, "config", "global")
		require.NoError(t, err)
		ddevVersion, err := exec.RunHostCommand(DdevBin, "version")
		require.NoError(t, err)
		assert.NoError(composeErr, "RequiredDockerComposeVersion=%s global config=%s ddevVersion=%s", globalconfig.DdevGlobalConfig.RequiredDockerComposeVersion, out, ddevVersion)
	}
}

// TestCheckDockerAuth tests the CheckDockerAuth function
func TestCheckDockerAuth(t *testing.T) {
	tmpHome := t.TempDir()

	// Change the homedir temporarily
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	dockerConfigDir := filepath.Join(tmpHome, ".docker")
	err := os.MkdirAll(dockerConfigDir, 0755)
	require.NoError(t, err)

	// Test 1: No config file - should pass
	err = dockerutil.CheckDockerAuth()
	require.NoError(t, err)

	// Test 2: Config with non-existent credsStore - should fail
	badConfig := `{
  "credsStore": "nonexistent-helper"
}`
	configPath := filepath.Join(dockerConfigDir, "config.json")
	err = fileutil.TemplateStringToFile(badConfig, nil, configPath)
	require.NoError(t, err)

	err = dockerutil.CheckDockerAuth()
	require.Error(t, err)
	require.Contains(t, err.Error(), "docker-credential-nonexistent-helper")
	require.Contains(t, err.Error(), "not found in PATH")
	require.Contains(t, err.Error(), "This will cause 'ddev start' to fail")

	// Test 3: Config without credsStore - should pass
	goodConfig := `{
  "auths": {}
}`
	err = fileutil.TemplateStringToFile(goodConfig, nil, configPath)
	require.NoError(t, err)

	err = dockerutil.CheckDockerAuth()
	require.NoError(t, err)

	// Test 4: Malformed JSON - should fail gracefully
	malformedConfig := `{
  "credsStore": "osxkeychain",
  invalid json
}`
	err = fileutil.TemplateStringToFile(malformedConfig, nil, configPath)
	require.NoError(t, err)

	err = dockerutil.CheckDockerAuth()
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to parse Docker config")
}
