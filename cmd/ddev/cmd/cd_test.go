package cmd

import (
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	asrt "github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

// TestCdCmd runs `ddev cd` to see if it works.
func TestCdCmd(t *testing.T) {
	assert := asrt.New(t)
	out, err := exec.RunHostCommand(DdevBin, "cd")
	assert.NoError(err)
	assert.Contains(out, filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/host/shells/ddev.sh"))
	assert.Contains(out, filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/host/shells/ddev.fish"))
}
