package cmd

import (
	"testing"

	"encoding/json"

	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/plugins/platform"
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
		app, err := platform.GetActiveApp(v.Name)
		assert.NoError(err)

		// Look for standard items in the regular ddev list output
		assert.Contains(string(out), v.Name)
		assert.Contains(string(out), app.URL())
		assert.Contains(string(out), app.GetType())
		assert.Contains(string(out), platform.RenderHomeRootedDir(app.AppRoot()))

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
				assert.EqualValues(platform.RenderHomeRootedDir(app.AppRoot()), item["shortroot"])
				assert.EqualValues(app.AppRoot(), item["approot"])
				break
			}
		}
		assert.True(found, "Failed to find site %s in ddev list -j", v.Name)

	}

}
