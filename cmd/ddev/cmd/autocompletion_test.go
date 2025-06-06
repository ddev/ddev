package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
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
	origDir, _ := os.Getwd()

	t.Cleanup(func() {
		removeSites()
		_ = os.Chdir(origDir)
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

	origDir, _ := os.Getwd()

	t.Cleanup(func() {
		removeSites()
		_ = os.Chdir(origDir)
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

// TestAutocompletionForDescribeCmd checks autocompletion of project names for ddev describe
func TestAutocompletionForDescribeCmd(t *testing.T) {
	assert := asrt.New(t)

	// Skip if we don't have enough tests.
	if len(TestSites) < 2 {
		t.Skip("Must have at least two test sites to test autocompletion")
	}
	origDir, _ := os.Getwd()

	t.Cleanup(func() {
		removeSites()
		pwd, err := os.Getwd()
		util.Debug("pwd=%s, err=%v", pwd, err)
		_ = os.Chdir(origDir)
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

// TestAutocompletionForConfigCmd checks autocompletion for ddev config
func TestAutocompletionForConfigCmd(t *testing.T) {
	assert := asrt.New(t)

	testCases := map[string][]string{
		"--project-type":                ddevapp.GetValidAppTypes(),
		"--php-version":                 nodeps.GetValidPHPVersions(),
		"--router-http-port":            {nodeps.DdevDefaultRouterHTTPPort},
		"--router-https-port":           {nodeps.DdevDefaultRouterHTTPSPort},
		"--xdebug-enabled":              {"true", "false"},
		"--no-project-mount":            {"true", "false"},
		"--omit-containers":             nodeps.GetValidOmitContainers(),
		"--webserver-type":              nodeps.GetValidWebserverTypes(),
		"--performance-mode":            types.ValidPerformanceModeOptions(types.ConfigTypeProject),
		"--xhprof-mode":                 types.ValidXHProfModeOptions(types.ConfigTypeProject),
		"--fail-on-hook-fail=":          {"true", "false"},
		"--mailpit-http-port":           {nodeps.DdevDefaultMailpitHTTPPort},
		"--mailpit-https-port":          {nodeps.DdevDefaultMailpitHTTPSPort},
		"--project-tld":                 {nodeps.DdevDefaultTLD},
		"--use-dns-when-possible":       {"true", "false"},
		"--disable-settings-management": {"true", "false"},
		"--composer-version":            {"2", "2.2", "1", "stable", "preview", "snapshot"},
		"--bind-all-interfaces":         {"true", "false"},
		"--database":                    nodeps.GetValidDatabaseVersions(),
		"--nodejs-version":              {nodeps.NodeJSDefault, "auto", "engine"},
		"--default-container-timeout":   {nodeps.DefaultDefaultContainerTimeout},
		"--disable-upload-dirs-warning": {"true", "false"},
		"--corepack-enable":             {"true", "false"},
	}

	for flag, expected := range testCases {
		t.Run(flag, func(t *testing.T) {
			var out string
			var err error
			if reflect.DeepEqual(expected, []string{"true", "false"}) {
				// Cobra autocompletion for boolean works only with equal sign
				out, err = exec.RunHostCommand(DdevBin, "__complete", "config", flag+`=""`)
			} else {
				out, err = exec.RunHostCommand(DdevBin, "__complete", "config", flag, "")
			}
			assert.NoError(err)
			for _, val := range expected {
				assert.Contains(out, val)
			}
		})
	}

	// --omit-containers is special because it can take multiple values separated by commas
	tests := []struct {
		input       string
		expected    []string
		notExpected []string
	}{
		{"", []string{"db", "ddev-ssh-agent"}, nil},
		{"db,", []string{"ddev-ssh-agent"}, nil},
		{"ddev-ssh-agent,", []string{"db"}, nil},
		{"db,ddev-ssh-agent,", nil, []string{"db", "ddev-ssh-agent"}},
		{"ddev-ssh-agent,db,", nil, []string{"db", "ddev-ssh-agent"}},
	}

	for _, tc := range tests {
		t.Run("--omit-containers="+tc.input, func(t *testing.T) {
			out, err := exec.RunHostCommand(DdevBin, "__complete", "config", "--omit-containers", tc.input)
			assert.NoError(err)
			if tc.expected != nil {
				for _, val := range tc.expected {
					assert.Contains(out, val)
				}
			}
			if tc.notExpected != nil {
				for _, val := range tc.notExpected {
					assert.NotContains(out, val)
				}
			}
		})
	}
}

// TestAutocompletionConfigGlobalCmd checks autocompletion for ddev config global
func TestAutocompletionConfigGlobalCmd(t *testing.T) {
	assert := asrt.New(t)

	testCases := map[string][]string{
		"--omit-containers":               globalconfig.GetValidOmitContainers(),
		"--instrumentation-opt-in":        {"true", "false"},
		"--router-bind-all-interfaces":    {"true", "false"},
		"--internet-detection-timeout-ms": {strconv.Itoa(nodeps.InternetDetectionTimeoutDefault)},
		"--use-letsencrypt":               {"true", "false"},
		"--simple-formatting":             {"true", "false"},
		"--use-hardened-images":           {"true", "false"},
		"--fail-on-hook-fail":             {"true", "false"},
		"--performance-mode":              types.ValidPerformanceModeOptions(types.ConfigTypeGlobal),
		"--xhprof-mode":                   types.ValidXHProfModeOptions(types.ConfigTypeGlobal),
		"--table-style":                   globalconfig.ValidTableStyleList(),
		"--project-tld":                   {nodeps.DdevDefaultTLD},
		"--no-bind-mounts":                {"true", "false"},
		"--router-http-port":              {nodeps.DdevDefaultRouterHTTPPort},
		"--router-https-port":             {nodeps.DdevDefaultRouterHTTPSPort},
		"--mailpit-http-port":             {nodeps.DdevDefaultMailpitHTTPPort},
		"--mailpit-https-port":            {nodeps.DdevDefaultMailpitHTTPSPort},
		"--traefik-monitor-port":          {nodeps.TraefikMonitorPortDefault},
	}

	for flag, expected := range testCases {
		t.Run(flag, func(t *testing.T) {
			var out string
			var err error
			if reflect.DeepEqual(expected, []string{"true", "false"}) {
				// Cobra autocompletion for boolean works only with equal sign
				out, err = exec.RunHostCommand(DdevBin, "__complete", "config", "global", flag+`=""`)
			} else {
				out, err = exec.RunHostCommand(DdevBin, "__complete", "config", "global", flag, "")
			}
			assert.NoError(err)
			for _, val := range expected {
				if val != "" {
					assert.Contains(out, val)
				}
			}
		})
	}

	// --omit-containers is special because it can take multiple values separated by commas
	tests := []struct {
		input       string
		expected    []string
		notExpected []string
	}{
		{"", []string{"ddev-router", "ddev-ssh-agent"}, nil},
		{"ddev-router,", []string{"ddev-ssh-agent"}, nil},
		{"ddev-ssh-agent,", []string{"ddev-router"}, nil},
		{"ddev-router,ddev-ssh-agent,", nil, []string{"ddev-router", "ddev-ssh-agent"}},
		{"ddev-ssh-agent,ddev-router,", nil, []string{"ddev-router", "ddev-ssh-agent"}},
	}

	for _, tc := range tests {
		t.Run("--omit-containers="+tc.input, func(t *testing.T) {
			out, err := exec.RunHostCommand(DdevBin, "__complete", "config", "global", "--omit-containers", tc.input)
			assert.NoError(err)
			if tc.expected != nil {
				for _, val := range tc.expected {
					assert.Contains(out, val)
				}
			}
			if tc.notExpected != nil {
				for _, val := range tc.notExpected {
					assert.NotContains(out, val)
				}
			}
		})
	}
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
		err = app.Stop(true, false)
		assert.NoError(err)
		testcommon.ResetGlobalDdevDir(t, tmpXdgConfigHomeDir)
		_ = fileutil.PurgeDirectory(filepath.Join(site.Dir, ".ddev", "commands"))
		_ = os.Chdir(origDir)
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

// TestAutocompleteServiceForServiceFlag checks service autocompletion for service flag
func TestAutocompleteServiceForServiceFlag(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	app, err := ddevapp.NewApp("", false)
	assert.NoError(err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		_ = os.Chdir(origDir)
	})
	err = app.Start()
	require.NoError(t, err)

	// Check completion results are as expected for each command
	for _, cmd := range []string{"exec", "logs", "ssh"} {
		out, err := exec.RunHostCommand(DdevBin, "__complete", cmd, "-s", "")
		assert.NoError(err)
		assert.Contains(out, "web")
		assert.Contains(out, "db")
		// xhgui is not running, so it should not be in the output
		assert.NotContains(out, "xhgui")

		// check with project argument
		if cmd != "exec" {
			out, err := exec.RunHostCommand(DdevBin, "__complete", cmd, site.Name, "-s", "")
			assert.NoError(err)
			assert.Contains(out, "web")
			assert.Contains(out, "db")
			// xhgui is not running, so it should not be in the output
			assert.NotContains(out, "xhgui")
		}
	}

	// ddev debug rebuild should contain all services, no matter if they are running or not
	out, err := exec.RunHostCommand(DdevBin, "__complete", "debug", "rebuild", "-s", "")
	assert.NoError(err)
	assert.Contains(out, "web")
	assert.Contains(out, "db")
	assert.Contains(out, "xhgui")

	err = app.Stop(false, false)
	require.NoError(t, err)

	// Check completion results are as expected for each command
	for _, cmd := range []string{"exec", "logs", "ssh"} {
		out, err := exec.RunHostCommand(DdevBin, "__complete", cmd, "-s", "")
		assert.NoError(err)
		// not running services should not be here
		assert.NotContains(out, "web")
		assert.NotContains(out, "db")
		assert.NotContains(out, "xhgui")

		// check with project argument
		if cmd != "exec" {
			out, err := exec.RunHostCommand(DdevBin, "__complete", cmd, site.Name, "-s", "")
			assert.NoError(err)
			// not running services should not be here
			assert.NotContains(out, "web")
			assert.NotContains(out, "db")
			assert.NotContains(out, "xhgui")
		}
	}

	// ddev debug rebuild should contain all services, no matter if they are running or not (with project argument)
	out, err = exec.RunHostCommand(DdevBin, "__complete", "debug", "rebuild", site.Name, "-s", "")
	assert.NoError(err)
	assert.Contains(out, "web")
	assert.Contains(out, "db")
	assert.Contains(out, "xhgui")
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
		err = app.Stop(true, false)
		assert.NoError(err)
		testcommon.ResetGlobalDdevDir(t, tmpXdgConfigHomeDir)
		_ = fileutil.PurgeDirectory(filepath.Join(site.Dir, ".ddev", "commands"))
		_ = os.Chdir(origDir)
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
