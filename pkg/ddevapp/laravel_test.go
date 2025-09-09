package ddevapp_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLaravelPostStartAction tests that Laravel post-start action respects omit_containers configuration
func TestLaravelPostStartAction(t *testing.T) {
	assert := asrt.New(t)
	origDir, _ := os.Getwd()

	// Clear Docker environment variables for test isolation
	testcommon.ClearDockerEnv()

	// Create a temporary directory for our test
	tmpDir := testcommon.CreateTmpDir(t.Name())
	testDir := filepath.Join(tmpDir, "laravel-test")
	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_ = os.RemoveAll(tmpDir)
	})

	// Create a minimal Laravel project structure
	err = os.MkdirAll(filepath.Join(testDir, "config"), 0755)
	require.NoError(t, err)

	// Create artisan file to make it detectable as Laravel
	err = os.WriteFile(filepath.Join(testDir, "artisan"), []byte("#!/usr/bin/env php\n<?php\n// Laravel artisan file"), 0644)
	require.NoError(t, err)

	// Create config/database.php with mariadb driver support
	databaseConfig := `<?php
return [
    'connections' => [
        'mariadb' => [
            'driver' => 'mariadb',
        ],
    ],
];`
	err = os.WriteFile(filepath.Join(testDir, "config", "database.php"), []byte(databaseConfig), 0644)
	require.NoError(t, err)

	// Create .env.example file
	envExample := `APP_NAME=Laravel
APP_ENV=local
APP_KEY=
APP_DEBUG=true
APP_URL=http://localhost

DB_CONNECTION=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_DATABASE=laravel
DB_USERNAME=root
DB_PASSWORD=

MAIL_MAILER=smtp
MAIL_HOST=mailpit
MAIL_PORT=1025`
	err = os.WriteFile(filepath.Join(testDir, ".env.example"), []byte(envExample), 0644)
	require.NoError(t, err)

	_ = os.Chdir(testDir)

	// Test 1: Normal Laravel project with database container
	t.Run("WithDatabaseContainer", func(t *testing.T) {
		// Test different database types
		databases := []struct {
			name       string
			dbType     string
			version    string
			port       string
			connection string
		}{
			{"MariaDB", nodeps.MariaDB, nodeps.MariaDBDefaultVersion, "3306", "mariadb"},
			{"MySQL", nodeps.MySQL, nodeps.MySQL80, "3306", "mysql"},
			{"PostgreSQL", nodeps.Postgres, nodeps.PostgresDefaultVersion, "5432", "pgsql"},
		}

		for _, db := range databases {
			t.Run(db.name, func(t *testing.T) {
				// Create a separate directory for this subtest
				subtestDir := filepath.Join(testDir, "WithDatabaseContainer", db.name)
				err := os.MkdirAll(subtestDir, 0755)
				require.NoError(t, err)

				// Copy Laravel files to subtest directory
				err = os.WriteFile(filepath.Join(subtestDir, "artisan"), []byte("#!/usr/bin/env php\n<?php\n// Laravel artisan file"), 0644)
				require.NoError(t, err)
				err = os.MkdirAll(filepath.Join(subtestDir, "config"), 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(subtestDir, "config", "database.php"), []byte(databaseConfig), 0644)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(subtestDir, ".env.example"), []byte(envExample), 0644)
				require.NoError(t, err)

				app, err := ddevapp.NewApp(subtestDir, false)
				require.NoError(t, err)
				app.Name = t.Name()
				app.Type = nodeps.AppTypeLaravel
				app.Database = ddevapp.DatabaseDesc{Type: db.dbType, Version: db.version}

				t.Cleanup(func() {
					_ = app.Stop(true, false)
					_ = os.Remove(filepath.Join(subtestDir, ".env"))
				})

				// Run the Laravel post-start action
				err = app.WriteConfig()
				require.NoError(t, err)

				// Run the post-start action
				err = app.PostStartAction()
				require.NoError(t, err)

				// Check that .env file was created and contains database settings
				envContent, err := os.ReadFile(filepath.Join(subtestDir, ".env"))
				require.NoError(t, err)
				envStr := string(envContent)

				assert.Contains(envStr, `DB_HOST="db"`)
				assert.Contains(envStr, fmt.Sprintf(`DB_PORT="%s"`, db.port))
				assert.Contains(envStr, `DB_DATABASE="db"`)
				assert.Contains(envStr, `DB_USERNAME="db"`)
				assert.Contains(envStr, `DB_PASSWORD="db"`)
				assert.Contains(envStr, fmt.Sprintf(`DB_CONNECTION="%s"`, db.connection))
				assert.Contains(envStr, `MAIL_MAILER="smtp"`)
				assert.Contains(envStr, `MAIL_HOST="127.0.0.1"`)
				assert.Contains(envStr, `MAIL_PORT="1025"`)
				assert.Contains(envStr, fmt.Sprintf(`APP_URL="%s"`, app.GetPrimaryURL()))
			})
		}
	})

	// Test 2: Laravel project with omitted database container
	t.Run("WithOmittedDatabaseContainer", func(t *testing.T) {
		// Create a separate directory for this subtest
		subtestDir := filepath.Join(testDir, "WithOmittedDatabaseContainer")
		err := os.MkdirAll(subtestDir, 0755)
		require.NoError(t, err)

		// Copy Laravel files to subtest directory
		err = os.WriteFile(filepath.Join(subtestDir, "artisan"), []byte("#!/usr/bin/env php\n<?php\n// Laravel artisan file"), 0644)
		require.NoError(t, err)
		err = os.MkdirAll(filepath.Join(subtestDir, "config"), 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(subtestDir, "config", "database.php"), []byte(databaseConfig), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(subtestDir, ".env.example"), []byte(envExample), 0644)
		require.NoError(t, err)

		app, err := ddevapp.NewApp(subtestDir, false)
		require.NoError(t, err)
		app.Name = t.Name()
		app.Type = nodeps.AppTypeLaravel
		app.Database = ddevapp.DatabaseDesc{Type: nodeps.MariaDB, Version: nodeps.MariaDBDefaultVersion}
		app.OmitContainers = []string{"db"} // Omit the database container

		t.Cleanup(func() {
			_ = app.Stop(true, false)
			_ = os.Remove(filepath.Join(subtestDir, ".env"))
		})

		// Create a custom .env with external database configuration
		customEnv := `APP_NAME=Laravel
APP_ENV=local
APP_URL=http://localhost
DB_HOST=ddev-external-db
DB_PORT=3306
DB_DATABASE=external_db
DB_USERNAME=external_user
DB_PASSWORD=external_pass
DB_CONNECTION=mysql`
		err = os.WriteFile(filepath.Join(subtestDir, ".env"), []byte(customEnv), 0644)
		require.NoError(t, err)

		err = app.WriteConfig()
		require.NoError(t, err)

		// Run the post-start action
		err = app.PostStartAction()
		require.NoError(t, err)

		// Check that .env file was updated but database settings were preserved
		envContent, err := os.ReadFile(filepath.Join(subtestDir, ".env"))
		require.NoError(t, err)
		envStr := string(envContent)

		// Database settings should NOT be overwritten when db container is omitted
		assert.NotContains(envStr, `DB_HOST="db"`)
		assert.NotContains(envStr, `DB_DATABASE="db"`)
		assert.NotContains(envStr, `DB_USERNAME="db"`)
		assert.NotContains(envStr, `DB_PASSWORD="db"`)

		// Mail settings should still be set
		assert.Contains(envStr, `MAIL_MAILER="smtp"`)
		assert.Contains(envStr, `MAIL_HOST="127.0.0.1"`)
		assert.Contains(envStr, `MAIL_PORT="1025"`)
		// APP_URL should be set to the primary URL
		assert.Contains(envStr, fmt.Sprintf(`APP_URL="%s"`, app.GetPrimaryURL()))

		// Original external database configuration should be preserved
		assert.Contains(envStr, "DB_HOST=ddev-external-db")
		assert.Contains(envStr, "DB_DATABASE=external_db")
		assert.Contains(envStr, "DB_USERNAME=external_user")
		assert.Contains(envStr, "DB_PASSWORD=external_pass")
	})

	// Test 3: Laravel project with ddev- prefixed host (should be overwritten)
	t.Run("WithDdevPrefixedHost", func(t *testing.T) {
		// Create a separate directory for this subtest
		subtestDir := filepath.Join(testDir, "WithDdevPrefixedHost")
		err := os.MkdirAll(subtestDir, 0755)
		require.NoError(t, err)

		// Copy Laravel files to subtest directory
		err = os.WriteFile(filepath.Join(subtestDir, "artisan"), []byte("#!/usr/bin/env php\n<?php\n// Laravel artisan file"), 0644)
		require.NoError(t, err)
		err = os.MkdirAll(filepath.Join(subtestDir, "config"), 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(subtestDir, "config", "database.php"), []byte(databaseConfig), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(subtestDir, ".env.example"), []byte(envExample), 0644)
		require.NoError(t, err)

		app, err := ddevapp.NewApp(subtestDir, false)
		require.NoError(t, err)
		app.Name = t.Name()
		app.Type = nodeps.AppTypeLaravel
		app.Database = ddevapp.DatabaseDesc{Type: nodeps.MariaDB, Version: nodeps.MariaDBDefaultVersion}

		t.Cleanup(func() {
			_ = app.Stop(true, false)
			_ = os.Remove(filepath.Join(subtestDir, ".env"))
		})

		// Create .env with ddev- prefixed database host
		envWithDdevHost := `APP_NAME=Laravel
APP_ENV=local
APP_URL=http://localhost
DB_HOST=ddev-external-project-db
DB_PORT=3306
DB_DATABASE=external_db
DB_USERNAME=external_user
DB_PASSWORD=external_pass
DB_CONNECTION=mysql`
		err = os.WriteFile(filepath.Join(subtestDir, ".env"), []byte(envWithDdevHost), 0644)
		require.NoError(t, err)

		err = app.WriteConfig()
		require.NoError(t, err)

		// Run the post-start action
		err = app.PostStartAction()
		require.NoError(t, err)

		// Check that .env file was updated but ddev- prefixed host was preserved
		envContent, err := os.ReadFile(filepath.Join(subtestDir, ".env"))
		require.NoError(t, err)
		envStr := string(envContent)

		// When ddev- prefixed host is found, ALL database settings should be still be overwritten
		assert.NotContains(envStr, "DB_HOST=ddev-external-project-db")
		assert.NotContains(envStr, "DB_PORT=3306")
		assert.NotContains(envStr, "DB_DATABASE=external_db")
		assert.NotContains(envStr, "DB_USERNAME=external_user")
		assert.NotContains(envStr, "DB_PASSWORD=external_pass")
		assert.NotContains(envStr, "DB_CONNECTION=mysql")

		assert.Contains(envStr, `DB_HOST="db"`)
		assert.Contains(envStr, `DB_PORT="3306"`)
		assert.Contains(envStr, `DB_DATABASE="db"`)
		assert.Contains(envStr, `DB_USERNAME="db"`)
		assert.Contains(envStr, `DB_PASSWORD="db"`)
		assert.Contains(envStr, `DB_CONNECTION="mariadb"`)
		assert.Contains(envStr, `MAIL_MAILER="smtp"`)
		assert.Contains(envStr, `MAIL_HOST="127.0.0.1"`)
		assert.Contains(envStr, `MAIL_PORT="1025"`)
		assert.Contains(envStr, fmt.Sprintf(`APP_URL="%s"`, app.GetPrimaryURL()))
	})

	// Test 4: Laravel project with disable_settings_management
	t.Run("WithDisabledSettingsManagement", func(t *testing.T) {
		// Create a separate directory for this subtest
		subtestDir := filepath.Join(testDir, "WithDisabledSettingsManagement")
		err := os.MkdirAll(subtestDir, 0755)
		require.NoError(t, err)

		// Copy Laravel files to subtest directory
		err = os.WriteFile(filepath.Join(subtestDir, "artisan"), []byte("#!/usr/bin/env php\n<?php\n// Laravel artisan file"), 0644)
		require.NoError(t, err)
		err = os.MkdirAll(filepath.Join(subtestDir, "config"), 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(subtestDir, "config", "database.php"), []byte(databaseConfig), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(subtestDir, ".env.example"), []byte(envExample), 0644)
		require.NoError(t, err)

		app, err := ddevapp.NewApp(subtestDir, false)
		require.NoError(t, err)
		app.Name = t.Name()
		app.Type = nodeps.AppTypeLaravel
		app.Database = ddevapp.DatabaseDesc{Type: nodeps.MariaDB, Version: nodeps.MariaDBDefaultVersion}
		app.DisableSettingsManagement = true

		t.Cleanup(func() {
			_ = app.Stop(true, false)
			_ = os.Remove(filepath.Join(subtestDir, ".env"))
		})

		// Create a custom .env that should not be modified
		originalEnv := `APP_NAME=MyCustomApp
DB_HOST=my-custom-host
DB_PORT=5432
DB_DATABASE=my_custom_db`
		err = os.WriteFile(filepath.Join(subtestDir, ".env"), []byte(originalEnv), 0644)
		require.NoError(t, err)

		err = app.WriteConfig()
		require.NoError(t, err)

		// Run the post-start action
		err = app.PostStartAction()
		require.NoError(t, err)

		// Check that .env file was NOT modified
		envContent, err := os.ReadFile(filepath.Join(subtestDir, ".env"))
		require.NoError(t, err)
		envStr := string(envContent)

		// Original content should be preserved exactly
		assert.Equal(originalEnv, strings.TrimSpace(envStr))
		assert.Contains(envStr, "DB_HOST=my-custom-host")
		assert.Contains(envStr, "DB_PORT=5432")
		assert.NotContains(envStr, `MAIL_MAILER="smtp"`)
	})

	// Test 5: Laravel project with different ddev-prefixed hostnames
	t.Run("WithDifferentHostsThatWillBeOverwritten", func(t *testing.T) {
		// Test different ddev- prefixed hostnames to verify our detection logic
		testHosts := []string{
			"external-db",
			"ddev-shared-database",
			"ddev-another-project-db",
		}

		for _, hostname := range testHosts {
			// Create a separate directory for this subtest
			subtestDir := filepath.Join(testDir, "DifferentDdevPrefixedHosts", hostname)
			err := os.MkdirAll(subtestDir, 0755)
			require.NoError(t, err)

			// Copy Laravel files to subtest directory
			err = os.WriteFile(filepath.Join(subtestDir, "artisan"), []byte("#!/usr/bin/env php\n<?php\n// Laravel artisan file"), 0644)
			require.NoError(t, err)
			err = os.MkdirAll(filepath.Join(subtestDir, "config"), 0755)
			require.NoError(t, err)
			err = os.WriteFile(filepath.Join(subtestDir, "config", "database.php"), []byte(databaseConfig), 0644)
			require.NoError(t, err)
			err = os.WriteFile(filepath.Join(subtestDir, ".env.example"), []byte(envExample), 0644)
			require.NoError(t, err)

			app, err := ddevapp.NewApp(subtestDir, false)
			require.NoError(t, err)
			app.Name = t.Name() + hostname
			app.Type = nodeps.AppTypeLaravel
			app.Database = ddevapp.DatabaseDesc{Type: nodeps.MariaDB, Version: nodeps.MariaDBDefaultVersion}

			t.Cleanup(func() {
				_ = app.Stop(true, false)
				_ = os.Remove(filepath.Join(subtestDir, ".env"))
			})

			// Create .env file with custom DB_HOST that should be overwritten
			envContent := fmt.Sprintf(`APP_NAME=Laravel
APP_ENV=local
APP_URL=http://localhost
DB_HOST=%s
DB_PORT=3306
DB_DATABASE=external_db
DB_USERNAME=external_user
DB_PASSWORD=external_pass
DB_CONNECTION=mysql`, hostname)
			err = os.WriteFile(filepath.Join(subtestDir, ".env"), []byte(envContent), 0644)
			require.NoError(t, err)

			err = app.WriteConfig()
			require.NoError(t, err)

			// Run the post-start action
			err = app.PostStartAction()
			require.NoError(t, err)

			// Check that ALL database settings were preserved
			envResult, err := os.ReadFile(filepath.Join(subtestDir, ".env"))
			require.NoError(t, err)
			envStr := string(envResult)

			// Custom database settings should be overwritten
			assert.NotContains(envStr, fmt.Sprintf("DB_HOST=%s", hostname), "Original DB_HOST should be overwritten")
			assert.NotContains(envStr, "DB_DATABASE=external_db", "DB_DATABASE should be overwritten")
			assert.NotContains(envStr, "DB_USERNAME=external_user", "DB_USERNAME should be overwritten")
			assert.NotContains(envStr, "DB_PASSWORD=external_pass", "DB_PASSWORD should be overwritten")

			// All settings should still be managed by DDEV
			assert.Contains(envStr, `DB_HOST="db"`)
			assert.Contains(envStr, `DB_PORT="3306"`)
			assert.Contains(envStr, `DB_DATABASE="db"`)
			assert.Contains(envStr, `DB_USERNAME="db"`)
			assert.Contains(envStr, `DB_PASSWORD="db"`)
			assert.Contains(envStr, `DB_CONNECTION="mariadb"`)
			assert.Contains(envStr, `MAIL_MAILER="smtp"`)
			assert.Contains(envStr, `MAIL_HOST="127.0.0.1"`)
			assert.Contains(envStr, `MAIL_PORT="1025"`)
			assert.Contains(envStr, fmt.Sprintf(`APP_URL="%s"`, app.GetPrimaryURL()))
		}
	})
}
