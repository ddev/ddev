package cmd

import (
	"testing"

	"github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

// TestLegacyExecBadArgs run `drud legacy exec` without the proper args
func TestLegacyExecBadArgs(t *testing.T) {
	assert := assert.New(t)
	args := []string{"dev", "exec", LegacyTestApp, LegacyTestEnv}
	out, err := utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Invalid arguments detected.")

	args = []string{"dev", "exec", "pwd"}
	out, err = utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "app_name and deploy_name are expected as arguments")

	// Try with an invalid number of args
	args = []string{"dev", "exec", LegacyTestApp, "pwd"}
	out, err = utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Invalid arguments detected")
}

// TestLegacyExec run `drud legacy exec pwd` with proper args
func TestLegacyExec(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}

	// Run an exec by passing in TestApp + TestEnv
	assert := assert.New(t)

	args := []string{"dev", "exec", LegacyTestApp, LegacyTestEnv, "pwd"}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "/var/www/html/docroot")

	// Try again with active app set.
	err = setActiveApp(LegacyTestApp, LegacyTestEnv)
	assert.NoError(err)
	args = []string{"dev", "exec", LegacyTestApp, LegacyTestEnv, "pwd"}
	out, err = utils.RunCommand(DrudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "/var/www/html/docroot")
}

// TestLegacyExec runs drud legacy exec using basic drush commands
func TestLegacyExecDrush(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	d8App := LegacyTestSites[1][0]
	d7App := LegacyTestSites[2][0]
	assert := assert.New(t)

	for _, app := range []string{d8App, d7App} {
		args := []string{"dev", "exec", app, LegacyTestEnv, "drush uli"}
		out, err := utils.RunCommand(DrudBin, args)
		assert.NoError(err)
		assert.Contains(string(out), "http://")

		// Try again with active app set.
		err = setActiveApp(LegacyTestSites[1][0], LegacyTestEnv)
		assert.NoError(err)
		args = []string{"dev", "exec", app, LegacyTestEnv, "drush uli"}
		out, err = utils.RunCommand(DrudBin, args)
		assert.NoError(err)
		assert.Contains(string(out), "http://")

		args = []string{"dev", "exec", app, LegacyTestEnv, "drush status"}
		out, err = utils.RunCommand(DrudBin, args)
		assert.NoError(err)
		// Check for database status
		assert.Contains(string(out), "Connected")
		// Check for PHP configuration
		assert.Contains(string(out), "/etc/php/7.0/cli/php.ini")
		// Check for drush version
		assert.Contains(string(out), "/etc/php/7.0/cli/php.ini", "8.1.8")
	}
}

// TestLegacyExec run for drud legacy exec using the wp-cli
func TestLegacyExecWpCLI(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	wpApp := LegacyTestSites[0][0]

	// Run an exec by passing in TestApp + TestEnv
	assert := assert.New(t)

	args := []string{"dev", "exec", wpApp, LegacyTestEnv, "wp --info"}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "/etc/php/7.0/cli/php.ini")

	args = []string{"dev", "exec", wpApp, LegacyTestEnv, "wp plugin status"}
	out, err = utils.RunCommand(DrudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "riot-autoloader")
	assert.Contains(string(out), "drudio-core")
}
