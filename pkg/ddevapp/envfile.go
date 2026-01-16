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
// returns
// - envMap (map of items found)
// - envText (plain text unaltered of existing env file
// - error/nil
func ReadProjectEnvFile(envFilePath string) (envMap map[string]string, envText string, err error) {
	// envFilePath := filepath.Join(app.AppRoot, ".env")
	envText, _ = fileutil.ReadFileIntoString(envFilePath)
	// godotenv is not perfect, there can be some edge cases with escaping
	// such as https://github.com/joho/godotenv/issues/225
	envMap, err = godotenv.Read(envFilePath)

	return envMap, envText, err
}

// WriteProjectEnvFile writes the passed envText into the envFilePath .env file
// changing items in envMap changed in envText there
func WriteProjectEnvFile(envFilePath string, envMap map[string]string, envText string) error {
	for k, v := range envMap {
		v = EscapeEnvFileValue(v)
		// If the item is already in envText, use regex to replace it
		// otherwise, append it to the envText.
		// (^|[\r\n]+) - first group $1 matches the start of a line or newline characters
		// #*[ \t]* - matches optional comments with spaces/tabs, i.e. find lines like '# FOO=BAR'
		// (%s) - second group $2 matches the variable name (QuoteMeta escapes dots and other
		//        regex special chars, e.g. for CodeIgniter's "database.default.hostname")
		// [ \t]*=[ \t]* - matches equals sign with optional spaces/tabs
		exp := regexp.MustCompile(fmt.Sprintf(`(^|[\r\n]+)#*[ \t]*(%s)[ \t]*=[ \t]*(.*)`, regexp.QuoteMeta(k)))
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

// EscapeEnvFileValue escapes the value so it can be used in an .env file
// The value is wrapped in double quotes for correct work with spaces.
func EscapeEnvFileValue(value string) string {
	value = strings.NewReplacer(
		// Escape all dollar signs so they are not interpreted as bash variables
		`$`, `\$`,
		// Escape all double quotes since we wrap the value in double quotes
		`"`, `\"`,
	).Replace(value)
	// Wrap the value in double quotes
	return `"` + value + `"`
}
