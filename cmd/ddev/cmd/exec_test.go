package cmd

import (
	"testing"

	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

// TestDevExecBadArgs run `ddev exec` without the proper args
func TestDevExecBadArgs(t *testing.T) {
	// Change to the first DevTestSite for the duration of this test.
	defer DevTestSites[0].Chdir()()
	assert := assert.New(t)

	args := []string{"exec"}
	out, err := system.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Invalid arguments detected.")

	// Try with an invalid number of args
	args = []string{"exec", "RandomValue", "pwd"}
	out, err = system.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Invalid arguments detected")
}

// TestDevExec run `ddev exec pwd` with proper args
func TestDevExec(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	assert := assert.New(t)
	for _, v := range DevTestSites {
		cleanup := v.Chdir()

		args := []string{"exec", "pwd"}
		out, err := system.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.Contains(string(out), "/var/www/html/docroot")

		cleanup()
	}
}

// @TODO: These are still valid tests, but we should only be doing them after an import.

// TestDevExec runs drud Dev exec using basic drush commands
func TestDevExecDrush(t *testing.T) {
	/**if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	d8App := DevTestSites[1][0]
	d7App := DevTestSites[2][0]
	assert := assert.New(t)

	for _, app := range []string{d8App, d7App} {
		args := []string{"exec", app, DevTestEnv, "drush uli"}
		out, err := system.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.Contains(string(out), "http://")

		// Try again with active app set.
		err = setActiveApp(DevTestSites[1][0], DevTestEnv)
		assert.NoError(err)
		args = []string{"exec", app, DevTestEnv, "drush uli"}
		out, err = system.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.Contains(string(out), "http://")

		args = []string{"exec", app, DevTestEnv, "drush status"}
		out, err = system.RunCommand(DdevBin, args)
		assert.NoError(err)
		// Check for database status
		assert.Contains(string(out), "Connected")
		// Check for PHP configuration
		assert.Contains(string(out), "/etc/php/7.0/cli/php.ini")
		// Check for drush version
		assert.Contains(string(out), "/etc/php/7.0/cli/php.ini", "8.1.8")
	}
	**/
}

// TestDevExec run for drud Dev exec using the wp-cli
func TestDevExecWpCLI(t *testing.T) {
	/**
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	wpApp := DevTestSites[0][0]

	// Run an exec by passing in TestApp + TestEnv
	assert := assert.New(t)

	args := []string{"exec", wpApp, DevTestEnv, "wp --info"}
	out, err := system.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "/etc/php/7.0/cli/php.ini")

	args = []string{"exec", wpApp, DevTestEnv, "wp plugin status"}
	out, err = system.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "installed plugins")
	**/
}
