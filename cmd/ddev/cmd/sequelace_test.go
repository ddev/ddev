package cmd

import (
	"testing"

	"path/filepath"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
)

// TestSequelAceOperation tests basic operation.
func TestSequelAceOperation(t *testing.T) {
	if !detectSequelAce() {
		t.SkipNow()
	}
	assert := asrt.New(t)
	v := TestSites[0]
	cleanup := v.Chdir()

	_, err := ddevapp.GetActiveApp("")
	assert.NoError(err)

	out, err := handleSequelAceCommand(SequelAceLoc)
	assert.NoError(err)
	assert.Contains(string(out), "sequelace command finished successfully")

	dir, err := ddevapp.GetActiveAppRoot("")
	assert.NoError(err)
	assert.Equal(true, fileutil.FileExists(filepath.Join(dir, ".ddev/sequelace.spf")))

	cleanup()
}

// TestSequelAceBadApp tests non-site operation and bad args
func TestSequelAceBadApp(t *testing.T) {
	if !detectSequelAce() {
		t.SkipNow()
	}

	assert := asrt.New(t)

	// Create a temporary directory and switch to it for the duration of this test.
	tmpdir := testcommon.CreateTmpDir("sequelace_badargs")
	defer testcommon.Chdir(tmpdir)()
	defer testcommon.CleanupDir(tmpdir)

	// Ensure it fails if we run outside of an application root.
	_, err := handleSequelAceCommand(SequelAceLoc)
	assert.Error(err)
	assert.Contains(err.Error(), "Could not find a project in")
}
