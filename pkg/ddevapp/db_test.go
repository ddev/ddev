package ddevapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/stretchr/testify/require"
)

func TestDBTypeVersionFromString(t *testing.T) {
	expectations := map[string]string{
		"9":    "postgres:9",
		"9.6":  "postgres:9",
		"10":   "postgres:10",
		"11":   "postgres:11",
		"12":   "postgres:12",
		"13":   "postgres:13",
		"14":   "postgres:14",
		"15":   "postgres:15",
		"16":   "postgres:16",
		"17":   "postgres:17",
		"18":   "postgres:18", // PostgreSQL 18 support
		"19":   "postgres:19", // Future PostgreSQL versions
		"20":   "postgres:20",
		"5.5":  "mariadb:5.5",
		"5.6":  "mysql:5.6",
		"5.7":  "mysql:5.7",
		"8.0":  "mysql:8.0",
		"10.0": "mariadb:10.0",
		"10.1": "mariadb:10.1",
		"10.2": "mariadb:10.2",
		"10.3": "mariadb:10.3",
		"10.4": "mariadb:10.4",
		"10.5": "mariadb:10.5",
		"10.6": "mariadb:10.6",
		"10.7": "mariadb:10.7",

		"mariadb_10.2":  "mariadb:10.2",
		"mariadb_10.3":  "mariadb:10.3",
		"mariadb_10.4":  "mariadb:10.4",
		"mariadb_10.7":  "mariadb:10.7",
		"mariadb_10.11": "mariadb:10.11",
		"mariadb_11.4":  "mariadb:11.4",
		"mariadb_11.8":  "mariadb:11.8",
		"mysql_5.7":     "mysql:5.7",
		"mysql_8.0":     "mysql:8.0",
		"mysql_8.4":     "mysql:8.4",
	}

	for input, expectation := range expectations {
		require.Equal(t, expectation, dbTypeVersionFromString(input))
	}
}

