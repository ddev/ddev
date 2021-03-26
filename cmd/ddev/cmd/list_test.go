package cmd

import (
	"bufio"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/stretchr/testify/require"
	"runtime"
	"strings"
	"testing"
	"time"

	oexec "os/exec"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
)

// TestCmdList runs the binary with "ddev list" and checks the results
func TestCmdList(t *testing.T) {
	assert := asrt.New(t)

	// This gratuitous ddev start -a repopulates the ~/.ddev/global_config.yaml
	// project list, which has been damaged by other tests which use
	// direct app techniques.
	_, err := exec.RunCommand(DdevBin, []string{"start", "-a", "-y"})
	assert.NoError(err)

	// Execute "ddev list" and harvest plain text output.
	out, err := exec.RunCommand(DdevBin, []string{"list"})
	assert.NoError(err, "error runnning ddev list: %v output=%s", out)

	// Execute "ddev list -j" and harvest the json output
	jsonOut, err := exec.RunCommand(DdevBin, []string{"list", "-j"})
	assert.NoError(err, "error running ddev list -j: %v, output=%s", jsonOut)

	siteList := getTestingSitesFromList(t, jsonOut)
	assert.Equal(len(TestSites), len(siteList))

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
		assert.Contains(string(out), ddevapp.RenderHomeRootedDir(app.GetAppRoot()))

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
				assert.EqualValues(ddevapp.RenderHomeRootedDir(app.GetAppRoot()), item["shortroot"])
				assert.EqualValues(app.GetAppRoot(), item["approot"])
				break
			}
		}
		assert.True(found, "Failed to find project %s in ddev list -j", v.Name)

	}

	// Stop the first app
	out, err = exec.RunCommand(DdevBin, []string{"stop", TestSites[0].Name})
	t.Logf("Stopped first project with ddev stop %s", TestSites[0].Name)
	assert.NoError(err, "error runnning ddev stop %v: %v output=%s", TestSites[0].Name, err, out)

	// Execute "ddev list" and harvest json output.
	// Now there should be one less active project in list
	jsonOut, err = exec.RunCommand(DdevBin, []string{"list", "-jA"})
	assert.NoError(err, "error runnning ddev list: %v output=%s", err, jsonOut)

	siteList = getTestingSitesFromList(t, jsonOut)
	assert.Equal(len(TestSites)-1, len(siteList))
	t.Logf("test projects active with ddev list -jA: %v", siteList)

	// Now list without -A, make sure we show all projects
	jsonOut, err = exec.RunCommand(DdevBin, []string{"list", "-j"})
	assert.NoError(err, "error runnning ddev list: %v output=%s", err, out)
	siteList = getTestingSitesFromList(t, jsonOut)
	assert.Equal(len(TestSites), len(siteList))
	t.Logf("test projects (including inactive) shown with ddev list -j: %v", siteList)

	// Leave firstApp running for other tests
	out, err = exec.RunCommand(DdevBin, []string{"start", "-y", TestSites[0].Name})
	assert.NoError(err, "error runnning ddev start: %v output=%s", err, out)
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
	assert.True(ok)
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

	// Execute "ddev list --continuous"
	cmd := oexec.Command(DdevBin, "list", "-j", "--continuous")
	stdout, err := cmd.StdoutPipe()
	assert.NoError(err)

	err = cmd.Start()
	assert.NoError(err)

	reader := bufio.NewReader(stdout)

	blob := make([]byte, 16000)
	byteCount, err := reader.Read(blob)
	assert.NoError(err)
	blob = blob[:byteCount-1]
	require.True(t, byteCount > 300, "byteCount should have been >300 and was %v", byteCount)

	f, err := unmarshalJSONLogs(string(blob))
	if err != nil {
		assert.NoError(err, "could not unmarshal ddev output: %v", err)
		t.Logf("============== ddev list -j --continuous failed logs =================\n%s\n", string(blob))
		t.FailNow()
	}
	assert.NotEmpty(f[0]["raw"])
	time1 := f[0]["time"]
	if len(f) > 1 {
		t.Logf("Expected just one line in initial read, but got %d lines instead", len(f))
	}

	time.Sleep(time.Millisecond * 1500)

	// Now read more from the pipe after resetting blob
	blob = make([]byte, 16000)
	byteCount, err = reader.Read(blob)
	assert.NoError(err)
	blob = blob[:byteCount-1]
	require.True(t, byteCount > 300)
	f, err = unmarshalJSONLogs(string(blob))
	require.NoError(t, err)
	if len(f) > 1 {
		t.Logf("Expected just one line in second read, but got %d lines instead", len(f))
	}
	assert.NotEmpty(f[0]["raw"]) // Just to make sure it's a list object
	time2 := f[0]["time"]
	assert.NotEqual(time1, time2)
	assert.True(time2.(string) > time1.(string))

	// Kill the process we started.
	err = cmd.Process.Kill()
	assert.NoError(err)

}
