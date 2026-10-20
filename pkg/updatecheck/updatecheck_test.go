package updatecheck

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testOrg = "ddev"
const testRepo = "ddev"

// TestGetContainerHealth tests the function for processing container readiness.
func TestUpdateNeeded(t *testing.T) {
	assert := asrt.New(t)
	tmpdir := testcommon.CreateTmpDir("TestUpdateNeeded")
	updateFile := filepath.Join(tmpdir, ".update")

	// Ensure updates are required when the update file doesn't exist yet.
	updateRequired, err := IsUpdateNeeded(updateFile, 60*time.Second)
	assert.True(updateRequired, "Update is required when the update file does not exist")
	assert.NoError(err)

	// Ensure updates are not required when the update duration is impossibly far in the future.
	updateRequired, err = IsUpdateNeeded(updateFile, 9999999*time.Second)
	assert.False(updateRequired, "Update is not required when the update interval has not been met")
	assert.NoError(err)

	time.Sleep(2 * time.Second)

	// Ensure updates are required for a duration lower than the sleep.
	updateRequired, err = IsUpdateNeeded(updateFile, 1*time.Second)
	assert.True(updateRequired, "Update is required after the update interval has passed")
	assert.NoError(err)

	testcommon.CleanupDir(tmpdir)
}

// TestIsReleaseVersion tests isReleaseVersion to ensure it correctly picks up on release builds vs dev builds
func TestIsReleaseVersion(t *testing.T) {
	assert := asrt.New(t)
	var versionTests = []struct {
		in  string
		out bool
	}{
		{"0.1.0", true},
		{"v0.1.0", true},
		{"v19.99.99", true},
		{"v1.17.0-alpha1", true},
		{"v1.18.0-alpha4-43-gb5ff9108-dirty", false},
		{"v1.18.0-alpha4-43-gb5ff9108", false},
		{"19.99.99-8us8dfgh7-dirty", false},
		{"v0.3-7-g3ca5586-dirty", false},
	}

	for _, tt := range versionTests {
		result := isReleaseVersion(tt.in)
		assert.Equal(result, tt.out, fmt.Sprintf("Got output which was not expected from isReleaseVersion. Input: %s Output: %t Expected: %t", tt.in, result, tt.out))
	}
}

// TestIsNewerVersion covers the version comparison without calling GitHub.
func TestIsNewerVersion(t *testing.T) {
	var versionTests = []struct {
		current string
		latest  string
		out     bool
	}{
		{"0.0.0", "v1.25.4", true},
		{"v0.1.1", "v1.25.4", true},
		{"v1.25.3", "v1.25.4", true},
		{"v1.25.4", "v1.25.4", false},
		{"v1.26.0-alpha1", "v1.25.4", false},
		{"v1.25.4-alpha1", "v1.25.4", true},
		{"v999999.999999.999999", "v1.25.4", false},
	}

	for _, tt := range versionTests {
		newer, err := isNewerVersion(tt.current, tt.latest)
		require.NoError(t, err)
		require.Equal(t, tt.out, newer, "current=%s latest=%s", tt.current, tt.latest)
	}

	_, err := isNewerVersion("v1.25.4", "not-a-version")
	require.Error(t, err)
}

// TestAvailableUpdates checks AvailableUpdates against the live ddev/ddev releases.
func TestAvailableUpdates(t *testing.T) {
	testcommon.SkipIfGotestShort(t)

	updateNeeded, newVersion, updateURL, err := AvailableUpdates(testOrg, testRepo, "0.0.0")
	if err != nil {
		t.Skipf("AvailableUpdates() failed, err=%v", err)
	}
	require.True(t, updateNeeded, "newVersion=%s", newVersion)
	require.Contains(t, updateURL, "https://")

	updateNeeded, _, _, err = AvailableUpdates(testOrg, testRepo, "v999999.999999.999999")
	if err != nil {
		t.Skipf("AvailableUpdates() failed, err=%v", err)
	}
	require.False(t, updateNeeded)

	updateNeeded, _, _, err = AvailableUpdates(testOrg, testRepo, "v1.18.0-alpha4-43-gb5ff9108-dirty")
	require.NoError(t, err)
	require.False(t, updateNeeded, "dev builds never report an update")
}
