package cmd

import (
	"bufio"
	"os"
	oexec "os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCmdList runs the binary with "ddev list" and checks the results
func TestCmdList(t *testing.T) {
	assert := asrt.New(t)
	origDir, _ := os.Getwd()
	origDdevDebug := os.Getenv("DDEV_DEBUG")
	_ = os.Unsetenv("DDEV_DEBUG")

	site := TestSites[0]
	err := os.Chdir(site.Dir)

	globalconfig.DdevGlobalConfig.SimpleFormatting = true
	_ = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		globalconfig.DdevGlobalConfig.SimpleFormatting = false
		_ = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		_ = os.Setenv("DDEV_DEBUG", origDdevDebug)
	})
	// This gratuitous ddev start -a repopulates the ~/.ddev/project_list.yaml
	// project list, which has been damaged by other tests which use
	// direct app techniques.
	_, err = exec.RunHostCommand(DdevBin, "start", "-a", "-y")
	assert.NoError(err)

	// Execute "ddev list" and harvest plain text output.
	out, err := exec.RunHostCommand(DdevBin, "list", "-W")
	assert.NoError(err, "error running ddev list: %v output=%s", out)

	// Execute "ddev list -j" and harvest the json output
	jsonOut, err := exec.RunHostCommand(DdevBin, "list", "-j")
	assert.NoError(err, "error running ddev list -j: %v, output=%s", jsonOut)

	siteList := getTestingSitesFromList(t, jsonOut)
	assert.Equal(len(TestSites), len(siteList), "didn't find expected number of sites in list: %v", siteList)

	for _, v := range TestSites {
		app, err := ddevapp.GetActiveApp(v.Name)
		assert.NoError(err)

		// Look for standard items in the regular ddev list output
		assert.Contains(string(out), v.Name)
		testURL := app.GetHTTPSURL()
		if globalconfig.GetCAROOT() == "" {
			testURL = app.GetHTTPURL()
		}
		assert.Contains(string(out), testURL)
		assert.Contains(string(out), app.GetType())
		assert.Contains(string(out), fileutil.ShortHomeJoin(app.GetAppRoot()))

		// Look through list results in json for this site.
		found := false
		for _, listitem := range siteList {
			item, ok := listitem.(map[string]interface{})
			assert.True(ok)
			// Check to see that we can find our item
			if item["name"] == v.Name {
				found = true
				assert.Equal(app.GetHTTPURL(), item["httpurl"])
				assert.Equal(app.GetHTTPSURL(), item["httpsurl"])
				assert.Equal(app.Name, item["name"])
				assert.Equal(app.GetType(), item["type"])
				assert.EqualValues(fileutil.ShortHomeJoin(app.GetAppRoot()), item["shortroot"])
				assert.EqualValues(app.GetAppRoot(), item["approot"])
				break
			}
		}
		assert.True(found, "Failed to find project %s in ddev list -j", v.Name)
	}

	// Now filter the list by the type of the first running test app
	jsonOut, err = exec.RunHostCommand(DdevBin, "list", "-j", "--type", TestSites[0].Type)
	assert.NoError(err, "error running ddev list: %v output=%s", err, out)
	siteList = getTestingSitesFromList(t, jsonOut)
	assert.GreaterOrEqual(len(siteList), 1)

	// Now filter the list by a not existing type
	jsonOut, err = exec.RunHostCommand(DdevBin, "list", "-j", "--type", "not-existing-type")
	assert.NoError(err, "error running ddev list: %v output=%s", err, out)
	siteList = getTestingSitesFromList(t, jsonOut)
	assert.Equal(0, len(siteList))

	// Stop the first app
	out, err = exec.RunHostCommand(DdevBin, "stop", TestSites[0].Name)
	assert.NoError(err, "error running ddev stop %v: %v output=%s", TestSites[0].Name, err, out)

	// Execute "ddev list" and harvest json output.
	// Now there should be one less active project in list
	jsonOut, err = exec.RunHostCommand(DdevBin, "list", "-jA", "-W")
	assert.NoError(err, "error running ddev list: %v output=%s", err, jsonOut)

	siteList = getTestingSitesFromList(t, jsonOut)
	assert.Equal(len(TestSites)-1, len(siteList))

	// Now list without -A, make sure we show all projects
	jsonOut, err = exec.RunHostCommand(DdevBin, "list", "-j")
	assert.NoError(err, "error running ddev list: %v output=%s", err, out)
	siteList = getTestingSitesFromList(t, jsonOut)
	assert.Equal(len(TestSites), len(siteList))

	// Leave firstApp running for other tests
	out, err = exec.RunHostCommand(DdevBin, "start", "-y", TestSites[0].Name)
	assert.NoError(err, "error running ddev start: %v output=%s", err, out)
}

