package cmd

import (
	"testing"

	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

// TestDescribeBadArgs ensures the binary behaves as expected when used with invalid arguments or working directories.
func TestDescribeBadArgs(t *testing.T) {
	assert := assert.New(t)

	// Create a temporary directory and switch to it for the duration of this test.
	tmpdir := testcommon.CreateTmpDir("badargs")
	defer testcommon.Chdir(tmpdir)()
	defer testcommon.CleanupDir(tmpdir)

	// Ensure it fails if we run the vanilla describe outside of an application root.
	args := []string{"describe"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "unable to determine the application for this command")

	// Ensure we get a failure if we run a describe on a named application which does not exist.
	args = []string{"describe", util.RandString(16)}
	out, err = exec.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Could not describe app")

	// Ensure we get a failure if using too many arguments.
	args = []string{"describe", util.RandString(16), util.RandString(16)}
	out, err = exec.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Too many arguments provided")

}

// TestDescribe tests that the describe command works properly when using the binary.
func TestDescribe(t *testing.T) {
	assert := assert.New(t)

	for _, v := range DevTestSites {
		// First, try to do a describe from another directory.
		tmpdir := testcommon.CreateTmpDir("")
		cleanup := testcommon.Chdir(tmpdir)
		defer testcommon.CleanupDir(tmpdir)

		args := []string{"describe", v.Name}
		out, err := exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.Contains(string(out), "NAME")
		assert.Contains(string(out), "LOCATION")
		assert.Contains(string(out), v.Name)
		assert.Contains(string(out), "running")

		cleanup()

		cleanup = v.Chdir()

		args = []string{"describe"}
		out, err = exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.Contains(string(out), "NAME")
		assert.Contains(string(out), "LOCATION")
		assert.Contains(string(out), v.Name)
		assert.Contains(string(out), "running")

		cleanup()
	}
}

// TestDescribeAppFunction performs unit tests on the describeApp function from the working directory.
func TestDescribeAppFunction(t *testing.T) {
	assert := assert.New(t)
	for _, v := range DevTestSites {
		cleanup := v.Chdir()

		app, err := platform.GetActiveApp("")
		assert.NoError(err)

		out, err := describeApp("")
		assert.NoError(err)
		assert.Contains(string(out), app.URL())
		assert.Contains(string(out), app.GetName())
		assert.Contains(string(out), "running")
		assert.Contains(string(out), v.Dir)
		assert.Contains(string(out), "DDEV ROUTER STATUS: "+color.CyanString("running"))

		_, err = exec.RunCommand("docker", []string{"stop", "ddev-router"})
		assert.NoError(err)
		out, err = describeApp("")
		assert.NoError(err)
		assert.Contains(string(out), "DDEV ROUTER STATUS: "+color.RedString("stopped"))
		assert.Contains(string(out), "The router is not currently running")
		_, err = exec.RunCommand("docker", []string{"start", "ddev-router"})
		assert.NoError(err)

		cleanup()
	}
}

// TestDescribeAppUsingSitename performs unit tests on the describeApp function using the sitename as an argument.
func TestDescribeAppUsingSitename(t *testing.T) {
	assert := assert.New(t)

	// Create a temporary directory and switch to it for the duration of this test.
	tmpdir := testcommon.CreateTmpDir("describeAppUsingSitename")
	defer testcommon.Chdir(tmpdir)()
	defer testcommon.CleanupDir(tmpdir)

	for _, v := range DevTestSites {
		out, err := describeApp(v.Name)
		assert.NoError(err)
		assert.Contains(string(out), "running")
		assert.Contains(string(out), v.Dir)
	}
}

// TestDescribeAppWithInvalidParams performs unit tests on the describeApp function using a variety of invalid parameters.
func TestDescribeAppWithInvalidParams(t *testing.T) {
	assert := assert.New(t)

	// Create a temporary directory and switch to it for the duration of this test.
	tmpdir := testcommon.CreateTmpDir("TestDescribeAppWithInvalidParams")
	defer testcommon.Chdir(tmpdir)()
	defer testcommon.CleanupDir(tmpdir)

	// Ensure describeApp fails from an invalid working directory.
	_, err := describeApp("")
	assert.Error(err)

	// Ensure describeApp fails with invalid site-names.
	_, err = describeApp(util.RandString(16))
	assert.Error(err)

	// Change to a sites working directory and ensure a failure still occurs with a invalid site name.
	cleanup := DevTestSites[0].Chdir()
	_, err = describeApp(util.RandString(16))
	assert.Error(err)
	cleanup()

}
