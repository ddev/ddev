package cmd

import (
	"bufio"
	"github.com/stretchr/testify/require"
	"runtime"
	"testing"
	"time"

	oexec "os/exec"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
)

// TestDevList runs the binary with "ddev list" and checks the results
func TestDevList(t *testing.T) {
	assert := asrt.New(t)

	// Execute "ddev list" and harvest plain text output.
	args := []string{"list"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)

	// Execute "ddev list -j" and harvest the json output
	args = []string{"list", "-j"}
	jsonOut, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)

	logItems, err := unmarshalJSONLogs(jsonOut)
	assert.NoError(err)

	// The list should be the last item; there may be a warning
	// or other info before that.
	data := logItems[len(logItems)-1]
	assert.EqualValues(data["level"], "info")

	raw, ok := data["raw"].([]interface{})
	assert.True(ok)

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
		for _, listitem := range raw {
			_ = listitem
			item, ok := listitem.(map[string]interface{})
			assert.True(ok)
			// Check to see that we can find our item
			if item["name"] == v.Name {
				found = true
				assert.Contains(item["httpurl"], app.HostName())
				assert.Contains(item["httpsurl"], app.HostName())
				assert.EqualValues(app.GetType(), item["type"])
				assert.EqualValues(ddevapp.RenderHomeRootedDir(app.GetAppRoot()), item["shortroot"])
				assert.EqualValues(app.GetAppRoot(), item["approot"])
				break
			}
		}
		assert.True(found, "Failed to find project %s in ddev list -j", v.Name)

	}

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

	blob := make([]byte, 4096)
	byteCount, err := reader.Read(blob)
	assert.NoError(err)
	blob = blob[:byteCount-1]
	require.True(t, byteCount > 1000)

	f, err := unmarshalJSONLogs(string(blob))
	require.NoError(t, err)
	assert.NotEmpty(f[0]["raw"])
	time1 := f[0]["time"]
	if len(f) > 1 {
		t.Logf("Expected just one line in initial read, but got %d lines instead", len(f))
	}

	time.Sleep(time.Millisecond * 1500)

	// Now read more from the pipe after resetting blob
	blob = make([]byte, 4096)
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
