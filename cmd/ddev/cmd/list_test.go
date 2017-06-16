package cmd

import (
	"testing"

	"github.com/drud/ddev/pkg/exec"
	"github.com/stretchr/testify/assert"
)

func TestDevList(t *testing.T) {
	assert := assert.New(t)
	args := []string{"list"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	for _, v := range DevTestSites {
		cleanup := v.Chdir()

		app, err := platform.GetActiveApp("")
		if err != nil {
			assert.Fail("Could not find an active ddev configuration: %v", err)
		}
		assert.Contains(string(out), v.Name)
		assert.Contains(string(out), app.URL())
		assert.Contains(string(out), app.GetType())
		assert.Contains(string(out), app.AppRoot())
		cleanup()
	}

}
