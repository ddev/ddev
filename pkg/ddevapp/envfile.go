package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/joho/godotenv"
	"path/filepath"
	"regexp"
	"strings"
)

// ReadEnvFile() reads the .env in the project root into a envText and envMap
// The map has the envFile content, but without comments
func ReadEnvFile(app *DdevApp) (envMap map[string]string, envText string, err error) {
	envFilePath := filepath.Join(app.AppRoot, ".env")
	envText, err = fileutil.ReadFileIntoString(envFilePath)
	envMap, err = godotenv.Read(envFilePath)

	if err != nil {
		return envMap, envText, err
	}
	return envMap, envText, nil
}

// WriteEnvFile writes the passed envText into the project-root .env file
// with all items in envMap changed in envText there
func WriteEnvFile(app *DdevApp, envMap map[string]string, envText string) error {
	envFilePath := filepath.Join(app.AppRoot, ".env")
	for k, v := range envMap {
		// If the item is already in envText, use regex to replace it
		// otherwise, append it to the envText.
		if strings.Contains(envText, k+"=") {
			exp := regexp.MustCompile(fmt.Sprintf(`%s=(.*)`, k))
			envText = exp.ReplaceAllString(envText, fmt.Sprintf(`%s="%s"`, k, v))
		} else {
			envText = strings.TrimSuffix(envText, "\n")
			envText = fmt.Sprintf("%s\n%s=%s\n", envText, k, v)
		}
	}
	err := fileutil.TemplateStringToFile(envText, nil, envFilePath)
	if err != nil {
		return err
	}
	return nil
}
