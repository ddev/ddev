package ddevapp

import (
	"fmt"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/joho/godotenv"
	"regexp"
	"strings"
)

// ReadProjectEnvFile reads the .env in the project root into a envText and envMap
// The map has the envFile content, but without comments
func ReadProjectEnvFile(envFilePath string) (envMap map[string]string, envText string, err error) {
	// envFilePath := filepath.Join(app.AppRoot, ".env")
	envText, _ = fileutil.ReadFileIntoString(envFilePath)
	envMap, err = godotenv.Read(envFilePath)

	return envMap, envText, err
}

// WriteProjectEnvFile writes the passed envText into the project-root .env file
// with all items in envMap changed in envText there
func WriteProjectEnvFile(envFilePath string, envMap map[string]string, envText string) error {
	// envFilePath := filepath.Join(app.AppRoot, ".env")
	for k, v := range envMap {
		// If the item is already in envText, use regex to replace it
		// otherwise, append it to the envText.
		exp := regexp.MustCompile(fmt.Sprintf(`((^|[\r\n]+)%s)=(.*)`, k))
		if exp.MatchString(envText) {
			envText = exp.ReplaceAllString(envText, fmt.Sprintf(`$1="%s"`, v))
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
