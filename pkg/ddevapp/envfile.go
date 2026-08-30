package ddevapp

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/joho/godotenv"
)

// envFileTarget reports whether baseName is a DDEV env file and which service it
// applies to. An empty target means every container.
//
// The name is `.env[.<service>[.<label>...]][.local]`, where a label only keeps
// files apart and `.local` only marks the file gitignored, so both are dropped
// before the service is read. `.example` files are not env files.
func envFileTarget(baseName string) (target string, ok bool) {
	if strings.HasSuffix(baseName, ".example") {
		return "", false
	}
	if baseName == ".env" || baseName == ".env.local" {
		return "", true
	}
	rest, found := strings.CutPrefix(baseName, ".env.")
	if !found {
		return "", false
	}
	target, _, _ = strings.Cut(strings.TrimSuffix(rest, ".local"), ".")
	if target == "" {
		return "", false
	}
	return target, true
}

// compareEnvFileNames orders two env file base names by the order they are
// applied in, less specific first.
func compareEnvFileNames(a, b string) int {
	aBase, aLocal := strings.CutSuffix(a, ".local")
	bBase, bLocal := strings.CutSuffix(b, ".local")
	if c := strings.Compare(aBase, bBase); c != 0 {
		return c
	}
	// A .local file overrides the file it shares a base name with.
	switch {
	case aLocal == bLocal:
		return 0
	case aLocal:
		return 1
	default:
		return -1
	}
}

// orderedEnvFilesInDir returns the full paths of the env files in dir, in the
// order they are applied. A missing directory is not an error.
func orderedEnvFilesInDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if _, ok := envFileTarget(entry.Name()); ok {
			names = append(names, entry.Name())
		}
	}
	slices.SortFunc(names, compareEnvFileNames)
	envFiles := make([]string, 0, len(names))
	for _, name := range names {
		envFiles = append(envFiles, filepath.Join(dir, name))
	}
	return envFiles, nil
}

// filterEnvFilesForTarget returns the env files from an already-resolved list
// that apply to the named service.
func filterEnvFilesForTarget(envFiles []string, target string) []string {
	var filtered []string
	for _, envFile := range envFiles {
		fileTarget, ok := envFileTarget(filepath.Base(envFile))
		if ok && (fileTarget == "" || fileTarget == target) {
			filtered = append(filtered, envFile)
		}
	}
	return filtered
}

// ReadEnvFilesForTarget merges the env files that apply to the named service,
// later files overriding earlier ones.
func (app *DdevApp) ReadEnvFilesForTarget(target string) (map[string]string, error) {
	envFiles, err := app.EnvFiles()
	if err != nil {
		return nil, err
	}
	envMap := map[string]string{}
	for _, envFile := range filterEnvFilesForTarget(envFiles, target) {
		fileMap, _, err := ReadProjectEnvFile(envFile)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("unable to read %s file: %v", envFile, err)
		}
		maps.Copy(envMap, fileMap)
	}
	return envMap, nil
}

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
