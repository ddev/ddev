package cmd

import (
	"bytes"
	"runtime"
	"testing"

	oexec "os/exec"
	"time"

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

	logItems, err := unmarshallJSONLogs(jsonOut)
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

// TestDdevListContinuous tests the --continuous flag for ddev list.
func TestDdevListContinuous(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping TestDdevListContinuous because Windows stdout capture doesn't work.")
	}

	assert := asrt.New(t)

	// Execute "ddev list --continuous"
	cmd := oexec.Command(DdevBin, "list", "--continuous")
	var cmdOutput bytes.Buffer
	cmd.Stdout = &cmdOutput
	err := cmd.Start()
	assert.NoError(err)

	// Take a snapshot of the output a little over one second apart.
	output1 := string(cmdOutput.Bytes())
	time.Sleep(time.Millisecond * 1500)
	output2 := string(cmdOutput.Bytes())

	// Kill the process we started.
	err = cmd.Process.Kill()
	assert.NoError(err)

	// The two snapshots of output should be different, and output2 should be larger.
	assert.NotEqual(output1, output2, "Outputs at 2 times should have been different. output1=\n===\n%s\n===\noutput2=\n===\n%s\n===\n", output1, output2)
	assert.True((len(output2) > len(output1)))
}
