package cmd

import (
	"testing"

	utils "github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

func TestLegacyList(t *testing.T) {
	args := []string{"legacy", "list"}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(t, err)
	assert.Contains(t, string(out), "containers found")
	assert.Contains(t, string(out), LegacyTestApp)
	assert.Contains(t, string(out), LegacyTestEnv)

}
