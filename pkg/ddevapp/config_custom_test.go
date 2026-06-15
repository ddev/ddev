package ddevapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterCustomConfigFilesIgnoresHiddenFiles(t *testing.T) {
	tmpDir := t.TempDir()

	hiddenFile := filepath.Join(tmpDir, ".DS_Store")
	err := os.WriteFile(hiddenFile, []byte("finder metadata"), 0644)
	require.NoError(t, err)

	envFile := filepath.Join(tmpDir, ".env")
	err = os.WriteFile(envFile, []byte("CUSTOM_VAR=value\n"), 0644)
	require.NoError(t, err)

	envLocalFile := filepath.Join(tmpDir, ".env.local")
	err = os.WriteFile(envLocalFile, []byte("LOCAL_VAR=value\n"), 0644)
	require.NoError(t, err)

	homeAdditionsFile := filepath.Join(tmpDir, ".bashrc.d", "custom.sh")
	err = os.MkdirAll(filepath.Dir(homeAdditionsFile), 0755)
	require.NoError(t, err)
	err = os.WriteFile(homeAdditionsFile, []byte("alias ll='ls -la'\n"), 0644)
	require.NoError(t, err)

	files := []string{hiddenFile, envFile, envLocalFile, homeAdditionsFile}
	customFiles := filterCustomConfigFiles(files, nil, map[string]string{}, false)

	require.Len(t, customFiles, 3)
	require.NotContains(t, customFiles, fileInfo{path: hiddenFile})

	customPaths := []string{customFiles[0].path, customFiles[1].path, customFiles[2].path}
	require.Contains(t, customPaths, envFile)
	require.Contains(t, customPaths, envLocalFile)
	require.Contains(t, customPaths, homeAdditionsFile)
}
