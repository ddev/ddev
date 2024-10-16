package cmd

import (
	"github.com/ddev/ddev/pkg/testcommon"
	copy2 "github.com/otiai10/copy"
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/stretchr/testify/require"

	asrt "github.com/stretchr/testify/assert"
)

// TestCmdDotEnvSetAndGet tests that `ddev dotenv get` and `ddev dotenv set` can read and write to .ddev/.env.* files
func TestCmdDotEnvGetAndSet(t *testing.T) {
	assert := asrt.New(t)

	// Create a temporary directory and switch to it.
	origDir, _ := os.Getwd()

	testDir := testcommon.CreateTmpDir(t.Name())
	err := os.Chdir(testDir)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err = exec.RunHostCommand(DdevBin, "delete", "-Oy", t.Name())
		assert.NoError(err)

		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(testDir)
	})

	out, err := exec.RunHostCommand(DdevBin, "config", "--project-name", t.Name())
	require.NoError(t, err, "Failed to run ddev config: %s", out)

	// It's important that these commands don't have access to the filesystem outside the project
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "get", filepath.Join(origDir, "testdata", t.Name()))
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "outside the project root")
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "set", filepath.Join(origDir, "testdata", t.Name()), "--test-value", "custom value")
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "outside the project root")

	// Success while using full path to the .env file
	envFile := filepath.Join(testDir, ".env")
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "set", envFile, "--test-value", "custom value")
	require.NoError(t, err, "out=%s", out)
	require.FileExists(t, envFile, "unable to find .env file, but it should be here")
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "get", envFile, "--test-value")
	require.NoError(t, err, "out=%s", out)
	require.Equal(t, out, "custom value\n")

	// Success while using relative path to the .env file
	envFileRelative := filepath.Join(".ddev", ".env.relative")
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "set", envFileRelative, "--test-value", "custom value")
	require.NoError(t, err, "out=%s", out)
	require.FileExists(t, envFile, "unable to find .env file, but it should be here")
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "get", envFileRelative, "--test-value")
	require.NoError(t, err, "out=%s", out)
	require.Equal(t, out, "custom value\n")

	envFileWrongNaming := filepath.Join(testDir, "some-file")
	err = copy2.Copy(envFile, envFileWrongNaming)
	require.NoError(t, err, "out=%s", out)

	// Test some validation errors
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "get", envFileWrongNaming, "--test-value")
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "The file should have .env prefix")

	out, err = exec.RunHostCommand(DdevBin, "dotenv", "set", envFileWrongNaming, "--test-value", "custom value")
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "The file should have .env prefix")

	out, err = exec.RunHostCommand(DdevBin, "dotenv", "set", envFile, "-a", "custom value")
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "flag must be in long format")

	out, err = exec.RunHostCommand(DdevBin, "dotenv", "set", envFile, "--TEST", "custom value")
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "the flag must be lowercase and start with a letter")

	out, err = exec.RunHostCommand(DdevBin, "dotenv", "set", envFile, "--1test", "custom value")
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "the flag must be lowercase and start with a letter")

	out, err = exec.RunHostCommand(DdevBin, "dotenv", "get", envFile, "--test-value", "--test-value-2")
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "one environment variable can be retrieved at a time")

	out, err = exec.RunHostCommand(DdevBin, "dotenv", "get", envFile, "--test-value-unknown")
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "environment variable 'TEST_VALUE_UNKNOWN' not found")

	out, err = exec.RunHostCommand(DdevBin, "dotenv", "get", envFile, "-a")
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "flag must be in long format")

	out, err = exec.RunHostCommand(DdevBin, "dotenv", "set", envFile, "--test-value-with-special-characters", `Test$variable\nwith\n_new_lines`)
	require.NoError(t, err, "out=%s", out)

	// New line character `\n` should be read as it is, without a multiline
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "get", envFile, "--test-value-with-special-characters")
	require.NoError(t, err, "out=%s", out)
	require.Equal(t, out, `Test$variable\nwith\n_new_lines`+"\n")

	// Double quotes should be escaped, and test setting several variables at once
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "set", envFile, "--my-value", `"t'h"i's`, "--another-value", "foobar")
	require.NoError(t, err, "out=%s", out)
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "get", envFile, "--my-value")
	require.NoError(t, err, "out=%s", out)
	require.Equal(t, out, `\"t'h\"i's`+"\n")
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "get", envFile, "--another-value")
	require.NoError(t, err, "out=%s", out)
	require.Equal(t, out, "foobar\n")

	// And check the value from the test start to make sure that file was not overwritten
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "get", envFile, "--test-value")
	require.NoError(t, err, "out=%s", out)
	require.Equal(t, out, "custom value\n")
}
