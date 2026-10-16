package ddevapp

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/dotenv"
	"github.com/stretchr/testify/require"
)

// unmanagedEnvLines are lines no project type manages, so every writer must
// leave them byte for byte as they are
const unmanagedEnvLines = `# Settings DDEV must not touch
CUSTOM_NAME=My App
APP_KEY=base64:abc+/def=
MAIL_FROM_NAME="${CUSTOM_NAME}"
SINGLE_QUOTED='keep $this \n as is'
UNQUOTED=value # trailing comment
SPACED = spaced value
MULTILINE="first
second"
EMPTY=

# Note: a prose comment with a colon
#CUSTOM_COMMENTED=keep
`

var envLineNameRegex = regexp.MustCompile(`^[ \t]*#*[ \t]*(?:export[ \t]+)?([A-Za-z0-9_.\-]+)[ \t]*=`)

// TestProjectTypeEnvWritersKeepOtherLines tests that each project type's .env
// writer changes only the keys it manages, in place, and leaves every other
// line, comment and blank line as it was
func TestProjectTypeEnvWritersKeepOtherLines(t *testing.T) {
	type writer func(*DdevApp) error
	withPath := func(f func(*DdevApp) (string, error)) writer {
		return func(app *DdevApp) error { _, err := f(app); return err }
	}
	testCases := []struct {
		name string
		// seedFile is written, checkFile is read back; they differ when the
		// writer starts from an example file
		seedFile, checkFile string
		write               writer
		// active is set in the seed with an old value, commented is commented out
		active, commented string
		crlf              bool
		// rerun is false when the writer copies its example file on every start
		rerun bool
		// files are other files the writer needs
		files map[string]string
	}{
		{name: "laravel", seedFile: ".env", write: laravelPostStartAction, active: "DB_HOST=127.0.0.1", commented: "# DB_PORT=3306", rerun: true},
		{name: "laravel from .env.example", seedFile: ".env.example", checkFile: ".env", write: laravelPostStartAction, active: "DB_HOST=127.0.0.1", commented: "# DB_PORT=3306", rerun: true},
		{name: "laravel with CRLF", seedFile: ".env", write: laravelPostStartAction, active: "DB_HOST=127.0.0.1", commented: "# DB_PORT=3306", crlf: true, rerun: true},
		{name: "asterios", seedFile: ".env", write: asteriosPostStartAction, active: "APP_URL=http://localhost", commented: "# DB_HOST=127.0.0.1", rerun: true},
		{name: "silverstripe", seedFile: ".env", write: silverstripePostStartAction, active: "SS_DATABASE_SERVER=localhost", commented: "# SS_DATABASE_NAME=silverstripe", rerun: true},
		{name: "wordpress bedrock", seedFile: ".env", write: wpBedrockPostStartAction, active: "DB_NAME=wordpress", commented: "# DB_HOST=localhost", rerun: true},
		{name: "codeigniter", seedFile: ".env", write: withPath(createCodeIgniterSettingsFile), active: "CI_ENVIRONMENT = production", commented: "# database.default.hostname = localhost", rerun: true},
		{name: "craftcms", seedFile: ".ddev/.env.web", write: withPath(updateCraftCMSDotEnv), active: "CRAFT_DB_SERVER=localhost", commented: "#PRIMARY_SITE_URL=https://example.com", rerun: true},
		{name: "cakephp", seedFile: "config/.env.example", checkFile: "config/.env", write: cakephpPostStartAction, active: `export APP_NAME="app"`, commented: `# export DEBUG="false"`, files: map[string]string{"config/bootstrap.php": "<?php\n"}},
		{name: "symfony", seedFile: ".env.local", write: symfonyPostStartAction, active: "MAILER_DSN=null://null", commented: "# DATABASE_HOST=localhost", rerun: true},
		{name: "shopware6", seedFile: ".env.local", write: shopware6PostStartAction, active: "APP_ENV=prod", commented: "# APP_URL=http://localhost", rerun: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app, err := NewApp(t.TempDir(), false)
			require.NoError(t, err)
			app.Name = "envwriters"
			setCraftCMSDotFileLocation(app)

			// The managed lines sit in the middle, so their position is checked too
			head, tail, _ := strings.Cut(unmanagedEnvLines, "\n\n")
			seed := head + "\n" + tc.active + "\n" + tc.commented + "\n\n" + tail
			if tc.crlf {
				seed = strings.ReplaceAll(seed, "\n", "\r\n")
			}
			seedPath := filepath.Join(app.AppRoot, tc.seedFile)
			require.NoError(t, os.MkdirAll(filepath.Dir(seedPath), 0755))
			require.NoError(t, os.WriteFile(seedPath, []byte(seed), 0644))
			for name, contents := range tc.files {
				require.NoError(t, os.WriteFile(filepath.Join(app.AppRoot, name), []byte(contents), 0644))
			}
			checkPath := seedPath
			if tc.checkFile != "" {
				checkPath = filepath.Join(app.AppRoot, tc.checkFile)
			}

			require.NoError(t, tc.write(app))
			b, err := os.ReadFile(checkPath)
			require.NoError(t, err)
			result := string(b)

			before, err := dotenv.ParseLiteral(seedPath, seed)
			require.NoError(t, err)
			after, err := dotenv.ParseLiteral(checkPath, result)
			require.NoError(t, err)
			changed := map[string]bool{}
			for name, value := range after {
				if old, ok := before[name]; !ok || old != value {
					changed[name] = true
				}
			}
			for name := range before {
				require.Contains(t, after, name, "%s was removed", name)
			}
			activeName := envLineNameRegex.FindStringSubmatch(tc.active)[1]
			commentedName := envLineNameRegex.FindStringSubmatch(tc.commented)[1]
			require.True(t, changed[activeName], "%s was not updated:\n%s", activeName, result)
			require.True(t, changed[commentedName], "%s was not uncommented:\n%s", commentedName, result)

			unmanaged, err := dotenv.ParseLiteral("unmanaged", unmanagedEnvLines)
			require.NoError(t, err)
			for name := range unmanaged {
				require.False(t, changed[name], "%s must not be changed:\n%s", name, result)
			}

			// Without the managed lines, the seed and the result must be identical
			seedLines, resultLines := strings.SplitAfter(seed, "\n"), strings.SplitAfter(result, "\n")
			require.Equal(t, withoutNames(seedLines, changed), withoutNames(resultLines, changed))
			require.Equal(t, lineIndex(seedLines, activeName), lineIndex(resultLines, activeName), "%s moved", activeName)
			require.Equal(t, lineIndex(seedLines, commentedName), lineIndex(resultLines, commentedName), "%s moved", commentedName)
			if tc.crlf {
				require.Equal(t, strings.Count(result, "\n"), strings.Count(result, "\r\n"), "every line must keep CRLF:\n%q", result)
			}

			if !tc.rerun {
				return
			}
			past := time.Now().Add(-time.Hour).Truncate(time.Second)
			require.NoError(t, os.Chtimes(checkPath, past, past))
			require.NoError(t, tc.write(app))
			b, err = os.ReadFile(checkPath)
			require.NoError(t, err)
			require.Equal(t, result, string(b), "a second start must not change the file")
			info, err := os.Stat(checkPath)
			require.NoError(t, err)
			require.True(t, info.ModTime().Equal(past), "a second start must not rewrite the file")
		})
	}
}

// withoutNames returns lines without those that set or comment out one of names
func withoutNames(lines []string, names map[string]bool) []string {
	var kept []string
	for _, line := range lines {
		if m := envLineNameRegex.FindStringSubmatch(line); m != nil && names[m[1]] {
			continue
		}
		kept = append(kept, line)
	}
	return kept
}

// lineIndex returns the index of the first line that sets or comments out name
func lineIndex(lines []string, name string) int {
	for i, line := range lines {
		if m := envLineNameRegex.FindStringSubmatch(line); m != nil && m[1] == name {
			return i
		}
	}
	return -1
}
