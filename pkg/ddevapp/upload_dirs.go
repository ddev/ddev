package ddevapp

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/util"
)

type UploadDirs []string

// addUploadDir adds a new upload dir if it does not already exist in the list.
func (app *DdevApp) addUploadDir(uploadDir string) {
	err := app.validateUploadDirs()
	if err != nil {
		// Should never happen
		panic(err)
	}

	if app.UploadDirs == false {
		app.UploadDirs = UploadDirs{}
	}

	for _, existingUploadDir := range app.UploadDirs.(UploadDirs) {
		if uploadDir == existingUploadDir {
			return
		}
	}

	app.UploadDirs = append(app.UploadDirs.(UploadDirs), uploadDir)
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
func (app *DdevApp) GetUploadDirs() UploadDirs {
	err := app.validateUploadDirs()
	if err != nil {
		util.Warning("Ignoring invalid upload_dirs value: %v", err)
		return UploadDirs{}
	}

	if app.UploadDirDeprecated != "" {
		uploadDirDeprecated := app.UploadDirDeprecated
		app.UploadDirDeprecated = ""
		app.addUploadDir(uploadDirDeprecated)
	}

	switch app.UploadDirs.(type) {
	case UploadDirs:
		if len(app.UploadDirs.(UploadDirs)) > 0 {
			return app.UploadDirs.(UploadDirs)
		}
		// Otherwise we go get the CMS-defined values
	case []any:
		if len(app.UploadDirs.([]any)) > 0 {
			// User provided a list of strings, convert it to UploadDirs.
			uploadDirsRaw := app.UploadDirs.([]any)
			uploadDirs := make(UploadDirs, 0, len(uploadDirsRaw))
			for _, v := range uploadDirsRaw {
				uploadDirs = append(uploadDirs, v.(string))
			}
			app.UploadDirs = uploadDirs
			return uploadDirs
		}
		// Otherwise we go get the CMS-defined values
	case bool:
		if app.UploadDirs.(bool) == false {
			return UploadDirs{}
		}
	default:
		util.Warning("app.UploadDirs is of invalid type %T", app.UploadDirs)
		return UploadDirs{}
	}

	appFuncs, ok := appTypeMatrix[app.GetType()]
	if ok && appFuncs.uploadDirs != nil {
		return appFuncs.uploadDirs(app)
	}

	return UploadDirs{}
}

// IsUploadDirsDisabled returns true if UploadDirs is disabled by the user.
func (app *DdevApp) IsUploadDirsDisabled() bool {
	return app.UploadDirs == false
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

// createUploadDirsIfNecessary creates the upload dirs if it doesn't exist, so we can properly
// set up bind-mounts when doing mutagen.
// There is no need to do it if mutagen is not enabled, and
// we'll just respect a symlink if it exists, and the user has to figure out the right
// thing to do with mutagen.
func (app *DdevApp) createUploadDirsIfNecessary() {
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
	if raw, ok := app.UploadDirs.(UploadDirs); ok {
		// User provided a list of strings, convert it to UploadDirs.
		app.UploadDirs = raw
	}

	typeOfUploadDirs := reflect.TypeOf(app.UploadDirs)
	switch {
	case typeOfUploadDirs == nil:
		// Config option is not set.
		app.UploadDirs = UploadDirs{}
	case typeOfUploadDirs.Kind() == reflect.Bool && app.UploadDirs == false:
		// bool false is fine too, means users has disabled it.
	case typeOfUploadDirs.Kind() == reflect.String:
		// User provided a string, convert it to UploadDirs.
		app.UploadDirs = UploadDirs{app.UploadDirs.(string)}
	case typeOfUploadDirs.Kind() == reflect.Slice:
		break
	default:
		// Provided value is not valid, user has to fix it.
		return fmt.Errorf("`upload_dirs` must be a string, a list of strings, or false but `%v` given", app.UploadDirs)
	}

	if dirs, ok := app.UploadDirs.(UploadDirs); ok {
		// Check upload dirs are in the project root.
		for _, uploadDir := range dirs {
			if !strings.HasPrefix(app.calculateHostUploadDirFullPath(uploadDir), app.AppRoot) {
				return fmt.Errorf("invalid upload dir `%s` outside of project root `%s` found", uploadDir, app.AppRoot)
			}
		}
	}

	return nil
}
