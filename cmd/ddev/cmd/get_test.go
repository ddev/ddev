package cmd

import (
	"github.com/drud/ddev/pkg/exec"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"

	asrt "github.com/stretchr/testify/assert"
)

// TestCmdGet tests various `ddev get` commands .
func TestCmdGet(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err = exec.RunHostCommand(DdevBin, "service", "disable", "memcached")
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	tarballFile := filepath.Join(origDir, "testdata", t.Name(), "ddev-memcached-1.0.1.tar.gz")
	for _, arg := range []string{
		"drud/ddev-memcached",
		"https://github.com/drud/ddev-memcached/archive/refs/tags/v1.0.1.tar.gz",
		tarballFile} {
		out, err := exec.RunHostCommand(DdevBin, "get", arg)
		assert.NoError(err)
		assert.Contains(out, "Downloaded add-on")
	}
}
