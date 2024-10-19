package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
)

// TestDebugNFSMount tries out the `ddev debug nfsmount` command.
// It requires nfsd running of course.
func TestDebugNFSMount(t *testing.T) {
	t.Skip("Skipping because NFS is deprecated")
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
	defer exec.RunCommand(DdevBin, []string{"stop", "-RO"})

	// Test basic `ddev debug nfsmount`
	args = []string{"debug", "nfsmount"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(out, "Successfully accessed NFS")
	assert.Contains(out, "/nfsmount")
	pwd, err := os.Getwd()
	assert.NoError(err)

	source := pwd
	if runtime.GOOS == "darwin" && fileutil.IsDirectory(filepath.Join("/System/Volumes/Data", source)) {
		source = filepath.Join("/System/Volumes/Data", source)
	}
	assert.Contains(out, ":"+dockerutil.MassageWindowsNFSMount(source))
}
