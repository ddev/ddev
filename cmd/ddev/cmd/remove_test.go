package cmd

import (
	"fmt"
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

		// Note that the "ddev remove -y" case use used in the TestMain, so that should cover that option.

		cmd := fmt.Sprintf("echo n | %s remove", DdevBin)
		out, err := exec.RunCommand("sh", []string{"-c", cmd})
		assert.Error(err, "ddev remove should fail and instead succeeded ('n' response to prompt)")
		assert.Contains(out, "App removal canceled")

		cmd = fmt.Sprintf("echo so silly to expect this to work | %s remove", DdevBin)
		out, err = exec.RunCommand("sh", []string{"-c", cmd})
		assert.Error(err, "ddev remove should fail and instead succeeded (silly response to prompt)")
		assert.Contains(out, "App removal canceled")

		cmd = fmt.Sprintf("echo y | %s remove", DdevBin)
		out, err = exec.RunCommand("sh", []string{"-c", cmd})
		assert.NoError(err, "ddev remove should succeed but failed, err: %v, output: %s", err, out)
		assert.Contains(out, "Successfully removed")

		cleanup()
	}
	// Now put the sites back together so other tests can use them.
	addSites()
}
