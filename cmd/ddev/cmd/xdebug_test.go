package cmd

import (
	"testing"

	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
)

// TestCmdXdebug tests the ddev xdebug command
func TestCmdXdebug(t *testing.T) {
	assert := asrt.New(t)

	v := TestSites[0]

	cleanup := testcommon.Chdir(v.Dir)
	_, err := exec.RunCommand(DdevBin, []string{"start"})
	assert.NoError(err)

	out, err := exec.RunCommand(DdevBin, []string{"xdebug", "status"})
	assert.NoError(err)
	assert.Contains(string(out), "xdebug disabled")

	out, err = exec.RunCommand(DdevBin, []string{"xdebug", "on"})
	assert.NoError(err)
	assert.Contains(string(out), "Enabled xdebug")

	out, err = exec.RunCommand(DdevBin, []string{"xdebug", "status"})
	assert.NoError(err)
	assert.Contains(string(out), "xdebug enabled")

	out, err = exec.RunCommand(DdevBin, []string{"xdebug", "off"})
	assert.NoError(err)
	assert.Contains(string(out), "Disabled xdebug")

	out, err = exec.RunCommand(DdevBin, []string{"xdebug", "status"})
	assert.NoError(err)
	assert.Contains(string(out), "xdebug disabled")

	cleanup()
}
