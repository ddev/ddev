package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/github"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// versionLineRegex matches the "version: ..." line in an add-on manifest.yaml
var versionLineRegex = regexp.MustCompile(`(?m)^version:.*$`)

// TestCmdAddonUpdate tests `ddev add-on update`
func TestCmdAddonUpdate(t *testing.T) {
	if !github.HasGitHubToken() {
		t.Skip("Skipping because DDEV_GITHUB_TOKEN is not set")
	}
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "redis")
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "example")
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis")
	require.NoError(t, err, "output=%s", out)

	// Install a directory-based addon too; it can't be checked for updates.
	exampleDir := filepath.Join(origDir, "testdata", "TestCmdAddon", "example-repo")
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", exampleDir)
	require.NoError(t, err, "output=%s", out)

	// Fake an outdated version so 'update' has something to do, without
	// depending on a specific historical redis release.
	manifestFile := app.GetConfigPath("addon-metadata/redis/manifest.yaml")
	require.FileExists(t, manifestFile)
	content, err := os.ReadFile(manifestFile)
	require.NoError(t, err)
	fakeContent := versionLineRegex.ReplaceAll(content, []byte("version: v0.0.1-fake-old-for-testing"))
	err = os.WriteFile(manifestFile, fakeContent, 0644)
	require.NoError(t, err)

	// Dry run should report redis as updatable and skip the directory-based addon,
	// without changing anything.
	out, err = exec.RunHostCommand(DdevBin, "add-on", "update", "--dry-run")
	require.NoError(t, err, "output=%s", out)
	assert.Contains(out, "redis")
	assert.Contains(out, "can be updated")
	assert.Contains(out, "Skipping 'example'")

	content, err = os.ReadFile(manifestFile)
	require.NoError(t, err)
	assert.Contains(string(content), "v0.0.1-fake-old-for-testing")

	// A real update should install the latest release and rewrite the manifest.
	// --yes skips the confirmation prompt.
	out, err = exec.RunHostCommand(DdevBin, "add-on", "update", "--yes")
	require.NoError(t, err, "output=%s", out)
	assert.Contains(out, "Updated 1 add-on(s): redis")
	assert.Contains(out, "To downgrade again:")
	assert.Contains(out, "ddev add-on get ddev/ddev-redis --version v0.0.1-fake-old-for-testing")

	content, err = os.ReadFile(manifestFile)
	require.NoError(t, err)
	assert.NotContains(string(content), "v0.0.1-fake-old-for-testing")

	// Running again should find nothing to update.
	out, err = exec.RunHostCommand(DdevBin, "add-on", "update")
	require.NoError(t, err, "output=%s", out)
	assert.Contains(out, "All add-ons are up to date.")
}
