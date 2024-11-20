package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
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
	bashScript := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/host/shells/ddev.sh")
	zshScript := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/host/shells/ddev.zsh")
	fishScript := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/host/shells/ddev.fish")
	// Shows help
	out, err := exec.RunHostCommand(DdevBin, "cd", TestSites[0].Name)
	assert.NoError(err)
	assert.Contains(out, bashScript)
	assert.Contains(out, zshScript)
	assert.Contains(out, fishScript)
	// Shows help
	out, err = exec.RunHostCommand(DdevBin, "cd", "does-not-exist-"+util.RandString(4))
	assert.NoError(err)
	assert.Contains(out, bashScript)
	assert.Contains(out, zshScript)
	assert.Contains(out, fishScript)
	// Shows help
	out, err = exec.RunHostCommand(DdevBin, "cd")
	assert.NoError(err)
	assert.Contains(out, bashScript)
	assert.Contains(out, zshScript)
	assert.Contains(out, fishScript)
	// Returns the path to the project
	out, err = exec.RunHostCommand(DdevBin, "cd", TestSites[0].Name, "--get-approot")
	assert.NoError(err)
	assert.Equal(strings.TrimRight(out, "\n"), TestSites[0].Dir)
	// Returns list of projects for autocompletion
	out, err = exec.RunHostCommand(DdevBin, "cd", "--list")
	assert.NoError(err)
	projects, err := ddevapp.GetProjects(false)
	assert.NoError(err)
	for _, project := range projects {
		assert.Contains(out, project.Name)
	}
}
