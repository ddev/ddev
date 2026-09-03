package ddevapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/stretchr/testify/require"
)

// TestOSGeneratedFilesSkippedInCustomConfig verifies that files in osGeneratedFiles
// are not reported as custom config, while other files (including hidden dotfiles that
// are legitimate user customizations) are still reported.
func TestOSGeneratedFilesSkippedInCustomConfig(t *testing.T) {
	tmpDir := t.TempDir()

	writeFile := func(name, content string) string {
		path := filepath.Join(tmpDir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
		return path
	}

	dsStore := writeFile(".DS_Store", "finder metadata")
	thumbsDB := writeFile("Thumbs.db", "windows thumbnails")
	desktopIni := writeFile("desktop.ini", "[.ShellClassInfo]")
	bashrc := writeFile(".bashrc", "alias ll='ls -la'")
	gitconfig := writeFile(".gitconfig", "[user]\n\tname = Test")
	normalScript := writeFile("myscript.sh", "#!/bin/bash")

	files := []string{dsStore, thumbsDB, desktopIni, bashrc, gitconfig, normalScript}
	customFiles := filterCustomConfigFiles(files, nil, map[string]string{}, false)

	customPaths := make([]string, len(customFiles))
	for i, f := range customFiles {
		customPaths[i] = f.path
	}

	// OS-generated files must not appear.
	require.NotContains(t, customPaths, dsStore)
	require.NotContains(t, customPaths, thumbsDB)
	require.NotContains(t, customPaths, desktopIni)

	// Legitimate hidden dotfiles (e.g., homeadditions) and normal files must appear.
	require.Contains(t, customPaths, bashrc)
	require.Contains(t, customPaths, gitconfig)
	require.Contains(t, customPaths, normalScript)
}

// TestCheckCustomConfigDBImageReleaseTag verifies that a pinned dbimage is not
// flagged stale just because its com.ddev.image-tag label doesn't match the raw
// content-hash BaseDBTag constant. On a release build, BaseDBTagBranch is a
// vX.Y.Z tag, and docker.GetDBImage() (and the official image DDEV publishes)
// both resolve to that readable tag via docker.ResolveImageTag - so a pinned
// image built from that same official image must be compared against the
// resolved tag too, not the raw hash. See ddev/ddev#8705's
// TestUtilityCheckCustomConfigCmd/"dbimage built for a different DDEV version"
// for the companion case where the label genuinely doesn't match.
func TestCheckCustomConfigDBImageReleaseTag(t *testing.T) {
	if dockerutil.IsPodman() {
		t.Skip("Skipping: podman qualifies locally built image names with localhost/")
	}

	origTag, origBranch := versionconstants.BaseDBTag, versionconstants.BaseDBTagBranch
	versionconstants.BaseDBTag = "abc123"
	versionconstants.BaseDBTagBranch = "v9.9.9"
	t.Cleanup(func() {
		versionconstants.BaseDBTag, versionconstants.BaseDBTagBranch = origTag, origBranch
	})

	currentImage := "ddev-test-current-dbimage:latest"
	buildDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(buildDir, "Dockerfile"),
		[]byte("FROM busybox\nLABEL com.ddev.image-tag=v9.9.9\n"), 0644))
	_, err := exec.RunHostCommand("docker", "build", "-t", currentImage, buildDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = exec.RunHostCommand("docker", "rmi", "-f", currentImage)
	})

	app := newBaseDBSeedTestApp(t, nodeps.MariaDB, nodeps.MariaDBDefaultVersion)
	app.DBImage = currentImage

	message, _ := app.CheckCustomConfig(true)
	require.Contains(t, message, "dbimage: "+currentImage)
	require.NotContains(t, message, "but this DDEV expects",
		"a dbimage labeled with the resolved release tag must not be flagged stale against the raw content-hash constant")
}
