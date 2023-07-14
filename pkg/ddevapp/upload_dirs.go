package ddevapp

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/util"
)

// addUploadDir adds a new upload dir if it does not already exist in the list.
func (app *DdevApp) addUploadDir(uploadDir string) {
	err := app.validateUploadDirs()
	if err != nil {
		util.Failed("Failed to validate upload_dirs: %v", err)
	}

	for _, existingUploadDir := range app.UploadDirs {
		if uploadDir == existingUploadDir {
			return
		}
	}

	app.UploadDirs = append(app.UploadDirs, uploadDir)
}

// GetUploadDir returns the first upload (public files) directory.
// This value is relative to the docroot
func (app *DdevApp) GetUploadDir() string {
	uploadDirs := app.GetUploadDirs()
	if len(uploadDirs) > 0 {
		return uploadDirs[0]
	}

	return ""
}

// GetUploadDirs returns the upload (public files) directories.
// These are gathered from the per-CMS configurations and the
// value of upload_dirs. upload_dirs overrides the per-CMS configuration
func (app *DdevApp) GetUploadDirs() []string {
	err := app.validateUploadDirs()
	if err != nil {
		util.Warning("Ignoring invalid upload_dirs value: %v", err)
		return []string{}
	}

	if app.UploadDirDeprecated != "" {
		uploadDirDeprecated := app.UploadDirDeprecated
		app.UploadDirDeprecated = ""
		app.addUploadDir(uploadDirDeprecated)
	}

	// If an UploadDirs has been specified for the app, it overrides
	// anything that the project type would give us.
	if len(app.UploadDirs) > 0 {
		return app.UploadDirs
	}

	// Otherwise continue to get the UploadDirs from the project type
	appFuncs, ok := appTypeMatrix[app.GetType()]
	if ok && appFuncs.uploadDirs != nil {
		return appFuncs.uploadDirs(app)
	}

	return []string{}
}

// IsUploadDirsWarningDisabled returns true if UploadDirs is disabled by the user.
func (app *DdevApp) IsUploadDirsWarningDisabled() bool {
	return app.DisableUploadDirsWarning
}

// calculateHostUploadDirFullPath returns the full path to the upload directory
// on the host or "" if there is none.
func (app *DdevApp) calculateHostUploadDirFullPath(uploadDir string) string {
	if uploadDir != "" {
		return path.Clean(path.Join(app.AppRoot, app.Docroot, uploadDir))
	}

	return ""
}

// GetHostUploadDirFullPath returns the full path to the first upload directory on the
// host or "" if there is none.
func (app *DdevApp) GetHostUploadDirFullPath() string {
	uploadDirs := app.GetUploadDirs()
	if len(uploadDirs) > 0 {
		return app.calculateHostUploadDirFullPath(uploadDirs[0])
	}

	return ""
}

// calculateContainerUploadDirFullPath returns the full path to the upload
// directory in container or "" if there is none.
func (app *DdevApp) calculateContainerUploadDirFullPath(uploadDir string) string {
	if uploadDir != "" {
		return path.Clean(path.Join("/var/www/html", app.Docroot, uploadDir))
	}

	return ""
}

// getContainerUploadDir returns the full path to the first upload
// directory in container or "" if there is none.
func (app *DdevApp) getContainerUploadDir() string {
	uploadDirs := app.GetUploadDirs()
	if len(uploadDirs) > 0 {
		return app.calculateContainerUploadDirFullPath(uploadDirs[0])
	}

	return ""
}

// getContainerUploadDirs returns a slice of the full path to the upload
// directories in container.
func (app *DdevApp) getContainerUploadDirs() []string {
	uploadDirs := app.GetUploadDirs()
	containerUploadDirs := make([]string, 0, len(uploadDirs))

	for _, uploadDir := range uploadDirs {
		containerUploadDirs = append(containerUploadDirs, app.calculateContainerUploadDirFullPath(uploadDir))
	}

	return containerUploadDirs
}

// getUploadDirsHostContainerMapping returns a slice containing host / container
// mapping separated by ":" to be used within docker-compose config.
func (app *DdevApp) getUploadDirsHostContainerMapping() []string {
	uploadDirs := app.GetUploadDirs()
	uploadDirsMapping := make([]string, 0, len(uploadDirs))

	for _, uploadDir := range uploadDirs {
		hostUploadDir := app.calculateHostUploadDirFullPath(uploadDir)

		// Exclude non existing dirs
		if !fileutil.FileExists(hostUploadDir) {
			continue
		}

		uploadDirsMapping = append(uploadDirsMapping, fmt.Sprintf(
			"%s:%s",
			hostUploadDir,
			app.calculateContainerUploadDirFullPath(uploadDir),
		))
	}

	return uploadDirsMapping
}

// getUploadDirsRelative returns a slice containing upload dirs to be used with
// Mutagen config.
func (app *DdevApp) getUploadDirsRelative() []string {
	uploadDirs := app.GetUploadDirs()
	uploadDirsMap := make([]string, 0 /*, len(uploadDirs)*/)

	for _, uploadDir := range uploadDirs {
		hostUploadDir := app.calculateHostUploadDirFullPath(uploadDir)

		// Exclude non existing dirs
		if !fileutil.FileExists(hostUploadDir) {
			continue
		}

		uploadDirsMap = append(uploadDirsMap, path.Join(app.Docroot, uploadDir))
	}

	return uploadDirsMap
}

// CreateUploadDirsIfNecessary creates the upload dirs if it doesn't exist, so we can properly
// set up bind-mounts when doing mutagen.
// There is no need to do it if mutagen is not enabled, and
// we'll just respect a symlink if it exists, and the user has to figure out the right
// thing to do with mutagen.
func (app *DdevApp) CreateUploadDirsIfNecessary() {
	for _, target := range app.GetUploadDirs() {
		if hostDir := app.calculateHostUploadDirFullPath(target); hostDir != "" && app.IsMutagenEnabled() && !fileutil.FileExists(hostDir) {
			err := os.MkdirAll(hostDir, 0755)
			if err != nil {
				util.Warning("unable to create upload directory %s: %v", hostDir, err)
			}
		}
	}
}

// validateUploadDirs validates and converts UploadDirs to app.UploadDirs
// interface or if disabled to bool false and returns nil if succeeded or an
// error if not.
// app.UploadDirs must be one of:
// - slice of string (possibly empty)
// - boolean false
func (app *DdevApp) validateUploadDirs() error {

	// Check that upload dirs aren't outside the project root.
	for _, uploadDir := range app.UploadDirs {
		if !strings.HasPrefix(app.calculateHostUploadDirFullPath(uploadDir), app.AppRoot) {
			return fmt.Errorf("invalid upload dir `%s` outside of project root `%s` found", uploadDir, app.AppRoot)
		}
	}

	return nil
}
