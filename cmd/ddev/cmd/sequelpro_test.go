package cmd

import (
	"testing"

	"path/filepath"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
)

// TestSequelproOperation tests basic operation.
func TestSequelproOperation(t *testing.T) {
	if !detectSequelpro() {
		t.SkipNow()
	}
	assert := asrt.New(t)
	v := TestSites[0]
	cleanup := v.Chdir()

	_, err := ddevapp.GetActiveApp("")
	assert.NoError(err)

	out, err := handleSequelProCommand(SequelproLoc)
	assert.NoError(err)
	assert.Contains(string(out), "sequelpro command finished successfully")

	dir, err := ddevapp.GetActiveAppRoot("")
	assert.NoError(err)
	assert.Equal(true, fileutil.FileExists(filepath.Join(dir, ".ddev/sequelpro.spf")))

	cleanup()
}

// TestSequelproBadApp tests non-site operation and bad args
func TestSequelproBadApp(t *testing.T) {
	if !detectSequelpro() {
		t.SkipNow()
	}

	assert := asrt.New(t)

	// Create a temporary directory and switch to it for the duration of this test.
	tmpdir := testcommon.CreateTmpDir("sequelpro_badargs")
	defer testcommon.Chdir(tmpdir)()
	defer testcommon.CleanupDir(tmpdir)

	// Ensure it fails if we run outside of an application root.
	_, err := handleSequelProCommand(SequelproLoc)
	assert.Error(err)
	assert.Contains(err.Error(), "Could not find a project in")
}
