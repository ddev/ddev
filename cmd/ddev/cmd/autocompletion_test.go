package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAutocompletionForStopCmd checks autocompletion of project names for ddev stop
func TestAutocompletionForStopCmd(t *testing.T) {
	assert := asrt.New(t)

	// Skip if we don't have enough tests.
	if len(TestSites) < 2 {
		t.Skip("Must have at least two test sites to test autocompletion")
	}

	t.Cleanup(func() {
		removeSites()
	})

	// Make sure we have some sites.
	err := addSites()
	require.NoError(t, err)

	var siteNames []string
	for _, site := range TestSites {
		siteNames = append(siteNames, site.Name)
	}

	// ddev stop should show all running sites
	out, err := exec.RunHostCommand(DdevBin, "__complete", "stop", "")
	assert.NoError(err)
	filteredOut := getTestingSitesFromOutput(out)
	for _, name := range siteNames {
		assert.Contains(filteredOut, name)
	}

	// Project names shouldn't be repeated
	out, err = exec.RunHostCommand(DdevBin, "__complete", "stop", siteNames[0], "")
	assert.NoError(err)
	filteredOut = getTestingSitesFromOutput(out)
	for i, name := range siteNames {
		if i == 0 {
			assert.NotContains(filteredOut, name)
		} else {
			assert.Contains(filteredOut, name)
		}
	}

	// If we've used all the project names, there should be no further suggestions
	allArgs := append([]string{"__complete", "stop"}, siteNames...)
	allArgs = append(allArgs, "")
	out, err = exec.RunHostCommand(DdevBin, allArgs...)
	assert.NoError(err)
	filteredOut = getTestingSitesFromOutput(out)
	assert.Empty(filteredOut)

	// If a project is stopped, it shouldn't be suggested for stop anymore
	_, err = exec.RunHostCommand(DdevBin, "stop", siteNames[0])
	assert.NoError(err)
	out, err = exec.RunHostCommand(DdevBin, "__complete", "stop", "")
	assert.NoError(err)
	filteredOut = getTestingSitesFromOutput(out)
	for i, name := range siteNames {
		if i == 0 {
			assert.NotContains(filteredOut, name)
		} else {
			assert.Contains(filteredOut, name)
		}
	}

	// If all projects are stopped (or no projects exist), completion should be empty
	allArgs = append([]string{"stop"}, siteNames...)
	_, err = exec.RunHostCommand(DdevBin, allArgs...)
	assert.NoError(err)
	out, err = exec.RunHostCommand(DdevBin, "__complete", "stop", "")
	assert.NoError(err)
	filteredOut = getTestingSitesFromOutput(out)
	assert.Empty(filteredOut)
}

// TestAutocompletionForStartCmd checks autocompletion of project names for ddev start
func TestAutocompletionForStartCmd(t *testing.T) {
	assert := asrt.New(t)

	// Skip if we don't have enough tests.
	if len(TestSites) < 2 {
		t.Skip("Must have at least two test sites to test autocompletion")
	}

	t.Cleanup(func() {
		removeSites()
	})

	// Make sure we have some sites.
	err := addSites()
	require.NoError(t, err)

	var siteNames []string
	for _, site := range TestSites {
		siteNames = append(siteNames, site.Name)
	}

	// If all projects are running, completion should be empty
	out, err := exec.RunHostCommand(DdevBin, "__complete", "start", "")
	assert.NoError(err)
	filteredOut := getTestingSitesFromOutput(out)
	assert.Empty(filteredOut)

	// All stopped projects should display in completion
	_, err = exec.RunHostCommand(DdevBin, "stop", siteNames[0])
	assert.NoError(err)
	out, err = exec.RunHostCommand(DdevBin, "__complete", "start", "")
	assert.NoError(err)
	filteredOut = getTestingSitesFromOutput(out)
	for i, name := range siteNames {
		if i == 0 {
			assert.Contains(filteredOut, name)
		} else {
			assert.NotContains(filteredOut, name)
		}
	}

	allArgs := append([]string{"stop"}, siteNames...)
	_, err = exec.RunHostCommand(DdevBin, allArgs...)
	assert.NoError(err)
	out, err = exec.RunHostCommand(DdevBin, "__complete", "start", "")
	assert.NoError(err)
	filteredOut = getTestingSitesFromOutput(out)
	for _, name := range siteNames {
		assert.Contains(filteredOut, name)
	}

	// Project names shouldn't be repeated
	out, err = exec.RunHostCommand(DdevBin, "__complete", "start", siteNames[0], "")
	assert.NoError(err)
	filteredOut = getTestingSitesFromOutput(out)
	for i, name := range siteNames {
		if i == 0 {
			assert.NotContains(filteredOut, name)
		} else {
			assert.Contains(filteredOut, name)
		}
	}

	// If we've used all the project names, there should be no further suggestions
	allArgs = append([]string{"__complete", "start"}, siteNames...)
	allArgs = append(allArgs, "")
	out, err = exec.RunHostCommand(DdevBin, allArgs...)
	assert.NoError(err)
	filteredOut = getTestingSitesFromOutput(out)
	assert.Empty(filteredOut)
}

