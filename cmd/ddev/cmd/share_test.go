package cmd

import (
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestShareCmd(t *testing.T) {
	assert := asrt.New(t)

	oldDir, err := os.Getwd()
	assert.NoError(err)
	// nolint: errcheck
	defer os.Chdir(oldDir)

	tmpDir := testcommon.CreateTmpDir(t.Name())
	err = os.Chdir(tmpDir)
	assert.NoError(err)

}