// TestGetDBVersionFromVolumeScript tests the shell script logic used in getDBVersionFromVolume
// to ensure it properly detects PostgreSQL 18+ directory structures
func TestGetDBVersionFromVolumeScript(t *testing.T) {
	// Create temporary directory structure to simulate different DB volume layouts
	tempDir, err := os.MkdirTemp("", "ddev-db-test-"+t.Name())
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name           string
		setupFunc      func() string // Returns expected output
		expectedOutput string
	}{
		{
			name: "MySQL/MariaDB version file",
			setupFunc: func() string {
				mysqlDir := filepath.Join(tempDir, "mysql")
				err := os.MkdirAll(mysqlDir, 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(mysqlDir, "db_mariadb_version.txt"), []byte("mariadb_10.11"), 0644)
				require.NoError(t, err)
				return "mariadb_10.11"
			},
			expectedOutput: "mariadb_10.11",
		},
		{
			name: "PostgreSQL 17 (pre-18 structure)",
			setupFunc: func() string {
				postgresDir := filepath.Join(tempDir, "postgres")
				err := os.MkdirAll(postgresDir, 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(postgresDir, "PG_VERSION"), []byte("17"), 0644)
				require.NoError(t, err)
				return "17"
			},
			expectedOutput: "17",
		},
		{
			name: "PostgreSQL 18 (new structure)",
			setupFunc: func() string {
				postgresDir := filepath.Join(tempDir, "postgres", "18", "docker")
				err := os.MkdirAll(postgresDir, 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(postgresDir, "PG_VERSION"), []byte("18"), 0644)
				require.NoError(t, err)
				return "18"
			},
			expectedOutput: "18",
		},
		{
			name: "PostgreSQL 19 (future version)",
			setupFunc: func() string {
				postgresDir := filepath.Join(tempDir, "postgres", "19", "docker")
				err := os.MkdirAll(postgresDir, 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(postgresDir, "PG_VERSION"), []byte("19"), 0644)
				require.NoError(t, err)
				return "19"
			},
			expectedOutput: "19",
		},
		{
			name: "No version files found",
			setupFunc: func() string {
				// Create empty directories
				err := os.MkdirAll(filepath.Join(tempDir, "mysql"), 0755)
				require.NoError(t, err)
				err = os.MkdirAll(filepath.Join(tempDir, "postgres"), 0755)
				require.NoError(t, err)
				return ""
			},
			expectedOutput: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Clean up temp directory for each test
			os.RemoveAll(tempDir)
			err := os.MkdirAll(tempDir, 0755)
			require.NoError(t, err)

			// Setup test scenario
			test.setupFunc()

			// Test the shell script logic directly
			cmd := []string{"sh", "-c", `
				# Check MySQL/MariaDB version file
				if [ -f /var/tmp/mysql/db_mariadb_version.txt ]; then
					cat /var/tmp/mysql/db_mariadb_version.txt
					exit 0
				fi

				# Check PostgreSQL version file (pre-18 location)
				if [ -f /var/tmp/postgres/PG_VERSION ]; then
					cat /var/tmp/postgres/PG_VERSION
					exit 0
				fi

				# Check PostgreSQL 18+ version files in version-specific directories
				for version in 18 19 20 21 22; do
					if [ -f "/var/tmp/postgres/$version/docker/PG_VERSION" ]; then
						cat "/var/tmp/postgres/$version/docker/PG_VERSION"
						exit 0
					fi
				done

				# No version file found
				exit 0
			`}

			volumes := []string{
				filepath.Join(tempDir, "mysql") + ":/var/tmp/mysql",
				filepath.Join(tempDir, "postgres") + ":/var/tmp/postgres",
			}

			_, out, err := dockerutil.RunSimpleContainer(
				versionconstants.UtilitiesImage,
				"test-db-version-detection-"+util.RandString(6),
				cmd,
				[]string{}, // envVars
				[]string{}, // uid
				volumes,
				"",    // workingDir
				true,  // rm
				false, // detach
				map[string]string{`com.ddev.site-name`: ""}, // labels
				nil, // networks
				nil, // healthcheck
			)

			require.NoError(t, err)
			result := strings.Trim(out, " \n\r\t")
			require.Equal(t, test.expectedOutput, result, "Test case: %s", test.name)
		})
	}
}

// TestGetPostgresDataDir tests the GetPostgresDataDir function for correct directory paths
func TestGetPostgresDataDir(t *testing.T) {
	tests := []struct {
		name         string
		dbType       string
		dbVersion    string
		expectedPath string
	}{
		{
			name:         "PostgreSQL 9",
			dbType:       nodeps.Postgres,
			dbVersion:    nodeps.Postgres9,
			expectedPath: "/var/lib/postgresql/data",
		},
		{
			name:         "PostgreSQL 17",
			dbType:       nodeps.Postgres,
			dbVersion:    nodeps.Postgres17,
			expectedPath: "/var/lib/postgresql/data",
		},
		{
			name:         "PostgreSQL 18",
			dbType:       nodeps.Postgres,
			dbVersion:    "18",
			expectedPath: "/var/lib/postgresql",
		},
		{
			name:         "PostgreSQL 19",
			dbType:       nodeps.Postgres,
			dbVersion:    "19",
			expectedPath: "/var/lib/postgresql",
		},
		{
			name:         "MySQL (should return empty)",
			dbType:       nodeps.MySQL,
			dbVersion:    nodeps.MySQL80,
			expectedPath: "",
		},
		{
			name:         "MariaDB (should return empty)",
			dbType:       nodeps.MariaDB,
			dbVersion:    nodeps.MariaDB1011,
			expectedPath: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := &DdevApp{
				Database: DatabaseDesc{
					Type:    test.dbType,
					Version: test.dbVersion,
				},
			}

			result := app.GetPostgresDataDir()
			require.Equal(t, test.expectedPath, result, "Test case: %s", test.name)
		})
	}
}
