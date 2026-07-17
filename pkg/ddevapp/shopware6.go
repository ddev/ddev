package ddevapp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/util"
	copy2 "github.com/otiai10/copy"
)

// isShopware6App returns true if the app is of type shopware6
func isShopware6App(app *DdevApp) bool {
	isShopware6, err := fileutil.FgrepStringInFile(filepath.Join(app.AppRoot, app.ComposerRoot, "composer.json"), `"name": "shopware/production"`)
	if err == nil && isShopware6 {
		return true
	}
	return false
}

// setShopware6SiteSettingsPaths sets the paths to .env.local file.
func setShopware6SiteSettingsPaths(app *DdevApp) {
	app.SiteSettingsPath = filepath.Join(app.AppRoot, app.ComposerRoot, ".env.local")
}

// shopware6ConfigOverrideAction exposes the ports needed by the shopware-cli
// watchers (admin-watch/storefront-watch), so a shopware6 project can run them
// without installing a separate add-on. Ports are only added when not already
// present, so it is safe to re-run and it never clobbers a user's own settings.
// The watcher environment (PROXY_URL etc.) is intentionally NOT set here: it is
// set at runtime inside the watcher commands, where ${DDEV_PRIMARY_URL} is a live
// variable. Putting it in web_environment is fragile (it can be replaced by a
// later `ddev config`, and ${DDEV_PRIMARY_URL} is not expanded there).
// Targets Shopware 6.7.4.2+ (Vite admin on 5173); see the bundled commands/web
// watchers.
func shopware6ConfigOverrideAction(app *DdevApp) error {
	watcherPorts := []WebExposedPort{
		{Name: "shopware-vite-admin", WebContainerPort: 5173, HTTPPort: 5172, HTTPSPort: 5173},
		{Name: "shopware-storefront-proxy", WebContainerPort: 9998, HTTPPort: 8888, HTTPSPort: 9998},
		{Name: "shopware-storefront-assets", WebContainerPort: 9999, HTTPPort: 8889, HTTPSPort: 9999},
	}
	for _, p := range watcherPorts {
		if !hasWebExposedPort(app.WebExtraExposedPorts, p.WebContainerPort) {
			app.WebExtraExposedPorts = append(app.WebExtraExposedPorts, p)
		}
	}

	return nil
}

// hasWebExposedPort reports whether a port with the given container port is
// already exposed, so shopware6ConfigOverrideAction does not add duplicates.
func hasWebExposedPort(ports []WebExposedPort, containerPort int) bool {
	for _, p := range ports {
		if p.WebContainerPort == containerPort {
			return true
		}
	}
	return false
}

// shopware6ImportFilesAction defines the shopware6 workflow for importing user-generated files.
func shopware6ImportFilesAction(app *DdevApp, uploadDir, importPath, extPath string) error {
	destPath := app.calculateHostUploadDirFullPath(uploadDir)

	// parent of destination dir should exist
	if !fileutil.FileExists(filepath.Dir(destPath)) {
		return fmt.Errorf("unable to import to %s: parent directory does not exist", destPath)
	}

	// parent of destination dir should be writable.
	if err := util.Chmod(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// If the destination path exists, purge it as was warned
	if fileutil.FileExists(destPath) {
		if err := fileutil.PurgeDirectory(destPath); err != nil {
			return fmt.Errorf("failed to cleanup %s before import: %v", destPath, err)
		}
	}

	if isTar(importPath) {
		if err := archive.Untar(importPath, destPath, extPath); err != nil {
			return fmt.Errorf("failed to extract provided archive: %v", err)
		}

		return nil
	}

	if isZip(importPath) {
		if err := archive.Unzip(importPath, destPath, extPath); err != nil {
			return fmt.Errorf("failed to extract provided archive: %v", err)
		}

		return nil
	}

	if err := copy2.Copy(importPath, destPath); err != nil {
		return err
	}

	return nil
}

// getShopwareUploadDirs will return the default paths.
func getShopwareUploadDirs(_ *DdevApp) []string {
	return []string{"media"}
}

// shopware6PostStartAction checks to see if the .env.local file is set up
func shopware6PostStartAction(app *DdevApp) error {
	if app.DisableSettingsManagement {
		return nil
	}
	envFilePath := filepath.Join(app.AppRoot, app.ComposerRoot, ".env.local")
	_, envText, err := ReadProjectEnvFile(envFilePath)
	var envMap = map[string]string{
		"DATABASE_URL": `mysql://db:db@db:3306/db`,
		"APP_ENV":      "dev",
		"APP_URL":      app.GetPrimaryURL(),
		"MAILER_DSN":   `smtp://127.0.0.1:1025?encryption=&auth_mode=`,
	}
	// If the .env.local doesn't exist, create it.
	switch {
	case err == nil:
		util.Warning("Updating %s with %v", envFilePath, envMap)
		fallthrough
	case errors.Is(err, os.ErrNotExist):
		err := WriteProjectEnvFile(envFilePath, envMap, envText)
		if err != nil {
			return err
		}
	default:
		util.Warning("error opening %s: %v", envFilePath, err)
	}

	return nil
}