// TestAutocompletionForStartCmd checks autocompletion of project names for ddev describe
func TestAutocompletionForDescribeCmd(t *testing.T) {
	assert := asrt.New(t)

	// Skip if we don't have enough tests.
	if len(TestSites) < 2 {
		t.Skip("Must have at least two test sites to test autocompletion")
	}

	t.Cleanup(func() {
		removeSites()
	})

	// Make sure we have some sites.
	err := addSites()
	require.NoError(t, err)

	var siteNames []string
	for _, site := range TestSites {
		siteNames = append(siteNames, site.Name)
	}

	// If all projects are running, completion should show all project names
	out, err := exec.RunHostCommand(DdevBin, "__complete", "describe", "")
	assert.NoError(err)
	filteredOut := getTestingSitesFromOutput(out)
	for _, name := range siteNames {
		assert.Contains(filteredOut, name)
	}

	// Even stopped projects should display in completion
	_, err = exec.RunHostCommand(DdevBin, "stop", siteNames[0])
	assert.NoError(err)
	out, err = exec.RunHostCommand(DdevBin, "__complete", "describe", "")
	assert.NoError(err)
	filteredOut = getTestingSitesFromOutput(out)
	for _, name := range siteNames {
		assert.Contains(filteredOut, name)
	}

	// If there's already an argument, nothing more should be suggested
	out, err = exec.RunHostCommand(DdevBin, "__complete", "describe", "anything", "")
	assert.NoError(err)
	filteredOut = getTestingSitesFromOutput(out)
	assert.Empty(filteredOut)
}

// TestAutocompletionForCustomCmds checks custom autocompletion for custom host and container commands
func TestAutocompletionForCustomCmds(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping because untested on Windows")
	}
	if dockerutil.IsColima() {
		t.Skip("Skipping on Colima because of slow mount responses")
	}
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	app, err := ddevapp.NewApp("", false)
	assert.NoError(err)

	tmpXdgConfigHomeDir := testcommon.CopyGlobalDdevDir(t)

	testdataCustomCommandsDir := filepath.Join(origDir, "testdata", t.Name())

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		testcommon.ResetGlobalDdevDir(t, tmpXdgConfigHomeDir)
		_ = fileutil.PurgeDirectory(filepath.Join(site.Dir, ".ddev", "commands"))
	})
	err = app.Start()
	require.NoError(t, err)

	tmpHomeGlobalCommandsDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands")
	projectCommandsDir := app.GetConfigPath("commands")

	// Remove existing commands
	err = os.RemoveAll(tmpHomeGlobalCommandsDir)
	assert.NoError(err)
	err = os.RemoveAll(projectCommandsDir)
	assert.NoError(err)
	// Copy project and global commands into project
	err = fileutil.CopyDir(filepath.Join(testdataCustomCommandsDir, "project_commands"), projectCommandsDir)
	assert.NoError(err)
	err = fileutil.CopyDir(filepath.Join(testdataCustomCommandsDir, "global_commands"), tmpHomeGlobalCommandsDir)
	require.NoError(t, err)
	_, _ = exec.RunHostCommand(DdevBin, "debug", "fix-commands")

	// Must sync our added commands before using them.
	err = app.MutagenSyncFlush()
	assert.NoError(err)

	// Check completion results are as expected for each command
	for _, cmd := range []string{"global-host-cmd", "global-web-cmd", "project-host-cmd", "project-web-cmd"} {
		out, err := exec.RunHostCommand(DdevBin, "__complete", cmd, "anArg")
		assert.NoError(err)
		expectedHost, _ := os.Hostname()
		if !strings.Contains(cmd, "host") {
			expectedHost = site.Name + "-web"
		}
		// We're not testing the internals of cobra's completion so we don't want to assert the exact output,
		// just check that the suggestions we expect are included in the output.
		assert.Contains(out, expectedHost)
		assert.Contains(out, cmd)
		assert.Contains(out, "anArg")
	}
}

