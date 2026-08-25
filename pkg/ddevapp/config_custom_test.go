package ddevapp

import (
	"os"
	"path/filepath"
	"testing"

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

// TestIsUnreleasedDdevVersion verifies detection of DDEV versions built from
// an untagged commit, as opposed to a clean release tag.
func TestIsUnreleasedDdevVersion(t *testing.T) {
	unreleased := []string{
		"v1.25.3-111-geadf098e9", // git describe, commits past a tag
		"v1.25.3-dirty",          // exact tag, but uncommitted changes
		"v1.25.3-111-geadf098e9-dirty",
		"geadf098", // debug.BuildInfo short-hash fallback
		"geadf098-dirty",
	}
	for _, v := range unreleased {
		require.True(t, IsUnreleasedDdevVersion(v), "expected %q to be detected as unreleased", v)
	}

	released := []string{
		"v1.25.3",
		"v1.25.3-beta1",
		"v0.0.0-overridden-by-make",
	}
	for _, v := range released {
		require.False(t, IsUnreleasedDdevVersion(v), "expected %q to be detected as a release", v)
	}
}
