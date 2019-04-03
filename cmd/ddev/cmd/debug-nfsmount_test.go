package cmd

import (
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"os"
	"testing"
)

// TestDebugNFSMount tries out the `ddev debug nfsmount` command.
// It requires nfsd running of course.
func TestDebugNFSMount(t *testing.T) {
	assert := asrt.New(t)

	oldDir, err := os.Getwd()
	assert.NoError(err)
	// nolint: errcheck
	defer os.Chdir(oldDir)

	tmpDir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(tmpDir)
	defer testcommon.Chdir(tmpDir)()

	err = os.Chdir(tmpDir)
	assert.NoError(err)

	args := []string{"config", "--project-type", "php"}
	_, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)

	// Running config creates a line in global config
	//nolint: errcheck
	defer exec.RunCommand("remove", []string{"--remove-data"})

	// Test basic `ddev debug nfsmount`
	args = []string{"debug", "nfsmount"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(out, "Successfully accessed NFS")
	assert.Contains(out, "/nfsmount")
	pwd, err := os.Getwd()
	assert.NoError(err)
	assert.Contains(out, ":"+dockerutil.MassageWIndowsNFSMount(pwd))
}
