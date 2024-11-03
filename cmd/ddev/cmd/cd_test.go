package cmd

import (
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"path/filepath"
	"strings"
	"testing"
)

// TestCdCmd runs `ddev cd` to see if it works.
func TestCdCmd(t *testing.T) {
	assert := asrt.New(t)
	// Shows help
	out, err := exec.RunHostCommand(DdevBin, "cd", TestSites[0].Name)
	assert.NoError(err)
	assert.Contains(out, filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/host/shells/ddev.sh"))
	assert.Contains(out, filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/host/shells/ddev.fish"))
	// Returns the path to the project
	out, err = exec.RunHostCommand(DdevBin, "cd", TestSites[0].Name, "--get-approot")
	assert.NoError(err)
	assert.Equal(strings.TrimRight(out, "\n"), TestSites[0].Dir)
	// Shows error
	out, err = exec.RunHostCommand(DdevBin, "cd", "does-not-exist-"+util.RandString(4))
	assert.Error(err)
	assert.Contains(out, "Failed to find path for project")
	// Shows error
	out, err = exec.RunHostCommand(DdevBin, "cd")
	assert.Error(err)
	assert.Contains(out, "This command only takes one argument: project-name")
}
