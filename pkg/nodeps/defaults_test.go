package nodeps_test

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

// TestVersionDefaultsReviewed makes adding a version force a decision about
// the default. A default that tests rely on to be current must be the newest
// valid version. The others record the newest version that existed when the
// default was last chosen, so adding a newer one fails here until someone
// decides whether the default moves and updates newestReviewed.
func TestVersionDefaultsReviewed(t *testing.T) {
	cases := []struct {
		constName      string
		defaultVersion string
		valid          map[string]bool
		trackNewest    bool
		newestReviewed string
	}{
		{"PostgresDefaultVersion", nodeps.PostgresDefaultVersion, nodeps.ValidPostgresVersions, true, ""},
		{"MySQLDefaultVersion", nodeps.MySQLDefaultVersion, nodeps.ValidMySQLVersions, false, nodeps.MySQL97},
		{"MariaDBDefaultVersion", nodeps.MariaDBDefaultVersion, nodeps.ValidMariaDBVersions, false, nodeps.MariaDB123},
		{"PHPDefault", nodeps.PHPDefault, nodeps.ValidPHPVersions, false, nodeps.PHP86},
	}
	for _, tc := range cases {
		t.Run(tc.constName, func(t *testing.T) {
			require.True(t, tc.valid[tc.defaultVersion], "%s=%s is not a valid version", tc.constName, tc.defaultVersion)
			newest := newestVersion(t, tc.valid)
			if tc.trackNewest {
				require.Equal(t, newest, tc.defaultVersion, "%s must be the newest valid version, %s, because tests use it as the current version", tc.constName, newest)
				return
			}
			require.Equal(t, tc.newestReviewed, newest, "version %s was added: decide whether %s (now %s) should change, then set newestReviewed for it to %s in this test", newest, tc.constName, tc.defaultVersion, newest)
		})
	}
}

func newestVersion(t *testing.T, valid map[string]bool) string {
	t.Helper()
	var newest *semver.Version
	newestStr := ""
	for v := range valid {
		sv, err := semver.NewVersion(v)
		require.NoError(t, err, "cannot parse version %q", v)
		if newest == nil || sv.GreaterThan(newest) {
			newest, newestStr = sv, v
		}
	}
	return newestStr
}
