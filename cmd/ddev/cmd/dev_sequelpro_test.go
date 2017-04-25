package cmd

import (
	"testing"

	"path"

	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

// TestSequelproOperation tests basic operation.
func TestSequelproOperation(t *testing.T) {
	if !detectSequelpro() {
		t.SkipNow()
	}
	assert := assert.New(t)
	v := DevTestSites[0]
	cleanup := v.Chdir()

	_, err := getActiveApp()
	assert.NoError(err)

	out, err := handleSequelProCommand(SequelproLoc)
	assert.NoError(err)
	assert.Contains(string(out), "sequelpro command finished successfully")

	dir, err := getActiveAppRoot()
	assert.Equal(true, system.FileExists(path.Join(dir, ".ddev/sequelpro.spf")))

	cleanup()
}

// TestSequelproBadArgs tests non-site operation and bad args
func TestSequelproBadArgs(t *testing.T) {
	if !detectSequelpro() {
		t.SkipNow()
	}

	assert := assert.New(t)

	// Create a temporary directory and switch to it for the duration of this test.
	tmpdir := testcommon.CreateTmpDir("sequelpro_badargs")
	defer testcommon.Chdir(tmpdir)()
	defer testcommon.CleanupDir(tmpdir)

	// Ensure it fails if we run outside of an application root.
	out, err := handleSequelProCommand(SequelproLoc)
	assert.Error(err)
	assert.Contains(string(out), "unable to determine the application for this command")

	// Ensure we get a failure if using arguments
	out, err = handleSequelProCommand(SequelproLoc, testcommon.RandString(16))
	assert.Error(err)
	assert.Contains(err, "Invalid arguments")
}
