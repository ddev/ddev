package ddevapp

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/joho/godotenv"
)

// ReadProjectEnvFile reads the .env in the project root into a envText and envMap
// The map has the envFile content, but without comments
func ReadProjectEnvFile(envFilePath string) (envMap map[string]string, envText string, err error) {
	// envFilePath := filepath.Join(app.AppRoot, ".env")
	envText, _ = fileutil.ReadFileIntoString(envFilePath)
	// godotenv is not perfect, there can be some edge cases with escaping
	// such as https://github.com/joho/godotenv/issues/225
	envMap, err = godotenv.Read(envFilePath)

	return envMap, envText, err
}

// WriteProjectEnvFile writes the passed envText into the project-root .env file
// with all items in envMap changed in envText there
func WriteProjectEnvFile(envFilePath string, envMap map[string]string, envText string) error {
	// envFilePath := filepath.Join(app.AppRoot, ".env")
	for k, v := range envMap {
		// Escape double quotes in the value, since we wrap it in double quotes
		v = `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
		// If the item is already in envText, use regex to replace it
		// otherwise, append it to the envText.
		// (^|[\r\n]+) - first group $1 matches the start of a line or newline characters
		// #*\s* - matches optional comments with whitespaces, i.e. find lines like '# FOO=BAR'
		// (%s) - second group $2 matches the variable name
		exp := regexp.MustCompile(fmt.Sprintf(`(^|[\r\n]+)#*\s*(%s)=(.*)`, k))
		if exp.MatchString(envText) {
			// To insert a literal $ in the output, use $$ in the template.
			// See https://pkg.go.dev/regexp?utm_source=godoc#Regexp.Expand
			v = strings.ReplaceAll(v, `$`, `$$`)
			// Remove comments with whitespaces here using only $1 and $2 groups
			envText = exp.ReplaceAllString(envText, fmt.Sprintf(`$1$2=%s`, v))
		} else {
			envText = strings.TrimSuffix(envText, "\n")
			if envText != "" {
				envText = fmt.Sprintf("%s\n%s=%s\n", envText, k, v)
			} else {
				envText = fmt.Sprintf("%s=%s\n", k, v)
			}
		}
	}
	err := fileutil.TemplateStringToFile(envText, nil, envFilePath)
	if err != nil {
		return err
	}
	return nil
}
