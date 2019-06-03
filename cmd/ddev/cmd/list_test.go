package cmd

import (
	"bufio"
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

	// Execute "ddev list" and harvest plain text output.
	out, err := exec.RunCommand(DdevBin, []string{"list"})
	assert.NoError(err, "error runnning ddev list: %v output=%s", out)

	// Execute "ddev list -j" and harvest the json output
	jsonOut, err := exec.RunCommand(DdevBin, []string{"list", "-j"})
	assert.NoError(err, "error running ddev list -j: %v, output=%s", jsonOut)

	siteList := getTestingSitesFromList(t, jsonOut)
	assert.Equal(len(DevTestSites), len(siteList))

	for _, v := range DevTestSites {
		app, err := ddevapp.GetActiveApp(v.Name)
		assert.NoError(err)

		// Look for standard items in the regular ddev list output
		assert.Contains(string(out), v.Name)
		assert.Contains(string(out), app.GetHTTPURL())
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
	firstApp, err := ddevapp.GetActiveApp(DevTestSites[0].Name)
	assert.NoError(err)
	err = firstApp.Stop(false, false)
	assert.NoError(err)

	// Execute "ddev list" and harvest plain text output.
	// Now there should be one less project in list
	jsonOut, err = exec.RunCommand(DdevBin, []string{"list", "-jA"})
	assert.NoError(err, "error runnning ddev list: %v output=%s", out)

	siteList = getTestingSitesFromList(t, jsonOut)
	assert.Equal(len(DevTestSites)-1, len(siteList))

	// Now list without -A, make sure we show all projects
	jsonOut, err = exec.RunCommand(DdevBin, []string{"list", "-j"})
	assert.NoError(err, "error runnning ddev list: %v output=%s", out)

	siteList = getTestingSitesFromList(t, jsonOut)
	assert.Equal(len(DevTestSites), len(siteList))

	// Leave firstApp running for other tests
	err = firstApp.Start()
	assert.NoError(err)
}

// getSitesFromList takes the json output of ddev list -j
// and returns the list of *test* sites ddev list returns as an array
// of interface{}
func getSitesFromList(t *testing.T, jsonOut string) []interface{} {
	assert := asrt.New(t)

	logItems, err := unmarshalJSONLogs(jsonOut)
	assert.NoError(err)
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

	blob := make([]byte, 8192)
	byteCount, err := reader.Read(blob)
	assert.NoError(err)
	blob = blob[:byteCount-1]
	require.True(t, byteCount > 1000)

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
	blob = make([]byte, 8192)
	byteCount, err = reader.Read(blob)
	assert.NoError(err)
	blob = blob[:byteCount-1]
	require.True(t, byteCount > 1000)
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
