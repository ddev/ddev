package updatecheck

import (
	"fmt"
	"path/filepath"
	"testing"

	"time"

	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/version"
	"github.com/stretchr/testify/assert"
	"os"
)

const testOrg = "drud"
const testRepo = "ddev"

// TestGetContainerHealth tests the function for processing container readiness.
func TestUpdateNeeded(t *testing.T) {
	assert := assert.New(t)
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
	assert := assert.New(t)
	var versionTests = []struct {
		in  string
		out bool
	}{
		{"0.1.0", true},
		{"v0.1.0", true},
		{"v19.99.99", true},
		{"19.99.99-8us8dfgh7-dirty", false},
		{"v0.3-7-g3ca5586-dirty", false},
	}

	for _, tt := range versionTests {
		result := isReleaseVersion(tt.in)
		assert.Equal(result, tt.out, fmt.Sprintf("Got output which was not expected from isReleaseVersion. Input: %s Output: %t Expected: %t", tt.in, result, tt.out))
	}
}

// TestAvailableUpdates tests isReleaseVersion to ensure it correctly picks up on release builds vs dev builds
func TestAvailableUpdates(t *testing.T) {
	assert := assert.New(t)
	if os.Getenv("GOTEST_SHORT") != "" {
		t.Skip("Skipping TestAvailableUpdates because GOTEST_SHORT env var is set")
	}
	var versionTests = []struct {
		in  string
		out bool
	}{
		{"0.0.0", true},
		{"v0.1.1", true},
		{version.DdevVersion, false},
		{"v999999.999999.999999", false},
	}

	for _, tt := range versionTests {
		updateNeeded, updateURL, err := AvailableUpdates(testOrg, testRepo, tt.in)
		if err != nil {
			t.Fatalf("AvailableUpdates() failed, err=%v", err)
		}
		assert.Equal(updateNeeded, tt.out, fmt.Sprintf("Unexpected output from AvailableUpdates. Input: %s Output: %t Expected: %t Org: %s Repo: %s", tt.in, updateNeeded, tt.out, testOrg, testRepo))

		if updateNeeded {
			assert.Contains(updateURL, "https://")
		}
	}
}
