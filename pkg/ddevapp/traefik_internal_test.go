package ddevapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
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

// TestProcessHTTPExposeMailpit checks when the Mailpit route is omitted.
func TestProcessHTTPExposeMailpit(t *testing.T) {
	origHardened := globalconfig.DdevGlobalConfig.UseHardenedImages
	origLetsEncrypt := globalconfig.DdevGlobalConfig.UseLetsEncrypt
	origBindAll := globalconfig.DdevGlobalConfig.RouterBindAllInterfaces
	t.Cleanup(func() {
		globalconfig.DdevGlobalConfig.UseHardenedImages = origHardened
		globalconfig.DdevGlobalConfig.UseLetsEncrypt = origLetsEncrypt
		globalconfig.DdevGlobalConfig.RouterBindAllInterfaces = origBindAll
	})

	// On a remote Docker host the router is reachable from other hosts
	// even without router_bind_all_interfaces.
	remote := dockerutil.IsRemoteDockerHost()

	testCases := []struct {
		name          string
		hardened      bool
		letsEncrypt   bool
		bindAll       bool
		expectMailpit bool
	}{
		{name: "default", expectMailpit: true},
		{name: "bind all interfaces", bindAll: true, expectMailpit: true},
		{name: "letsencrypt on localhost", letsEncrypt: true, expectMailpit: !remote},
		{name: "letsencrypt on all interfaces", letsEncrypt: true, bindAll: true, expectMailpit: false},
		{name: "hardened images", hardened: true, expectMailpit: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			globalconfig.DdevGlobalConfig.UseHardenedImages = tc.hardened
			globalconfig.DdevGlobalConfig.UseLetsEncrypt = tc.letsEncrypt
			globalconfig.DdevGlobalConfig.RouterBindAllInterfaces = tc.bindAll

			routes, err := processHTTPExpose("web", "80:80,8025:8025", false, []string{"example.ddev.site"})
			require.NoError(t, err)

			var ports []string
			for _, r := range routes {
				ports = append(ports, r.Service.InternalServicePort)
			}
			require.Contains(t, ports, "80")
			if tc.expectMailpit {
				require.Contains(t, ports, "8025")
			} else {
				require.NotContains(t, ports, "8025")
			}
		})
	}
}
