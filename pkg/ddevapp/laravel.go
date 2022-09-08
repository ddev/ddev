package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/fileutil"
	"os"
	"path/filepath"
)

const (
	// WarnTypeAbsent warns if type is absent
	WarnTypeAbsent = iota
	// WarnTypeNotConfigured warns if type not configured
	WarnTypeNotConfigured = iota
)

// isLaravelApp returns true if the app is of type laravel
func isLaravelApp(app *DdevApp) bool {
	return fileutil.FileExists(filepath.Join(app.AppRoot, "artisan"))
}

func laravelPostStartAction(app *DdevApp) error {
	envContents, err := ReadEnvFile(app)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Unable to read .env file: %v", err)
	}
	if os.IsNotExist(err) {
		err = fileutil.CopyFile(filepath.Join(app.AppRoot, ".env.example"), filepath.Join(app.AppRoot, ".env"))
		if err != nil {
			return err
		}
		envContents, err = ReadEnvFile(app)
		if err != nil {
			return err
		}
	}
	envContents["DB_HOST"] = "db"
	envContents["DB_PORT"] = "3306"
	envContents["DB_DATABASE"] = "db"
	envContents["DB_USERNAME"] = "db"
	envContents["DB_PASSWORD"] = "db"
	envContents["DB_CONNECTION"] = "ddev"
	err = WriteEnvFile(app, envContents)
	if err != nil {
		return err
	}

	return nil
}