// TestAutocompleteTermsForCustomCmds tests the AutocompleteTerms annotation for custom host and container commands
func TestAutocompleteTermsForCustomCmds(t *testing.T) {
	if dockerutil.IsColima() || dockerutil.IsLima() {
		t.Skip("Skipping on Colima/Lima")
	}
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	app, err := ddevapp.NewApp("", false)
	assert.NoError(err)

	tmpXdgConfigHomeDir := testcommon.CopyGlobalDdevDir(t)

	testdataCustomCommandsDir := filepath.Join(origDir, "testdata", t.Name())

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		testcommon.ResetGlobalDdevDir(t, tmpXdgConfigHomeDir)
		_ = fileutil.PurgeDirectory(filepath.Join(site.Dir, ".ddev", "commands"))
	})
	err = app.Start()
	require.NoError(t, err)

	tmpHomeGlobalCommandsDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands")
	projectCommandsDir := app.GetConfigPath("commands")

	// Remove existing commands
	err = os.RemoveAll(tmpHomeGlobalCommandsDir)
	assert.NoError(err)
	err = os.RemoveAll(projectCommandsDir)
	assert.NoError(err)
	// Copy project and global commands into project
	err = fileutil.CopyDir(filepath.Join(testdataCustomCommandsDir, "project_commands"), projectCommandsDir)
	assert.NoError(err)
	err = fileutil.CopyDir(filepath.Join(testdataCustomCommandsDir, "global_commands"), tmpHomeGlobalCommandsDir)
	require.NoError(t, err)
	_, _ = exec.RunHostCommand(DdevBin, "debug", "fix-commands")

	// Must sync our added commands before using them.
	err = app.MutagenSyncFlush()
	assert.NoError(err)

	// Check completion results are as expected for each command
	for _, cmd := range []string{"global-host-cmd", "global-web-cmd", "project-host-cmd", "project-web-cmd"} {
		out, err := exec.RunHostCommand(DdevBin, "__complete", cmd, "")
		assert.NoError(err)
		// We're not testing the internals of cobra's completion so we don't want to assert the exact output,
		// just check that the suggestions we expect are included in the output.
		assert.Contains(out, strings.Replace(cmd, "cmd", "one", 1))
		assert.Contains(out, "suggest two")
		assert.Contains(out, "three")
	}
}

// getTestingSitesFromOutput() finds only the ddev list items that
// have names starting with "Test" from a space separated list of project names.
// This is useful when running the tests locally, to filter out projects that
// aren't test-related.
func getTestingSitesFromOutput(output string) []interface{} {
	testSites := make([]interface{}, 0)
	for _, siteName := range strings.Fields(output) {
		if strings.HasPrefix(siteName, "Test") {
			testSites = append(testSites, siteName)
		}
	}
	return testSites
}
