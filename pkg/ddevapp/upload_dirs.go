package ddevapp

import (
	"fmt"
	"os"
	"path"
	"reflect"

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
func (app *DdevApp) GetUploadDir() string {
	uploadDirs := app.GetUploadDirs()
	if len(uploadDirs) > 0 {
		return uploadDirs[0]
	}

	return ""
}

// GetUploadDirs returns the upload (public files) directories.
func (app *DdevApp) GetUploadDirs() UploadDirs {
	err := app.validateUploadDirs()
	if err != nil {
		// Should never happen
		panic(err)
	}

	if app.UploadDirDeprecated != "" {
		uploadDirDeprecated := app.UploadDirDeprecated
		app.UploadDirDeprecated = ""
		app.addUploadDir(uploadDirDeprecated)
	}

	if app.UploadDirs == false {
		return UploadDirs{}
	}

	if len(app.UploadDirs.(UploadDirs)) > 0 {
		return app.UploadDirs.(UploadDirs)
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
		return path.Join(app.AppRoot, app.Docroot, uploadDir)
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
		return path.Join("/var/www/html", app.Docroot, uploadDir)
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

// validateUploadDirs validates and converts UploadDirs to a app.UploadDirs
// interface or if disabled to bool false and returns nil if succeeded or an
// error if not.
func (app *DdevApp) validateUploadDirs() error {
	if _, ok := app.UploadDirs.(UploadDirs); ok {
		// Conversion was already done, nothing to do.
		return nil
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
		// User provided a list of strings, convert it to UploadDirs.
		uploadDirsRaw := app.UploadDirs.([]any)
		uploadDirs := make(UploadDirs, 0, len(uploadDirsRaw))
		for _, v := range uploadDirsRaw {
			uploadDirs = append(uploadDirs, v.(string))
		}
		app.UploadDirs = uploadDirs
	default:
		// Provided value is not valid, user has to fix it.
		return fmt.Errorf("`upload_dirs` must be a string, a list of strings, or false but `%v` given", app.UploadDirs)
	}

	return nil
}
