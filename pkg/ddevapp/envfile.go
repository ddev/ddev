package ddevapp

import (
	"github.com/joho/godotenv"
	"path/filepath"
)

// ReadEnvFile() reads the .env in the project root into a map[string]string
func ReadEnvFile(app *DdevApp) (map[string]string, error) {
	var envFileContents map[string]string
	envFileContents, err := godotenv.Read(filepath.Join(app.AppRoot, ".env"))

	if err != nil {
		return nil, err
	}
	return envFileContents, nil
}

// WriteEnvFile writes the passed map[string]string into the project-root .env file
func WriteEnvFile(app *DdevApp, envFileContents map[string]string) error {
	err := godotenv.Write(envFileContents, filepath.Join(app.AppRoot, ".env"))
	if err != nil {
		return err
	}
	return nil
}
