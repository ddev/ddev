package ddevapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

// TestWriteTypo3SettingsFileDBConfig verifies that the generated TYPO3
// additional.php only contains a database connection block when the db
// container exists, and that the block is guarded at runtime so it can't
// override a project that has been configured for SQLite (or any other
// driver the db container doesn't provide). See
// https://github.com/ddev/ddev/pull/8582 for background.
func TestWriteTypo3SettingsFileDBConfig(t *testing.T) {
	writeSettings := func(t *testing.T, app *DdevApp) string {
		app.SiteDdevSettingsFile = filepath.Join(t.TempDir(), "config", "system", "additional.php")
		require.NoError(t, writeTypo3SettingsFile(app))
		content, err := os.ReadFile(app.SiteDdevSettingsFile)
		require.NoError(t, err)
		return string(content)
	}

	t.Run("db container present", func(t *testing.T) {
		app := &DdevApp{Name: t.Name(), Type: nodeps.AppTypeTYPO3}
		content := writeSettings(t, app)

		require.Contains(t, content, `'host' => 'db'`)
		require.Contains(t, content, `'driver' => 'mysqli'`)
		require.Contains(t, content, `'port' => 3306`)
		// The non-DB settings must be present too.
		require.Contains(t, content, `'GFX'`)
		require.Contains(t, content, `'MAIL'`)

		// The DB connection must only be applied inside the runtime driver
		// guard, so a settings.php configured for pdo_sqlite is left alone
		// on every restart.
		guardPos := strings.Index(content, `in_array($ddevDriver, ['mysqli', 'pdo_mysql', 'pdo_pgsql'], true)`)
		dbBlockPos := strings.Index(content, `$ddevConfig['DB']`)
		require.NotEqual(t, -1, guardPos, "runtime driver guard not found in generated additional.php")
		require.NotEqual(t, -1, dbBlockPos, "DB connection block not found in generated additional.php")
		require.Less(t, guardPos, dbBlockPos, "DB connection block must be inside the runtime driver guard")
		require.Contains(t, content, `$GLOBALS['TYPO3_CONF_VARS']['DB']['Connections']['Default']['driver'] ??`)
	})

	t.Run("postgres db container", func(t *testing.T) {
		app := &DdevApp{Name: t.Name(), Type: nodeps.AppTypeTYPO3}
		app.Database.Type = nodeps.Postgres
		content := writeSettings(t, app)

		require.Contains(t, content, `'driver' => 'pdo_pgsql'`)
		require.Contains(t, content, `'port' => 5432`)
	})

	t.Run("db container omitted in project config", func(t *testing.T) {
		app := &DdevApp{Name: t.Name(), Type: nodeps.AppTypeTYPO3, OmitContainers: []string{"db"}}
		content := writeSettings(t, app)

		require.NotContains(t, content, `'host' => 'db'`)
		require.NotContains(t, content, `'Connections'`)
		require.NotContains(t, content, `$ddevDriver`)
		// The non-DB settings must survive.
		require.Contains(t, content, `'GFX'`)
		require.Contains(t, content, `'MAIL'`)
		require.Contains(t, content, `array_replace_recursive`)
	})

	t.Run("db container omitted in global config", func(t *testing.T) {
		app := &DdevApp{Name: t.Name(), Type: nodeps.AppTypeTYPO3, OmitContainersGlobal: []string{"db"}}
		content := writeSettings(t, app)

		require.NotContains(t, content, `'host' => 'db'`)
		require.Contains(t, content, `'GFX'`)
	})
}

// TestWriteDrupalSettingsDdevPhpDBConfig verifies that settings.ddev.php
// only contains database connection settings when the db container exists.
func TestWriteDrupalSettingsDdevPhpDBConfig(t *testing.T) {
	writeSettings := func(t *testing.T, hasDBContainer bool) string {
		app := &DdevApp{Name: t.Name(), Type: nodeps.AppTypeDrupal11}
		settings := NewDrupalSettings(app)
		settings.HasDBContainer = hasDBContainer

		filePath := filepath.Join(t.TempDir(), "settings.ddev.php")
		require.NoError(t, writeDrupalSettingsDdevPhp(settings, filePath, app))
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		return string(content)
	}

	t.Run("db container present", func(t *testing.T) {
		content := writeSettings(t, true)

		require.Contains(t, content, `$databases['default']['default']['host'] = $host;`)
		require.Contains(t, content, `$settings['hash_salt']`)
	})

	t.Run("db container omitted", func(t *testing.T) {
		content := writeSettings(t, false)

		require.NotContains(t, content, `$databases`)
		require.NotContains(t, content, `$host`)
		// The non-DB settings must survive.
		require.Contains(t, content, `$settings['hash_salt']`)
		require.Contains(t, content, `trusted_host_patterns`)
	})
}
