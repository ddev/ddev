package ddevapp_test

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dotenv"
	"github.com/ddev/ddev/pkg/fileutil"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnvValuesInContainerAndPHP tests that a value DDEV writes to an env file
// arrives unchanged in the container, in ${VAR} interpolation of a compose
// file, and in the PHP dotenv parsers frameworks use for their own .env
func TestEnvValuesInContainerAndPHP(t *testing.T) {
	origDir, _ := os.Getwd()
	site := TestSites[0]
	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)
	checkDir := filepath.Join(app.AppRoot, ".envcheck")
	webEnvFile := app.GetConfigPath(".env.web")
	composeFile := app.GetConfigPath("docker-compose.envcheck.yaml")
	t.Cleanup(func() {
		require.NoError(t, app.Stop(true, false))
		_ = os.RemoveAll(checkDir)
		_ = os.Remove(webEnvFile)
		_ = os.Remove(composeFile)
	})

	written := map[string]string{
		"ENVCHECK_PAREN":     ")value(",
		"ENVCHECK_SHELL_OPS": "a;b&c|d<e>f",
		"ENVCHECK_SQUOTE":    "it's",
		"ENVCHECK_DQUOTE":    `say "hi"`,
		"ENVCHECK_BACKSLASH": `C:\new\table`,
		"ENVCHECK_TRAILING":  `ends with \`,
		"ENVCHECK_DOLLAR":    "asd$asd ${HOME} $$",
		"ENVCHECK_TEMPLATE":  "{{ .Name }}",
		"ENVCHECK_NEWLINE":   "line1\nline2",
		"ENVCHECK_HASH":      "value #not a comment",
		"ENVCHECK_EQUALS":    "a=b=c",
		"ENVCHECK_SPACES":    "  two  spaces  ",
		"ENVCHECK_UNICODE":   "héllo ✓",
	}
	// Written by hand, these follow compose's ${VAR} rules
	handWritten := map[string]string{
		"ENVCHECK_HAND_UNQUOTED": "asd",
		"ENVCHECK_HAND_DEFAULT":  "def",
		"ENVCHECK_HAND_REF":      "base-x",
		"ENVCHECK_HAND_ESCAPED":  "a$b",
		"ENVCHECK_HAND_SINGLE":   "asd$envcheck_unset",
	}
	envText := dotenv.Update("ENVCHECK_HAND_BASE=base\n"+
		"ENVCHECK_HAND_UNQUOTED=asd$envcheck_unset\n"+
		"ENVCHECK_HAND_DEFAULT=${ENVCHECK_UNSET:-def}\n"+
		`ENVCHECK_HAND_REF="${ENVCHECK_HAND_BASE}-x"`+"\n"+
		"ENVCHECK_HAND_ESCAPED=a$$b\n"+
		"ENVCHECK_HAND_SINGLE='asd$envcheck_unset'\n", written)
	require.NoError(t, os.WriteFile(webEnvFile, []byte(envText), 0644))

	var compose strings.Builder
	compose.WriteString("services:\n  web:\n    environment:\n")
	names := append(slices.Collect(maps.Keys(written)), slices.Collect(maps.Keys(handWritten))...)
	slices.Sort(names)
	for _, name := range names {
		compose.WriteString("      - INTERP_" + name + "=${" + name + "}\n")
	}
	require.NoError(t, os.WriteFile(composeFile, []byte(compose.String()), 0644))

	require.NoError(t, os.MkdirAll(checkDir, 0755))
	require.NoError(t, fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), "envcheck.php"), filepath.Join(checkDir, "envcheck.php")))
	require.NoError(t, ddevapp.WriteProjectEnvFile(filepath.Join(checkDir, ".env"), written, ""))

	require.NoError(t, app.Restart())
	_, _, err = app.Exec(&ddevapp.ExecOpts{Cmd: "composer require --working-dir=/var/www/html/.envcheck -q --no-interaction vlucas/phpdotenv symfony/dotenv"})
	require.NoError(t, err)
	namesJSON, err := json.Marshal(names)
	require.NoError(t, err)
	out, _, err := app.Exec(&ddevapp.ExecOpts{RawCmd: []string{"php", "/var/www/html/.envcheck/envcheck.php", string(namesJSON)}})
	require.NoError(t, err)

	var got map[string]map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &got), out)
	for name, want := range written {
		for _, source := range []string{"env", "interpolated", "phpdotenv", "symfony"} {
			require.Equal(t, want, got[name][source], "%s from %s", name, source)
		}
	}
	for name, want := range handWritten {
		for _, source := range []string{"env", "interpolated"} {
			require.Equal(t, want, got[name][source], "%s from %s", name, source)
		}
	}
}

func TestWriteProjectEnvFile(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	site := TestSites[0]
	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.RemoveAll(app.GetConfigPath(".env"))
	})

	testEnvFiles, err := fileutil.ListFilesInDirFullPath(filepath.Join(origDir, "testdata", t.Name()), false)
	require.NoError(t, err)
	appEnvFile := filepath.Join(app.AppRoot, ".env")
	for _, envFileName := range testEnvFiles {
		_ = os.RemoveAll(appEnvFile)
		err = fileutil.CopyFile(envFileName, appEnvFile)
		require.NoError(t, err)
		readEnvMap, readEnvText, err := ddevapp.ReadProjectEnvFile(appEnvFile)
		require.NoError(t, err)
		_ = readEnvMap

		// Override with some new items
		writeEnvMap := map[string]string{
			"DB_VAL_DIDNOTEXIST": "new_db_val_didnotexist",
			"DB_HOST":            "newdbhost",
			"DB_DATABASE":        "newdbdatabase",
			"DB_USERNAME":        "newdbusername",
			"DB_PASSWORD":        "newdbpassword",
			"DB_CONNECTION":      "new_mysql://root:root@somehost/somedb",
		}
		err = ddevapp.WriteProjectEnvFile(appEnvFile, writeEnvMap, readEnvText)
		require.NoError(t, err)

		postWriteEnvMap, postWriteEnvText, err := ddevapp.ReadProjectEnvFile(appEnvFile)
		require.NoError(t, err)

		// Make sure that the values we intended to change got changed
		for k := range writeEnvMap {
			assert.Equal(writeEnvMap[k], postWriteEnvMap[k], "Expected values for %s to match but writeEnvMap[%s]='%s' and postWriteEnvMap[%s]='%s' (envfile=%s)", k, k, writeEnvMap[k], k, postWriteEnvMap[k], envFileName)
		}

		// Now examine all values that should not have been changed
		for k := range readEnvMap {
			if _, ok := writeEnvMap[k]; ok {
				// If we intended to write the var, don't test, as we deliberately overwrite and tested above.
				continue
			}
			assert.Equal(readEnvMap[k], postWriteEnvMap[k], "Expected (unchanged) values for %s to match but readEnvMap[%s]='%s' and postWriteEnvMap[%s]='%s' (envfile=%s)", k, k, readEnvMap[k], k, postWriteEnvMap[k], envFileName)
		}

		// Look for comments that should have been preserved
		origLines := strings.Split(readEnvText, "\n")
		newLines := strings.Split(postWriteEnvText, "\n")
		for i, l := range origLines {
			if strings.HasPrefix(l, `#`) {
				assert.Equal(l, newLines[i], "comment '%s' in original .env expected in revised .env but doesn't match (envfile=%s)", l, envFileName)
			}
		}
	}
}
