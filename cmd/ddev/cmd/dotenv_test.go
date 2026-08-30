package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
	copy2 "github.com/otiai10/copy"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	// A relative path that traverses outside the project root must be rejected too
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "get", "../.env.outside")
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "outside the project root")
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "set", "../.env.outside", "--test-value", "custom value")
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "outside the project root")
	require.NoFileExists(t, filepath.Join(testDir, "..", ".env.outside"))

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
	require.Contains(t, out, "the flag must consist of lowercase letters, numbers, and hyphens")

	out, err = exec.RunHostCommand(DdevBin, "dotenv", "set", envFile, "--1test", "custom value")
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "the flag must consist of lowercase letters, numbers, and hyphens")

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

// TestCmdDotEnvGlobalGetAndSet tests that `ddev dotenv global get` and `ddev dotenv global set`
// can read and write to global .env.* files without an active project
func TestCmdDotEnvGlobalGetAndSet(t *testing.T) {
	origDir, _ := os.Getwd()

	// A directory with no DDEV project in it
	testDir := testcommon.CreateTmpDir(t.Name())
	err := os.Chdir(testDir)
	require.NoError(t, err)

	globalDir := globalconfig.GetGlobalDdevDir()
	// Not a service name, so this file cannot change the environment of any running project
	envFileName := ".env.dotenv-global-test"
	envFileArg := filepath.Join(".ddev", envFileName)
	envFile := filepath.Join(globalDir, envFileName)

	t.Cleanup(func() {
		assert := asrt.New(t)
		err = os.RemoveAll(envFile)
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(testDir)
	})

	// Success while using the .ddev/<file> form
	out, err := exec.RunHostCommand(DdevBin, "dotenv", "global", "set", envFileArg, "--test-value", "custom value")
	require.NoError(t, err, "out=%s", out)
	require.FileExists(t, envFile, "unable to find %s file, but it should be here", envFile)
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "global", "get", envFileArg, "--test-value")
	require.NoError(t, err, "out=%s", out)
	require.Equal(t, out, "custom value\n")

	// The 'add' alias works the same way
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "global", "add", envFileArg, "--another-value=foobar")
	require.NoError(t, err, "out=%s", out)
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "global", "get", envFileArg, "--another-value")
	require.NoError(t, err, "out=%s", out)
	require.Equal(t, out, "foobar\n")

	// A bare filename is rejected
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "global", "get", envFileName, "--test-value")
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "must begin with .ddev/")
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "global", "set", ".env.bare", "--test-value", "custom value")
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "must begin with .ddev/")
	require.NoFileExists(t, filepath.Join(globalDir, ".env.bare"))

	// A relative path that traverses outside the global DDEV directory must be rejected too.
	// Built by hand, not filepath.Join, so the second one reaches the check uncleaned.
	sep := string(filepath.Separator)
	for _, arg := range []string{
		".." + sep + ".env.outside",
		".ddev" + sep + ".." + sep + ".." + sep + ".env.outside",
		".." + sep + "outside" + sep + ".ddev" + sep + ".env",
	} {
		out, err = exec.RunHostCommand(DdevBin, "dotenv", "global", "get", arg, "--test-value")
		require.Error(t, err, "out=%s", out)
		require.Contains(t, out, "must begin with .ddev/")
		out, err = exec.RunHostCommand(DdevBin, "dotenv", "global", "set", arg, "--test-value", "custom value")
		require.Error(t, err, "out=%s", out)
		require.Contains(t, out, "must begin with .ddev/")
	}
	require.NoFileExists(t, filepath.Join(globalDir, "..", ".env.outside"))
	require.NoFileExists(t, filepath.Join(testDir, "..", ".env.outside"))

	// Success while using full path to the .env file
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "global", "get", envFile, "--test-value")
	require.NoError(t, err, "out=%s", out)
	require.Equal(t, out, "custom value\n")

	// It's important that these commands don't have access to the filesystem outside the global directory
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "global", "get", filepath.Join(testDir, ".env"), "--test-value")
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "outside the global DDEV directory")
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "global", "set", filepath.Join(testDir, ".env"), "--test-value", "custom value")
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "outside the global DDEV directory")

	// The same naming validation as in the project scope
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "global", "set", filepath.Join(".ddev", ".envfoo"), "--test-value", "custom value")
	require.Error(t, err, "out=%s", out)
	require.Contains(t, out, "The file should have .env prefix")
	require.NoFileExists(t, filepath.Join(globalDir, ".envfoo"))

	// While the project scope still requires a project
	out, err = exec.RunHostCommand(DdevBin, "dotenv", "set", filepath.Join(".ddev", ".env"), "--test-value", "custom value")
	require.Error(t, err, "out=%s", out)
}
