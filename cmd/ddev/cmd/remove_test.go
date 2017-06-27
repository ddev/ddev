package cmd

import (
	"testing"

	"github.com/drud/ddev/pkg/exec"
	"github.com/stretchr/testify/assert"
)

// TestDevRestart runs `drud legacy restart` on the test apps
func TestDevRemove(t *testing.T) {
	assert := assert.New(t)

	// Make sure we have running sites.
	addSites()

	for _, site := range DevTestSites {
		cleanup := site.Chdir()
		out, err := exec.RunCommand("ddev", []string{"remove"})
		assert.NoError(err, "ddev remove should succeed but failed, err: %v, output: %s", err, out)
		assert.Contains(out, "Successfully removed")

		cleanup()
	}
	// Now put the sites back together so other tests can use them.
	addSites()
}
