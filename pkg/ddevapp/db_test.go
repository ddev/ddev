package ddevapp_test

import (
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

func GetDBTypeVersionFromString(t *testing.T) {
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
		"mariadb_12.3":  "mariadb:12.3",
		"mysql_5.7":     "mysql:5.7",
		"mysql_8.0":     "mysql:8.0",
		"mysql_8.4":     "mysql:8.4",
		"mysql_9.7":     "mysql:9.7",
	}

	for input, expectation := range expectations {
		require.Equal(t, expectation, ddevapp.GetDBTypeVersionFromString(input))
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
			app := &ddevapp.DdevApp{
				Database: ddevapp.DatabaseDesc{
					Type:    test.dbType,
					Version: test.dbVersion,
				},
			}

			result := app.GetPostgresDataDir()
			require.Equal(t, test.expectedPath, result, "Test case: %s", test.name)
		})
	}
}

// TestGetPostgresDataPath tests the GetPostgresDataPath function for correct data file paths
func TestGetPostgresDataPath(t *testing.T) {
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
			dbVersion:    nodeps.Postgres18,
			expectedPath: "/var/lib/postgresql/18/docker",
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
			app := &ddevapp.DdevApp{
				Database: ddevapp.DatabaseDesc{
					Type:    test.dbType,
					Version: test.dbVersion,
				},
			}

			result := app.GetPostgresDataPath()
			require.Equal(t, test.expectedPath, result, "Test case: %s", test.name)
		})
	}
}
