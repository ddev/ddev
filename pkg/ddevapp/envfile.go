package ddevapp

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ddev/ddev/pkg/dotenv"
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
	return dotenv.ReadFiles(filterEnvFilesForTarget(envFiles, target)...)
}

// envFileNamesForTarget returns the names the env files for the named service
// set, see dotenv.ReadNames.
func (app *DdevApp) envFileNamesForTarget(target string) ([]string, error) {
	envFiles, err := app.EnvFiles()
	if err != nil {
		return nil, err
	}
	return dotenv.ReadNames(filterEnvFilesForTarget(envFiles, target)...)
}

// SetHostEnvFromFiles sets the values of the env files that target every
// service, .env and .env.local, global ones included, on DDEV's own process,
// so host commands it starts inherit them. As in compose, a variable already
// set on the host keeps its value. The returned function unsets them again,
// so another project handled by the same ddev run does not see them.
func (app *DdevApp) SetHostEnvFromFiles() (unset func(), err error) {
	values, err := app.ReadEnvFilesForTarget("")
	if err != nil {
		return func() {}, err
	}
	var names []string
	for name, value := range values {
		if _, set := os.LookupEnv(name); !set {
			_ = os.Setenv(name, value)
			names = append(names, name)
		}
	}
	return func() {
		for _, name := range names {
			_ = os.Unsetenv(name)
		}
	}, nil
}

// ReadEnvFileText returns the text of an env file without parsing it, for
// writers that edit an application's own .env, which compose never reads.
func ReadEnvFileText(envFilePath string) (string, error) {
	b, err := os.ReadFile(envFilePath)
	return string(b), err
}

// WriteProjectEnvFile writes envText to envFilePath with each item in envMap
// set, see dotenv.Update. A file that already has that content is left
// untouched, so file watchers and Mutagen only see a start that changed it.
func WriteProjectEnvFile(envFilePath string, envMap map[string]string, envText string) error {
	newText := dotenv.Update(envText, envMap)
	// envText can come from .env.example, so compare with the file itself
	if current, err := os.ReadFile(envFilePath); err == nil && string(current) == newText {
		return nil
	}
	return os.WriteFile(envFilePath, []byte(newText), 0644)
}
