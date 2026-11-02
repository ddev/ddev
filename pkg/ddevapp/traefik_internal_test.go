package ddevapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

// TestRemoveStaleTraefikFiles checks that only #ddev-generated files from
// another project name are removed, and that add-on files are kept.
func TestRemoveStaleTraefikFiles(t *testing.T) {
	app := &DdevApp{Name: "newname", AppRoot: t.TempDir()}
	configDir := app.GetConfigPath("traefik/config")
	certsDir := app.GetConfigPath("traefik/certs")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.MkdirAll(certsDir, 0755))

	generated := nodeps.DdevFileSignature + "\nhttp: {}\n"
	userOwned := "http: {}\n"
	files := map[string]string{
		"traefik/config/newname.yaml":    generated,
		"traefik/config/oldname.yaml":    generated,
		"traefik/config/user.yaml":       userOwned,
		"traefik/config/addon.yaml":      generated,
		"traefik/certs/newname.crt":      generated,
		"traefik/certs/newname.key":      generated,
		"traefik/certs/oldname.crt":      generated,
		"traefik/certs/oldname.key":      generated,
		"traefik/certs/mtls-ca.crt":      userOwned,
		"addon-metadata/a/manifest.yaml": "name: a\nproject_files:\n  - traefik/config/addon.yaml\n",
	}
	for name, content := range files {
		p := app.GetConfigPath(name)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0644))
	}

	removeStaleTraefikFiles(app, configDir, certsDir)

	removed := map[string]bool{
		"traefik/config/oldname.yaml": true,
		"traefik/certs/oldname.crt":   true,
		"traefik/certs/oldname.key":   true,
	}
	for name := range files {
		if removed[name] {
			require.NoFileExists(t, app.GetConfigPath(name))
		} else {
			require.FileExists(t, app.GetConfigPath(name))
		}
	}
}
