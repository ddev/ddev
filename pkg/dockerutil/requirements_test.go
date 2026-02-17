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

// TestGetCLIPlugins tests that Docker CLI plugins can be discovered.
func TestGetCLIPlugins(t *testing.T) {
	plugins, err := dockerutil.GetCLIPlugins()
	require.NoError(t, err)
	require.NotEmpty(t, plugins, "expected at least one CLI plugin to be installed")

	// buildx should be among the discovered plugins
	found := false
	for _, p := range plugins {
		if p.Name == "buildx" {
			found = true
			break
		}
	}
	require.True(t, found, "expected 'buildx' to be among discovered CLI plugins")
}

// TestGetBuildxVersion tests that the buildx version can be retrieved.
func TestGetBuildxVersion(t *testing.T) {
	v, err := dockerutil.GetBuildxVersion()
	require.NoError(t, err)
	require.NotEmpty(t, v, "expected non-empty buildx version")
	// Version should not have a v prefix (we strip it)
	require.NotEqual(t, "v", string(v[0]), "expected version without 'v' prefix, got %q", v)
}

// TestGetBuildxLocation tests that the buildx plugin path can be retrieved.
func TestGetBuildxLocation(t *testing.T) {
	pluginPath, err := dockerutil.GetBuildxLocation()
	require.NoError(t, err)
	require.NotEmpty(t, pluginPath, "expected non-empty buildx plugin path")
}

// TestCheckBuildx tests that CheckDockerBuildxVersion passes on a host with buildx installed.
func TestCheckBuildx(t *testing.T) {
	err := dockerutil.CheckDockerBuildxVersion(dockerutil.DockerRequirements)
	require.NoError(t, err)
}

// TestCheckDockerAuth tests the CheckDockerAuth function
func TestCheckDockerAuth(t *testing.T) {
	tmpHome := t.TempDir()
	_, dockerHost, _ := dockerutil.GetDockerContextNameAndHost()
	// Change the homedir temporarily
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	// Set DOCKER_HOST to the same value as before, otherwise wrong Docker context may be used
	// It's not needed for this exact test, but helps ensure consistency when $HOME is changed
	t.Setenv("DOCKER_HOST", dockerHost)

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
