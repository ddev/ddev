package cmd

import (
	"testing"

	drudutils "github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

func TestLegacyList(t *testing.T) {
	args := []string{"legacy", "list"}
	out, err := drudutils.RunCommand(drudBin, args)
	assert.NoError(t, err)
	assert.Contains(t, string(out), "containers found")
	assert.Contains(t, string(out), legacyTestApp)
	assert.Contains(t, string(out), legacyTestEnv)

}
