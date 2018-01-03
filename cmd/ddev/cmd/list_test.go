package cmd

import (
	"bytes"
	"runtime"
	"testing"

	"encoding/json"
	oexec "os/exec"
	"time"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	log "github.com/sirupsen/logrus"
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

	// Unmarshall the json results. The list function has 4 fields to output
	data := make(log.Fields, 4)
	err = json.Unmarshal([]byte(jsonOut), &data)
	assert.NoError(err)
	raw, ok := data["raw"].([]interface{})
	assert.True(ok)

	for _, v := range DevTestSites {
		app, err := ddevapp.GetActiveApp(v.Name)
		assert.NoError(err)

		// Look for standard items in the regular ddev list output
		assert.Contains(string(out), v.Name)
		assert.Contains(string(out), app.GetURL())
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
		assert.True(found, "Failed to find site %s in ddev list -j", v.Name)

	}

}

// TestDdevListContinuous tests the --continuous flag for ddev list.
func TestDdevListContinuous(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
		return
	}

	assert := asrt.New(t)

	// Execute "ddev list --continuous"
	cmd := oexec.Command(DdevBin, "list", "--continuous")
	var cmdOutput bytes.Buffer
	cmd.Stdout = &cmdOutput
	err := cmd.Start()
	assert.NoError(err)

	// Take a snapshot of the output a little over one second apart.
	output1 := len(cmdOutput.Bytes())
	time.Sleep(time.Millisecond * 1020)
	output2 := len(cmdOutput.Bytes())

	// Kill the process we started.
	err = cmd.Process.Kill()
	assert.NoError(err)

	// The two snapshots of output should be different, and output2 should be larger.
	assert.NotEqual(output1, output2)
	assert.True((output2 > output1))
}
