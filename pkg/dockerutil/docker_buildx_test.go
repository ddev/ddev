package dockerutil_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/stretchr/testify/require"
)

// TestDockerBuildxDownload verifies that we can download a particular docker-buildx version
func TestDockerBuildxDownload(t *testing.T) {
	_, err := dockerutil.DownloadDockerBuildxIfNeeded()
	require.NoError(t, err)

	tmpXdgConfigHomeDir := testcommon.CopyGlobalDdevDir(t)

	t.Cleanup(func() {
		testcommon.ResetGlobalDdevDir(t, tmpXdgConfigHomeDir)
	})

	// Remove previous binary
	previousDockerBuildx, _ := globalconfig.GetDockerBuildxDestination()
	_ = os.RemoveAll(previousDockerBuildx)

	// Nothing should be downloaded for "system"
	globalconfig.DdevGlobalConfig.DockerBuildxVersion = "system"
	downloaded, err := dockerutil.DownloadDockerBuildxIfNeeded()
	require.NoError(t, err)
	require.False(t, downloaded)

	// Nothing should be downloaded for ""
	globalconfig.DdevGlobalConfig.DockerBuildxVersion = ""
	downloaded, err = dockerutil.DownloadDockerBuildxIfNeeded()
	require.NoError(t, err)
	require.False(t, downloaded)

	// Set a specific version to trigger a download
	globalconfig.DdevGlobalConfig.DockerBuildxVersion = versionconstants.DockerBuildxRecommendedVersion

	downloaded, err = dockerutil.DownloadDockerBuildxIfNeeded()
	require.NoError(t, err)
	require.True(t, downloaded)
	v, err := dockerutil.GetDockerBuildxVersion()
	require.NoError(t, err)
	require.Equal(t, globalconfig.GetRequiredDockerBuildxVersion(), v)
	require.Equal(t, v, versionconstants.DockerBuildxRecommendedVersion)

	// Make sure it doesn't download a second time
	downloaded, err = dockerutil.DownloadDockerBuildxIfNeeded()
	require.NoError(t, err)
	require.False(t, downloaded)
}

// TestGetBuildxVersion tests that the buildx version can be retrieved.
func TestGetBuildxVersion(t *testing.T) {
	v, err := dockerutil.GetDockerBuildxVersion()
	require.NoError(t, err)
	require.NotEmpty(t, v, "expected non-empty buildx version")
	// Version should not have a v prefix (we strip it)
	require.NotEqual(t, "v", string(v[0]), "expected version without 'v' prefix, got %q", v)
}

// TestGetBuildxLocation tests that the buildx plugin path can be retrieved.
func TestGetBuildxLocation(t *testing.T) {
	pluginPath, err := dockerutil.GetDockerBuildxLocation()
	require.NoError(t, err)
	require.NotEmpty(t, pluginPath, "expected non-empty buildx plugin path")
}

// TestCheckDockerBuildx tests detection of docker-buildx.
func TestCheckDockerBuildx(t *testing.T) {
	ensureDdevBin()
	buildxErr := dockerutil.CheckDockerBuildxVersion()

	if buildxErr != nil {
		out, err := exec.RunHostCommand(DdevBin, "config", "global")
		require.NoError(t, err)
		ddevVersion, err := exec.RunHostCommand(DdevBin, "version")
		require.NoError(t, err)
		require.NoError(t, buildxErr, "DockerBuildxVersion=%s global config=%s ddevVersion=%s", globalconfig.DdevGlobalConfig.DockerBuildxVersion, out, ddevVersion)
	}
}

// TestBuildxMinVersionInSync ensures that versionconstants.DockerBuildxMinVersion stays in sync
// with the buildxMinVersion constant in the vendored Compose library.
// Run this test whenever docker/compose or versionconstants is updated.
func TestBuildxMinVersionInSync(t *testing.T) {
	data, err := os.ReadFile("../../vendor/github.com/docker/compose/v5/pkg/compose/api_versions.go")
	require.NoError(t, err, "could not read vendored compose api_versions.go; run 'go mod vendor'")

	re := regexp.MustCompile(`buildxMinVersion\s*=\s*"([^"]+)"`)
	matches := re.FindSubmatch(data)
	require.Len(t, matches, 2, "buildxMinVersion constant not found in vendor/github.com/docker/compose/v5/pkg/compose/api_versions.go")

	composeMin := string(matches[1])
	require.Equal(t, composeMin, versionconstants.DockerBuildxMinVersion,
		"versionconstants.DockerBuildxMinVersion (%q) is out of sync with compose's buildxMinVersion (%q); update pkg/versionconstants/versionconstants.go",
		versionconstants.DockerBuildxMinVersion, composeMin)
}

// TestBuildxRecommendedVersionInSync ensures that versionconstants.DockerBuildxRecommendedVersion
// stays in sync with the github.com/docker/buildx version in go.mod.
// Run this test whenever docker/buildx is bumped in go.mod or versionconstants is updated.
func TestBuildxRecommendedVersionInSync(t *testing.T) {
	data, err := os.ReadFile("../../go.mod")
	require.NoError(t, err, "could not read go.mod")

	re := regexp.MustCompile(`(?m)^\s*github\.com/docker/buildx\s+v([^\s]+)`)
	matches := re.FindSubmatch(data)
	require.Len(t, matches, 2, "github.com/docker/buildx require directive not found in go.mod")

	goModVersion := string(matches[1])
	require.Equal(t, goModVersion, versionconstants.DockerBuildxRecommendedVersion,
		"versionconstants.DockerBuildxRecommendedVersion (%q) is out of sync with go.mod's github.com/docker/buildx (%q); update pkg/versionconstants/versionconstants.go",
		versionconstants.DockerBuildxRecommendedVersion, goModVersion)
}