// getSitesFromList takes the json output of ddev list -j
// and returns the list of *test* sites ddev list returns as an array
// of interface{}
func getSitesFromList(t *testing.T, jsonOut string) []interface{} {
	assert := asrt.New(t)

	logItems, err := unmarshalJSONLogs(jsonOut)
	require.NoError(t, err)
	data := logItems[len(logItems)-1]
	assert.EqualValues(data["level"], "info")

	raw, ok := data["raw"].([]interface{})
	require.True(t, ok)
	return raw
}

// getTestingSitesFromList() finds only the ddev list items that
// have names starting with "Test"
func getTestingSitesFromList(t *testing.T, jsonOut string) []interface{} {
	assert := asrt.New(t)

	baseRaw := getSitesFromList(t, jsonOut)
	testSites := make([]interface{}, 0)
	for _, listItem := range baseRaw {
		item, ok := listItem.(map[string]interface{})
		assert.True(ok)

		if strings.HasPrefix(item["name"].(string), "Test") {
			testSites = append(testSites, listItem)
		}
	}
	return testSites
}

// TestCmdListContinuous tests the --continuous flag for ddev list.
func TestCmdListContinuous(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping TestCmdListContinuous because Windows stdout capture doesn't work.")
	}

	assert := asrt.New(t)

	t.Setenv("DDEV_DEBUG", "")
	// Execute "ddev list --continuous"
	cmd := oexec.Command(DdevBin, "list", "-j", "--continuous")
	stdout, err := cmd.StdoutPipe()
	assert.NoError(err)

	err = cmd.Start()
	assert.NoError(err)

	reader := bufio.NewReader(stdout)

	blob, err := reader.ReadBytes('\n')
	assert.NoError(err)
	byteCount := len(blob)
	blob = blob[:byteCount-1]
	require.True(t, byteCount > 300, "byteCount should have been >300 and was %v blob=%s", byteCount, string(blob))

	f, err := unmarshalJSONLogs(string(blob))
	require.NoError(t, err, "Could not unmarshal blob, err=%v, content=%s", err, blob)

	assert.NotEmpty(f[0]["raw"])
	time1 := f[0]["time"]
	if len(f) > 1 {
		t.Logf("Expected one line in initial read, but got %d lines instead", len(f))
	}

	time.Sleep(time.Millisecond * 1500)

	// Now read more from the pipe after resetting blob
	blob, err = reader.ReadBytes('\n')
	byteCount = len(blob)
	assert.NoError(err)
	blob = blob[:byteCount-1]
	require.True(t, byteCount > 300, "byteCount should have been >300 and was %v blob=%s", byteCount, string(blob))
	f, err = unmarshalJSONLogs(string(blob))
	require.NoError(t, err, "Could not unmarshal blob, err=%v, content=%s", err, blob)
	if len(f) > 1 {
		t.Logf("Expected one line in second read, but got %d lines instead", len(f))
	}
	assert.NotEmpty(f[0]["raw"]) // Just to make sure it's a list object
	time2 := f[0]["time"]
	assert.NotEqual(time1, time2)
	assert.True(time2.(string) > time1.(string))

	// Kill the process we started.
	err = cmd.Process.Kill()
	assert.NoError(err)
}
