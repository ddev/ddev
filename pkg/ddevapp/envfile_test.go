package ddevapp_test

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteProjectEnvFile(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	site := TestSites[0]
	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.RemoveAll(app.GetConfigPath(".env"))
	})

	testEnvFiles, err := fileutil.ListFilesInDirFullPath(filepath.Join(origDir, "testdata", t.Name()))
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
