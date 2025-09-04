package ddevapp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	goexec "os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	github2 "github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	dockerContainer "github.com/docker/docker/api/types/container"
	dockerMount "github.com/docker/docker/api/types/mount"
	dockerStrslice "github.com/docker/docker/api/types/strslice"
	"github.com/google/go-github/v72/github"
	"github.com/otiai10/copy"
	"go.yaml.in/yaml/v3"
)

const AddonMetadataDir = "addon-metadata"

// PHP strict mode template - equivalent to bash 'set -eu -o pipefail'
const phpStrictModeTemplate = `<?php
// PHP strict error handling equivalent to bash 'set -eu -o pipefail'
error_reporting(E_ALL);
ini_set('display_errors', 1);
set_error_handler(function($severity, $message, $file, $line) {
    throw new ErrorException($message, 0, $severity, $file, $line);
});
?>`

// Shell script template for PHP action execution
const phpActionShellScriptTemplate = `
# First create and validate the original PHP action (before strict mode)
cat > /tmp/original-script.php << 'DDEV_PHP_ORIGINAL_EOF'
%s
DDEV_PHP_ORIGINAL_EOF

# Validate original PHP syntax - exit early if invalid
# Suppress success output but preserve error output
php -l /tmp/original-script.php > /dev/null
original_syntax_check_exit_code=$?
if [ $original_syntax_check_exit_code -ne 0 ]; then
    # Re-run to show error output to user
    php -l /tmp/original-script.php
    echo "PHP syntax validation failed on original action"
    exit $original_syntax_check_exit_code
fi

# If original syntax is valid, create the script with strict mode and execute
cat > /tmp/addon-script.php << 'DDEV_PHP_EOF'
%s
DDEV_PHP_EOF

# Execute the script with strict mode
cd /var/www/html/.ddev
php /tmp/addon-script.php
`

// Format of install.yaml
type InstallDesc struct {
	// Name must be unique in a project; it will overwrite any existing add-on with the same name.
	Name                  string            `yaml:"name"`
	ProjectFiles          []string          `yaml:"project_files"`
	GlobalFiles           []string          `yaml:"global_files,omitempty"`
	DdevVersionConstraint string            `yaml:"ddev_version_constraint,omitempty"`
	Dependencies          []string          `yaml:"dependencies,omitempty"`
	PreInstallActions     []string          `yaml:"pre_install_actions,omitempty"`
	PostInstallActions    []string          `yaml:"post_install_actions,omitempty"`
	RemovalActions        []string          `yaml:"removal_actions,omitempty"`
	YamlReadFiles         map[string]string `yaml:"yaml_read_files"`
	Image                 string            `yaml:"image,omitempty"`
}

// format of the add-on manifest file
type AddonManifest struct {
	Name           string   `yaml:"name"`
	Repository     string   `yaml:"repository"`
	Version        string   `yaml:"version"`
	Dependencies   []string `yaml:"dependencies,omitempty"`
	InstallDate    string   `yaml:"install_date"`
	ProjectFiles   []string `yaml:"project_files"`
	GlobalFiles    []string `yaml:"global_files"`
	RemovalActions []string `yaml:"removal_actions"`
}

// GetInstalledAddons returns a list of the installed add-ons
func GetInstalledAddons(app *DdevApp) []AddonManifest {
	metadataDir := app.GetConfigPath(AddonMetadataDir)
	err := os.MkdirAll(metadataDir, 0755)
	if err != nil {
		util.Failed("Error creating metadata directory: %v", err)
	}
	// Read the contents of the .ddev/addon-metadata directory (directories)
	dirs, err := os.ReadDir(metadataDir)
	if err != nil {
		util.Failed("Error reading metadata directory: %v", err)
	}
	manifests := []AddonManifest{}

	// Loop through the directories in the .ddev/addon-metadata directory
	for _, d := range dirs {
		// Check if the file is a directory
		if d.IsDir() {
			// Read the contents of the manifest file
			manifestFile := filepath.Join(metadataDir, d.Name(), "manifest.yaml")
			manifestBytes, err := os.ReadFile(manifestFile)
			if err != nil {
				util.Warning("No manifest file found at %s: %v", manifestFile, err)
				continue
			}

			// Parse the manifest file
			var manifest AddonManifest
			err = yaml.Unmarshal(manifestBytes, &manifest)
			if err != nil {
				util.Failed("Unable to parse manifest file: %v", err)
			}
			manifests = append(manifests, manifest)
		}
	}
	return manifests
}

// GetInstalledAddonNames returns a list of the names of installed add-ons
func GetInstalledAddonNames(app *DdevApp) []string {
	manifests := GetInstalledAddons(app)
	names := []string{}
	for _, manifest := range manifests {
		names = append(names, manifest.Name)
	}
	return names
}

// GetInstalledAddonProjectFiles returns a list of project files installed by add-ons
func GetInstalledAddonProjectFiles(app *DdevApp) []string {
	manifests := GetInstalledAddons(app)
	uniqueFilesMap := make(map[string]struct{})
	for _, manifest := range manifests {
		for _, file := range manifest.ProjectFiles {
			uniqueFilesMap[filepath.Join(app.AppConfDir(), file)] = struct{}{}
		}
	}
	uniqueFiles := make([]string, 0, len(uniqueFilesMap))
	for file := range uniqueFilesMap {
		uniqueFiles = append(uniqueFiles, file)
	}
	return uniqueFiles
}

// ProcessAddonAction takes a stanza from yaml exec section and executes it, optionally in a container.
func ProcessAddonAction(action string, installDesc InstallDesc, app *DdevApp, verbose bool) error {
	if app == nil {
		return fmt.Errorf("app is required to ProcessAddonAction")
	}
	// Check if the action starts with <?php
	if strings.HasPrefix(strings.TrimSpace(action), "<?php") {
		return processPHPAction(action, installDesc, app, verbose)
	}
	return processBashHostAction(action, installDesc, app, verbose)
}

// processBashHostAction executes a bash action on the host system
func processBashHostAction(action string, installDesc InstallDesc, app *DdevApp, verbose bool) error {
	env, err := getInjectedEnvForBash(app, installDesc)
	if err != nil {
		return fmt.Errorf("unable to get injected env for bash: %v", err)
	}
	if env != "" {
		action = env + "\n" + action
	}
	// Default behavior for bash actions
	action = "set -eu -o pipefail\n" + action
	t, err := template.New("ProcessAddonAction").Funcs(getTemplateFuncMap()).Parse(action)
	if err != nil {
		return fmt.Errorf("could not parse action '%s': %v", action, err)
	}

	yamlMap := make(map[string]interface{})
	yamlMap["DdevGlobalConfig"], err = util.YamlFileToMap(globalconfig.GetGlobalConfigPath())
	if err != nil {
		util.Warning("Unable to read file %s: %v", globalconfig.GetGlobalConfigPath(), err)
	}

	for name, f := range installDesc.YamlReadFiles {
		fullPath := filepath.Join(app.GetAppRoot(), os.ExpandEnv(f))
		yamlMap[name], err = util.YamlFileToMap(fullPath)
		if err != nil {
			util.Warning("Unable to import yaml file %s: %v", fullPath, err)
		}
	}
	// Get project config with overrides
	var projectConfigMap map[string]interface{}
	if b, err := yaml.Marshal(app); err != nil {
		util.Warning("Unable to marshal app: %v", err)
	} else if err = yaml.Unmarshal(b, &projectConfigMap); err != nil {
		util.Warning("Unable to unmarshal app: %v", err)
	} else {
		yamlMap["DdevProjectConfig"] = projectConfigMap
	}

	dict, err := util.YamlToDict(yamlMap)
	if err != nil {
		return fmt.Errorf("unable to YamlToDict: %v", err)
	}

	var doc bytes.Buffer
	err = t.Execute(&doc, dict)
	if err != nil {
		return fmt.Errorf("could not parse/execute action '%s': %v", action, err)
	}
	action = doc.String()

	desc := GetAddonDdevDescription(action)
	if verbose {
		action = "set -x; " + action
	}
	out, err := exec.RunHostCommand(util.FindBashPath(), "-c", action)
	if err != nil {
		warningCode := GetAddonDdevWarningExitCode(action)
		if warningCode > 0 {
			var exitErr *goexec.ExitError
			if errors.As(err, &exitErr) {
				// Get the exit code
				exitCode := exitErr.ExitCode()
				if exitCode == warningCode {
					if desc != "" {
						util.Warning("%s %s (bash)", "\U000026A0\U0000FE0F", desc)
					}
					err = nil
				}
			}
		}
		if err != nil {
			if desc != "" {
				util.Warning("%c %s (bash)", '\U0001F44E', desc)
			}
			err = fmt.Errorf("unable to run bash action %v: %v, output=%s", action, err, out)
		}
	} else {
		if desc != "" {
			util.Success("%c %s (bash)", '\U0001F44D', desc)
		}
	}
	if len(out) > 0 {
		util.Warning(out)
	}
	return err
}

// getInjectedEnvForBash returns bash export string for env variables
// that will be used in PreInstallActions and PostInstallActions
func getInjectedEnvForBash(app *DdevApp, installDesc InstallDesc) (string, error) {
	if app == nil {
		return "", nil
	}
	envFile := app.GetConfigPath(".env." + installDesc.Name)
	envMap, _, err := ReadProjectEnvFile(envFile)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("unable to read %s file: %v", envFile, err)
	}
	if len(envMap) == 0 {
		return "", nil
	}
	injectedEnv := "export"
	for k, v := range envMap {
		// Escape all spaces and dollar signs
		v = strings.ReplaceAll(strings.ReplaceAll(v, `$`, `\$`), ` `, `\ `)
		injectedEnv = injectedEnv + fmt.Sprintf(" %s=%s ", k, v)
	}
	return injectedEnv, nil
}

// processPHPAction executes a PHP action in a container
func processPHPAction(action string, installDesc InstallDesc, app *DdevApp, verbose bool) error {
	// Extract description before processing
	desc := GetAddonDdevDescription(action)

	// Store the original action for validation
	originalAction := action

	image := installDesc.Image
	// Use the default ddev-webserver as the image if none is specified
	if image == "" {
		image = docker.GetWebImage()
	}

	// Validate included/required files on the host (since we need to read from filesystem)
	err := validatePHPIncludesAndRequires(action, app, image)
	if err != nil {
		return fmt.Errorf("PHP include/require validation error: %v", err)
	}

	// Inject PHP strict error handling
	action = injectPHPStrictMode(action)

	// Create configuration files for PHP action access (optional for tests and removal actions)
	err = createConfigurationFiles(app)
	if err != nil {
		return fmt.Errorf("failed to create configuration files for PHP action: %w", err)
	}

	// Create a shell script that validates original PHP syntax first, then executes with strict mode
	shellScript := fmt.Sprintf(phpActionShellScriptTemplate, originalAction, action)

	cmd := []string{"sh", "-c", shellScript}

	// Build environment variables array with standard DDEV variables
	env, err := buildPHPActionEnvironment(app, installDesc, verbose)
	if err != nil {
		return fmt.Errorf("failed to build PHP action environment: %v", err)
	}

	uidStr, _, _ := util.GetContainerUIDGid()

	config := &dockerContainer.Config{
		Image:      image,
		Cmd:        dockerStrslice.StrSlice(cmd),
		WorkingDir: "/var/www/html/.ddev",
		Env:        env,
		User:       uidStr,
	}

	hostConfig := &dockerContainer.HostConfig{
		Mounts: []dockerMount.Mount{
			{
				Type:   dockerMount.TypeBind,
				Source: app.AppRoot,
				Target: "/var/www/html",
			},
		},
	}

	_, out, err := dockerutil.RunSimpleContainerExtended("php-action-"+util.RandString(6), config, hostConfig, true, false)
	out = strings.TrimSpace(out)

	if err != nil {
		if desc != "" {
			util.Warning("%c %s (PHP)", '\U0001F44E', desc) // 👎 error emoji
		}
		// Include output in error message for debugging
		if out != "" {
			return fmt.Errorf("PHP script failed: %v - Output: %s", err, out)
		}
		return fmt.Errorf("PHP script failed: %v", err)
	}

	// Show description on success
	if desc != "" {
		util.Success("%c %s (PHP)", '\U0001F44D', desc) // 👍 success emoji
	}
	// Display captured output
	if out != "" {
		output.UserOut.Println(out)
	}
	return nil
}

// createConfigurationFiles creates temporary YAML configuration files for PHP actions
func createConfigurationFiles(app *DdevApp) error {
	configDir := filepath.Join(app.AppConfDir(), ".ddev-config")

	// Create the .ddev-config directory
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create config directory %s: %v", configDir, err)
	}

	// Generate project configuration YAML
	projectConfigYAML, err := app.GetProcessedProjectConfigYAML()
	if err != nil {
		return fmt.Errorf("failed to generate project configuration: %v", err)
	}

	// Write project configuration file
	projectConfigPath := filepath.Join(configDir, "project_config.yaml")
	err = os.WriteFile(projectConfigPath, projectConfigYAML, 0644)
	if err != nil {
		return fmt.Errorf("failed to write project config file %s: %v", projectConfigPath, err)
	}

	// Generate global configuration YAML
	globalConfigYAML, err := globalconfig.GetGlobalConfigYAML()
	if err != nil {
		return fmt.Errorf("failed to generate global configuration: %v", err)
	}

	// Write global configuration file
	globalConfigPath := filepath.Join(configDir, "global_config.yaml")
	err = os.WriteFile(globalConfigPath, globalConfigYAML, 0644)
	if err != nil {
		return fmt.Errorf("failed to write global config file %s: %v", globalConfigPath, err)
	}

	return nil
}

// CleanupConfigurationFiles removes temporary configuration files created for PHP actions
func (app *DdevApp) CleanupConfigurationFiles() error {
	configDir := filepath.Join(app.AppConfDir(), ".ddev-config")
	if fileutil.FileExists(configDir) {
		return os.RemoveAll(configDir)
	}
	return nil
}

// buildPHPActionEnvironment creates the environment variables for PHP actions
func buildPHPActionEnvironment(app *DdevApp, installDesc InstallDesc, verbose bool) ([]string, error) {
	// Database family for connection URLs
	dbFamily := "mysql"
	if app.Database.Type == "postgres" {
		dbFamily = "postgres"
	}

	env := []string{
		"DDEV_APPROOT=/var/www/html",
		"DDEV_DOCROOT=" + app.GetDocroot(),
		"DDEV_PROJECT_TYPE=" + app.Type,
		"DDEV_SITENAME=" + app.Name,
		"DDEV_PROJECT=" + app.Name,
		"DDEV_PHP_VERSION=" + app.PHPVersion,
		"DDEV_WEBSERVER_TYPE=" + app.WebserverType,
		"DDEV_DATABASE=" + app.Database.Type + ":" + app.Database.Version,
		"DDEV_DATABASE_FAMILY=" + dbFamily,
		"DDEV_FILES_DIRS=" + strings.Join(app.GetUploadDirs(), ","),
		"DDEV_MUTAGEN_ENABLED=" + strconv.FormatBool(app.IsMutagenEnabled()),
		"DDEV_VERSION=" + versionconstants.DdevVersion,
		"DDEV_TLD=" + app.ProjectTLD,
		"IS_DDEV_PROJECT=true",
	}

	if verbose {
		env = append(env, "DDEV_VERBOSE=true")
	}

	// Add all environment variables from the .ddev/.env.<addon-name>
	envFile := app.GetConfigPath(".env." + installDesc.Name)
	envMap, _, err := ReadProjectEnvFile(envFile)
	if err != nil && !os.IsNotExist(err) {
		return env, fmt.Errorf("unable to read %s file: %v", envFile, err)
	}
	for k, v := range envMap {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env, nil
}

// injectPHPStrictMode adds PHP strict error handling to the action
func injectPHPStrictMode(action string) string {
	// Add strict mode - prepend complete PHP block or replace existing <?php with enhanced version
	if !strings.HasPrefix(strings.TrimSpace(action), "<?php") {
		return phpStrictModeTemplate + "\n" + action
	}

	// If it already starts with <?php, replace the opening with strict mode and continue with original content
	lines := strings.Split(action, "\n")
	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		if firstLine == "<?php" {
			// Replace the <?php line with complete strict mode block, but without the closing ?>
			strictModeLines := strings.Split(phpStrictModeTemplate, "\n")
			// Remove the closing ?> from strict mode so it continues seamlessly
			strictModeWithoutClose := strings.Join(strictModeLines[:len(strictModeLines)-1], "\n")
			// Combine strict mode with the rest of the original action (skipping the original <?php line)
			return strictModeWithoutClose + "\n" + strings.Join(lines[1:], "\n")
		}
	}

	return action
}

// validatePHPSyntax validates PHP syntax by running php -l in a container
// This is used only for validating included/required files
func validatePHPSyntax(phpCode string, image string) error {
	// Use the provided image or default
	if image == "" {
		image = docker.GetWebImage()
	}

	// Create a shell script that writes the PHP code and validates it
	// Exit with error code if syntax check fails
	shellScript := fmt.Sprintf(`
cat > /tmp/validate-script.php << 'DDEV_PHP_VALIDATE_EOF'
%s
DDEV_PHP_VALIDATE_EOF

# Run PHP syntax check - suppress success output but preserve error output
php -l /tmp/validate-script.php > /dev/null
exit_code=$?
if [ $exit_code -ne 0 ]; then
    # Re-run to show error output to user
    php -l /tmp/validate-script.php
    echo "PHP syntax validation failed"
    exit $exit_code
fi
`, phpCode)

	_, out, err := dockerutil.RunSimpleContainer(image, "php-validate-"+util.RandString(6), []string{"sh", "-c", shellScript}, []string{}, []string{}, []string{}, "", true, false, map[string]string{"com.ddev.site-name": ""}, nil, nil)
	out = strings.TrimSpace(out)

	if err != nil {
		// Include validation output in error message for debugging
		if out != "" {
			return fmt.Errorf("PHP syntax validation failed: %s", out)
		}
		return fmt.Errorf("PHP syntax validation failed: %v", err)
	}

	return nil
}

// validatePHPIncludesAndRequires validates PHP syntax of included/required files
func validatePHPIncludesAndRequires(phpCode string, app *DdevApp, image string) error {
	// First check if this is actually PHP code by looking for standard PHP opening tags
	if !strings.Contains(phpCode, "<?php") {
		return nil // Not PHP code, no validation needed
	}

	// Extract include/require statements with proper regex
	// Matches: include, include_once, require, require_once followed by file references
	// Fixed: OLD pattern `.*\.(php|inc)` would truncate at first .php, missing closing quotes
	// Example: "require 'redis/scripts/setup-drupal-settings.php';" would only match up to ".php"
	// NEW pattern captures complete statements including semicolons and closing quotes
	includePattern := `(include|include_once|require|require_once)[[:space:]]+.*\.(php|inc)[^;]*;?`
	matches := nodeps.GrepStringInBuffer(phpCode, includePattern)

	if len(matches) == 0 {
		return nil // No includes/requires found
	}

	for _, match := range matches {
		// Extract potential file paths from the include/require statement
		// Handle common PHP include patterns:
		// - include 'file.php';
		// - require_once("config.inc");
		// - include __DIR__ . '/helper.php';
		filePaths := extractPHPFilePaths(match)

		for _, filePath := range filePaths {
			if filePath == "" {
				continue
			}

			// Skip dynamic includes (variables, function calls, etc.)
			if containsDynamicContent(filePath) {
				util.Warning("Skipping validation of dynamic include: %s", filePath)
				continue
			}

			// Try to locate and validate the file
			err := validateIncludedFile(filePath, app, image)
			if err != nil {
				return fmt.Errorf("validation failed for included file %s: %w", filePath, err)
			}
		}
	}

	return nil
}

// extractPHPFilePaths extracts potential PHP file paths from an include/require statement
func extractPHPFilePaths(statement string) []string {
	var paths []string

	// Look for quoted strings that end with .php or .inc
	quotedPattern := `['"]([^'"]*\.(php|inc))['"]`
	matches := nodeps.GrepStringInBuffer(statement, quotedPattern)

	for _, match := range matches {
		// Extract the path from within quotes
		if start := strings.IndexAny(match, `'"`); start != -1 {
			quote := match[start]
			if end := strings.IndexByte(match[start+1:], quote); end != -1 {
				path := match[start+1 : start+1+end]
				if path != "" {
					paths = append(paths, path)
				}
			}
		}
	}

	return paths
}

// containsDynamicContent checks if a file path contains dynamic elements
func containsDynamicContent(filePath string) bool {
	// Skip paths with variables, function calls, or concatenation
	dynamicPatterns := []string{"$", "__DIR__", "__FILE__", "dirname", "realpath", "getcwd"}

	// Fixed: OLD code had "." in dynamicPatterns, incorrectly flagging normal file extensions
	// Example: "setup-drupal-settings.php" was flagged as dynamic due to the ".php" extension
	// NEW code only flags concatenation patterns with spaces around the dot operator
	if strings.Contains(filePath, " . ") || strings.Contains(filePath, ". ") || strings.Contains(filePath, " .") {
		return true
	}

	for _, pattern := range dynamicPatterns {
		if strings.Contains(filePath, pattern) {
			return true
		}
	}

	return false
}

// validateIncludedFile locates and validates a PHP file referenced by include/require
func validateIncludedFile(filePath string, app *DdevApp, image string) error {
	// Try multiple potential locations for the file
	searchPaths := []string{
		filepath.Join(app.AppConfDir(), filePath),       // Relative to .ddev/
		filepath.Join(app.AppRoot, filePath),            // Relative to project root
		filepath.Join(app.AppConfDir(), "..", filePath), // Relative to .ddev parent
	}

	var fullPath string
	var found bool

	for _, searchPath := range searchPaths {
		if fileutil.FileExists(searchPath) {
			fullPath = searchPath
			found = true
			break
		}
	}

	if !found {
		util.Warning("Include/require file not found for validation: %s", filePath)
		return nil // Don't fail validation for missing files - they might be created dynamically
	}

	// Read and validate the included file
	includedContent, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read included file: %w", err)
	}

	// Only validate files that appear to contain PHP code
	content := string(includedContent)
	if strings.Contains(content, "<?php") || filepath.Ext(fullPath) == ".php" {
		err = validatePHPSyntax(content, image)
		if err != nil {
			return fmt.Errorf("PHP syntax error in included file: %w", err)
		}
	}

	return nil
}

// GetAddonDdevDescription returns what follows #ddev-description: in any line in action
func GetAddonDdevDescription(action string) string {
	descLines := nodeps.GrepStringInBuffer(action, `[\r\n]*#ddev-description:.*[\r\n]+`)
	if len(descLines) > 0 {
		d := strings.Split(descLines[0], ":")
		if len(d) > 1 {
			return strings.Trim(d[1], "\r\n\t")
		}
	}
	return ""
}

// GetAddonDdevWarningExitCode returns the integer following #ddev-warning-exit-code in the last match in action
// If no matches are found or the value is not an integer, returns 0
func GetAddonDdevWarningExitCode(action string) int {
	warnLines := nodeps.GrepStringInBuffer(action, `[\r\n]*#ddev-warning-exit-code:[ ]*[1-9][0-9]*[\r\n]+`)
	if len(warnLines) > 0 {
		// Get the last match if there are multiple
		lastLine := warnLines[len(warnLines)-1]
		parts := strings.Split(lastLine, ":")
		if len(parts) > 1 {
			codeStr := strings.Trim(parts[1], "\r\n\t ")
			// Try to convert to integer
			if code, err := strconv.Atoi(codeStr); err == nil {
				return code
			}
		}
	}
	return 0
}

// ListAvailableAddons lists the add-ons that are listed on github
func ListAvailableAddons(officialOnly bool) ([]*github.Repository, error) {
	client := github2.GetGithubClient(context.Background())
	q := "topic:ddev-get fork:true"
	if officialOnly {
		q = q + " org:" + globalconfig.DdevGithubOrg
	}

	opts := &github.SearchOptions{Sort: "updated", Order: "desc", ListOptions: github.ListOptions{PerPage: 200}}
	var allRepos []*github.Repository
	for {
		repos, resp, err := client.Search.Repositories(context.Background(), q, opts)
		if err != nil {
			msg := fmt.Sprintf("Unable to get list of available services: %v", err)
			if resp != nil {
				msg = msg + fmt.Sprintf(" rateinfo=%v", resp.Rate)
			}
			return nil, fmt.Errorf("%s", msg)
		}
		allRepos = append(allRepos, repos.Repositories...)
		if resp.NextPage == 0 {
			break
		}

		// Set the next page number for the next request
		opts.Page = resp.NextPage
	}
	out := ""
	for _, r := range allRepos {
		out = out + fmt.Sprintf("%s: %s\n", r.GetFullName(), r.GetDescription())
	}
	if len(allRepos) == 0 {
		return nil, fmt.Errorf("no add-ons found")
	}
	return allRepos, nil
}

// RemoveAddon removes an addon, taking care to respect #ddev-generated
// addonName can be the "Name", or the full "Repository" like ddev/ddev-redis, or
// the final par of the repository name like ddev-redis
func RemoveAddon(app *DdevApp, addonName string, verbose bool, skipRemovalActions bool) error {
	if addonName == "" {
		return fmt.Errorf("no add-on name specified for removal")
	}

	manifests, err := GatherAllManifests(app)
	if err != nil {
		util.Failed("Unable to gather all manifests: %v", err)
	}

	var manifestData AddonManifest
	var ok bool

	if manifestData, ok = manifests[addonName]; !ok {
		util.Failed("The add-on '%s' does not seem to have a manifest file; please upgrade it.\nUse `ddev add-on list --installed` to see installed add-ons.\n", addonName)
	}

	// Execute any removal actions
	if !skipRemovalActions {
		for i, action := range manifestData.RemovalActions {
			err = ProcessAddonAction(action, InstallDesc{}, app, verbose)
			if err != nil {
				desc := GetAddonDdevDescription(action)
				util.Warning("could not process removal action (%d) '%s': %v", i, desc, err)
			}
		}
	}

	// Remove any project files
	for _, f := range manifestData.ProjectFiles {
		p := app.GetConfigPath(f)
		err = fileutil.CheckSignatureOrNoFile(p, nodeps.DdevFileSignature)
		if err == nil {
			_ = os.RemoveAll(p)
		} else {
			util.Warning("Unwilling to remove '%s' because it does not have #ddev-generated in it: %v; you can manually delete it if it is safe to delete.", p, err)
		}
	}

	// Remove any global files
	globalDotDdev := filepath.Join(globalconfig.GetGlobalDdevDir())
	for _, f := range manifestData.GlobalFiles {
		p := filepath.Join(globalDotDdev, f)
		err = fileutil.CheckSignatureOrNoFile(p, nodeps.DdevFileSignature)
		if err == nil {
			_ = os.RemoveAll(p)
		} else {
			util.Warning("Unwilling to remove '%s' because it does not have #ddev-generated in it: %v; you can manually delete it if it is safe to delete.", p, err)
		}
	}
	if len(manifestData.Dependencies) > 0 {
		for _, dep := range manifestData.Dependencies {
			if m, ok := manifests[dep]; ok {
				util.Warning("The add-on you're removing ('%s') declares a dependency on '%s', which is not being removed. You may want to remove it manually if it is no longer needed.", addonName, m.Name)
			}
		}
	}

	// Clean up temporary configuration files created for PHP actions
	err = app.CleanupConfigurationFiles()
	if err != nil {
		util.Warning("Unable to clean up temporary configuration files: %v", err)
	}

	err = os.RemoveAll(app.GetConfigPath(filepath.Join(AddonMetadataDir, manifestData.Name)))
	if err != nil {
		return fmt.Errorf("error removing addon metadata directory %s: %v", manifestData.Name, err)
	}
	util.Success("Removed add-on %s", addonName)
	return nil
}

// GetGitHubRelease gets the tarball URL and version for a GitHub repository release
func GetGitHubRelease(owner, repo, requestedVersion string) (tarballURL, downloadedRelease string, err error) {
	ctx := context.Background()
	client := github2.GetGithubClient(ctx)

	releases, resp, err := client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{PerPage: 100})
	if err != nil {
		var rate github.Rate
		if resp != nil {
			rate = resp.Rate
		}
		return "", "", fmt.Errorf("unable to get releases for %v: %v\nresp.Rate=%v", repo, err, rate)
	}
	if len(releases) == 0 {
		return "", "", fmt.Errorf("no releases found for %v", repo)
	}

	releaseItem := 0
	releaseFound := false
	if requestedVersion != "" {
		for i, release := range releases {
			if release.GetTagName() == requestedVersion {
				releaseItem = i
				releaseFound = true
				break
			}
		}
		if !releaseFound {
			return "", "", fmt.Errorf("no release found for %v with tag %v", repo, requestedVersion)
		}
	}

	tarballURL = releases[releaseItem].GetTarballURL()
	downloadedRelease = releases[releaseItem].GetTagName()
	return tarballURL, downloadedRelease, nil
}

// Global variables to track installation stack for circular dependency detection
var installStack []string
var installStackMap map[string]bool

func init() {
	installStackMap = make(map[string]bool)
}

// ResetInstallStack clears the installation stack (for testing)
func ResetInstallStack() {
	installStack = []string{}
	installStackMap = make(map[string]bool)
}

// ParseRuntimeDependencies reads and parses a .runtime-deps file
func ParseRuntimeDependencies(runtimeDepsFile string) ([]string, error) {
	if !fileutil.FileExists(runtimeDepsFile) {
		return nil, nil // No runtime dependencies file
	}

	content, err := fileutil.ReadFileIntoString(runtimeDepsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read runtime dependencies file %s: %v", runtimeDepsFile, err)
	}

	var dependencies []string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		dependencies = append(dependencies, line)
	}

	return dependencies, nil
}

// NormalizeAddonIdentifier converts various addon identifier formats to a canonical form
// This helps detect circular dependencies when the same addon is referenced in different ways
func NormalizeAddonIdentifier(addonIdentifier string) string {
	// For GitHub URLs like https://github.com/owner/repo/archive/refs/tags/v1.0.0.tar.gz
	if strings.HasPrefix(addonIdentifier, "https://github.com/") {
		parts := strings.Split(addonIdentifier, "/")
		if len(parts) >= 5 {
			return parts[3] + "/" + parts[4] // Extract owner/repo
		}
	}

	// For owner/repo format, use as-is
	parts := strings.Split(addonIdentifier, "/")
	if len(parts) == 2 && !strings.Contains(addonIdentifier, ".") && !strings.HasPrefix(addonIdentifier, ".") {
		return addonIdentifier
	}

	// For local paths or other formats, use the basename without extension
	base := filepath.Base(addonIdentifier)
	// Remove common archive extensions
	for _, ext := range []string{".tar.gz", ".tgz", ".tar", ".zip"} {
		if strings.HasSuffix(base, ext) {
			base = strings.TrimSuffix(base, ext)
			break
		}
	}

	return base
}

// AddToInstallStack adds an addon to the installation stack and checks for circular dependencies
func AddToInstallStack(addonName string) error {
	// Normalize the addon identifier for consistent circular dependency detection
	normalizedName := NormalizeAddonIdentifier(addonName)

	// Check for circular dependencies using normalized name
	if installStackMap[normalizedName] {
		return fmt.Errorf("circular dependency detected: %s",
			strings.Join(append(installStack, addonName), " -> "))
	}
	installStack = append(installStack, addonName)
	installStackMap[normalizedName] = true
	return nil
}

// InstallDependencies installs a list of dependencies, checking if they're already installed
func InstallDependencies(app *DdevApp, dependencies []string, verbose bool) error {
	m, err := GatherAllManifests(app)
	if err != nil {
		return fmt.Errorf("unable to gather manifests: %w", err)
	}

	for _, dep := range dependencies {
		if _, exists := m[dep]; !exists {
			util.Success("Installing missing dependency: %s", dep)
			err = installAddonRecursive(app, dep, verbose)
			if err != nil {
				return fmt.Errorf("failed to install dependency '%s': %w", dep, err)
			}
			// Refresh manifest cache after installation
			m, _ = GatherAllManifests(app)
		} else if verbose {
			util.Success("Dependency '%s' is already installed", dep)
		}
	}
	return nil
}

// installAddonRecursive installs an addon and its dependencies recursively
func installAddonRecursive(app *DdevApp, addonName string, verbose bool) error {
	// Normalize the addon identifier for consistent circular dependency detection
	normalizedName := NormalizeAddonIdentifier(addonName)

	// Check for circular dependencies using normalized name
	if installStackMap[normalizedName] {
		return fmt.Errorf("circular dependency detected: %s",
			strings.Join(append(installStack, addonName), " -> "))
	}

	installStack = append(installStack, addonName)
	installStackMap[normalizedName] = true
	defer func() {
		// Clean up both slice and map using normalized name
		installStack = installStack[:len(installStack)-1]
		delete(installStackMap, normalizedName)
	}()

	// Handle different dependency formats (same as ddev add-on get)
	parts := strings.Split(addonName, "/")
	extractedDir := ""
	tarballURL := ""
	var cleanup func()

	switch {
	// Local directory path (check this first before GitHub parsing)
	case fileutil.IsDirectory(addonName):
		extractedDir = addonName
		if verbose {
			util.Success("Installing from local directory: %s", addonName)
		}

	// Local tarball file
	case fileutil.FileExists(addonName) && (strings.HasSuffix(filepath.Base(addonName), "tar.gz") || strings.HasSuffix(filepath.Base(addonName), "tar") || strings.HasSuffix(filepath.Base(addonName), "tgz")):
		var err error
		extractedDir, cleanup, err = archive.ExtractTarballWithCleanup(addonName, true)
		if err != nil {
			return fmt.Errorf("unable to extract %s: %v", addonName, err)
		}
		defer cleanup()

	// URL to tarball (including GitHub tarball URLs)
	case strings.HasPrefix(addonName, "http://") || strings.HasPrefix(addonName, "https://"):
		tarballURL = addonName
		var err error
		extractedDir, cleanup, err = archive.DownloadAndExtractTarball(tarballURL, true)
		if err != nil {
			return fmt.Errorf("unable to download %v: %v", addonName, err)
		}
		defer cleanup()

	// GitHub owner/repo format (check this last, and exclude paths)
	case len(parts) == 2 && !strings.Contains(addonName, ".") && !strings.HasPrefix(addonName, "."):
		return InstallAddonFromGitHub(app, addonName, "", verbose)

	default:
		return fmt.Errorf("unsupported dependency format: %s (must be owner/repo, /path/to/addon, or https://...)", addonName)
	}

	// If we have a local extraction, handle it directly
	if extractedDir != "" {
		return InstallAddonFromDirectory(app, extractedDir, verbose)
	}

	return fmt.Errorf("no extraction directory available for addon: %s", addonName)
}

// InstallAddonFromGitHub handles GitHub-based addon installation
func InstallAddonFromGitHub(app *DdevApp, addonName, requestedVersion string, verbose bool) error {
	// Parse owner/repo from addonName
	parts := strings.Split(addonName, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid addon name format, expected 'owner/repo': %s", addonName)
	}

	owner := parts[0]
	repo := parts[1]

	// Get GitHub release
	tarballURL, downloadedRelease, err := GetGitHubRelease(owner, repo, requestedVersion)
	if err != nil {
		return err
	}

	// Download and install
	return InstallAddonFromTarball(app, tarballURL, downloadedRelease, verbose)
}

// InstallAddonFromDirectory handles installation from a local directory
func InstallAddonFromDirectory(app *DdevApp, extractedDir string, verbose bool) error {
	// Parse install.yaml
	yamlFile := filepath.Join(extractedDir, "install.yaml")
	yamlContent, err := fileutil.ReadFileIntoString(yamlFile)
	if err != nil {
		return fmt.Errorf("unable to read %v: %v", yamlFile, err)
	}
	var s InstallDesc
	err = yaml.Unmarshal([]byte(yamlContent), &s)
	if err != nil {
		return fmt.Errorf("unable to parse %v: %v", yamlFile, err)
	}

	// Check version constraint
	if s.DdevVersionConstraint != "" {
		err := CheckDdevVersionConstraint(s.DdevVersionConstraint, fmt.Sprintf("Unable to install the '%s' add-on", s.Name), "")
		if err != nil {
			return err
		}
	}

	// Install dependencies recursively, resolving relative paths
	if len(s.Dependencies) > 0 {
		resolvedDeps := make([]string, len(s.Dependencies))
		for i, dep := range s.Dependencies {
			if strings.HasPrefix(dep, "../") || strings.HasPrefix(dep, "./") {
				// Resolve relative path relative to extractedDir
				resolvedPath := filepath.Join(extractedDir, dep)
				// Clean the path to resolve .. and . components
				resolvedDeps[i] = filepath.Clean(resolvedPath)
				if verbose {
					util.Success("Resolved relative dependency '%s' to '%s'", dep, resolvedDeps[i])
				}
			} else {
				resolvedDeps[i] = dep
			}
		}
		err = InstallDependencies(app, resolvedDeps, verbose)
		if err != nil {
			return fmt.Errorf("unable to install dependencies for '%s': %v", s.Name, err)
		}
	}

	// Run pre-install actions
	if len(s.PreInstallActions) > 0 {
		util.Success("\nExecuting pre-install actions:")
	}
	for i, action := range s.PreInstallActions {
		err = ProcessAddonAction(action, s, app, verbose)
		if err != nil {
			desc := GetAddonDdevDescription(action)
			if !verbose {
				return fmt.Errorf("could not process pre-install action (%d) '%s'", i, desc)
			} else {
				return fmt.Errorf("could not process pre-install action (%d) '%s'; error=%v\n action=%s", i, desc, err, action)
			}
		}
	}

	// Check for runtime dependencies generated during pre-install actions
	runtimeDepsFile := app.GetConfigPath(".runtime-deps-" + s.Name)
	runtimeDeps, err := ParseRuntimeDependencies(runtimeDepsFile)
	if err != nil {
		return fmt.Errorf("failed to parse runtime dependencies: %v", err)
	}
	if len(runtimeDeps) > 0 {
		util.Success("Installing runtime dependencies:")
		// Resolve relative paths for runtime dependencies too
		resolvedRuntimeDeps := make([]string, len(runtimeDeps))
		for i, dep := range runtimeDeps {
			if strings.HasPrefix(dep, "../") || strings.HasPrefix(dep, "./") {
				// Resolve relative to the extracted addon directory (where relative paths are based)
				resolvedPath := filepath.Join(extractedDir, dep)
				resolvedRuntimeDeps[i] = filepath.Clean(resolvedPath)
				if verbose {
					util.Success("Resolved runtime dependency '%s' to '%s'", dep, resolvedRuntimeDeps[i])
				}
			} else {
				resolvedRuntimeDeps[i] = dep
			}
		}
		err = InstallDependencies(app, resolvedRuntimeDeps, verbose)
		if err != nil {
			return fmt.Errorf("failed to install runtime dependencies for '%s': %v", s.Name, err)
		}
		// Clean up the runtime dependencies file
		_ = os.Remove(runtimeDepsFile)
	}

	// Install project files
	if len(s.ProjectFiles) > 0 {
		util.Success("\nInstalling project-level components:")
	}

	projectFiles, err := fileutil.ExpandFilesAndDirectories(extractedDir, s.ProjectFiles)
	if err != nil {
		return fmt.Errorf("unable to expand files and directories: %v", err)
	}
	for _, file := range projectFiles {
		src := filepath.Join(extractedDir, file)
		dest := app.GetConfigPath(file)
		if err = fileutil.CheckSignatureOrNoFile(dest, nodeps.DdevFileSignature); err == nil {
			err = copy.Copy(src, dest)
			if err != nil {
				return fmt.Errorf("unable to copy %v to %v: %v", src, dest, err)
			}
			util.Success("%c %s", '\U0001F44D', file)
		} else {
			util.Warning("NOT overwriting %s. The #ddev-generated signature was not found in the file, so it will not be overwritten. You can remove the file and use ddev add-on get again if you want it to be replaced: %v", dest, err)
		}
	}

	// Install global files
	globalDotDdev := filepath.Join(globalconfig.GetGlobalDdevDir())
	if len(s.GlobalFiles) > 0 {
		util.Success("\nInstalling global components:")
	}

	globalFiles, err := fileutil.ExpandFilesAndDirectories(extractedDir, s.GlobalFiles)
	if err != nil {
		return fmt.Errorf("unable to expand global files and directories: %v", err)
	}
	for _, file := range globalFiles {
		src := filepath.Join(extractedDir, file)
		dest := filepath.Join(globalDotDdev, file)

		// If the file existed and had #ddev-generated OR if it did not exist, copy it in.
		if err = fileutil.CheckSignatureOrNoFile(dest, nodeps.DdevFileSignature); err == nil {
			err = copy.Copy(src, dest)
			if err != nil {
				return fmt.Errorf("unable to copy %v to %v: %v", src, dest, err)
			}
			util.Success("%c %s", '\U0001F44D', file)
		} else {
			util.Warning("NOT overwriting %s. The #ddev-generated signature was not found in the file, so it will not be overwritten. You can remove the file and use ddev add-on get again if you want it to be replaced: %v", dest, err)
		}
	}

	// Change to project directory for post-install actions
	origDir, _ := os.Getwd()
	defer func() {
		err = os.Chdir(origDir)
		if err != nil {
			util.Warning("Unable to chdir back to %v: %v", origDir, err)
		}
	}()

	err = os.Chdir(app.GetConfigPath(""))
	if err != nil {
		return fmt.Errorf("unable to chdir to %v: %v", app.GetConfigPath(""), err)
	}

	// Run post-install actions
	if len(s.PostInstallActions) > 0 {
		util.Success("\nExecuting post-install actions:")
	}
	for i, action := range s.PostInstallActions {
		err = ProcessAddonAction(action, s, app, verbose)
		if err != nil {
			desc := GetAddonDdevDescription(action)
			if !verbose {
				return fmt.Errorf("could not process post-install action (%d) '%s'", i, desc)
			} else {
				return fmt.Errorf("could not process post-install action (%d) '%s'; error=%v\n action=%s", i, desc, err, action)
			}
		}
	}

	// Clean up temporary configuration files created for PHP actions
	err = app.CleanupConfigurationFiles()
	if err != nil {
		util.Warning("Unable to clean up temporary configuration files: %v", err)
	}

	util.Success("Successfully installed %s from directory", s.Name)
	return nil
}

// InstallAddonFromTarball handles complete installation process for tarball-based addons
func InstallAddonFromTarball(app *DdevApp, tarballURL, downloadedRelease string, verbose bool) error {
	// Extract tarball
	extractedDir, cleanup, err := archive.DownloadAndExtractTarball(tarballURL, true)
	if err != nil {
		return fmt.Errorf("unable to download %v: %v", tarballURL, err)
	}
	defer cleanup()

	// Use the directory installation method for complete processing
	return InstallAddonFromDirectory(app, extractedDir, verbose)
}

// ValidateDependencies checks that all declared dependencies exist without installing them
// Used when the --no-dependencies flag is specified
func ValidateDependencies(app *DdevApp, dependencies []string, extractedDir, addonName string) error {
	m, err := GatherAllManifests(app)
	if err != nil {
		return fmt.Errorf("unable to gather manifests: %w", err)
	}
	for _, dep := range dependencies {
		checkName := dep
		// For relative paths, we need to check the actual addon name, not the path
		if strings.HasPrefix(dep, "../") || strings.HasPrefix(dep, "./") {
			// Resolve relative path and get the addon name from install.yaml
			resolvedPath := filepath.Clean(filepath.Join(extractedDir, dep))
			if fileutil.IsDirectory(resolvedPath) {
				yamlFile := filepath.Join(resolvedPath, "install.yaml")
				if yamlContent, err := fileutil.ReadFileIntoString(yamlFile); err == nil {
					var depDesc InstallDesc
					if err := yaml.Unmarshal([]byte(yamlContent), &depDesc); err == nil {
						checkName = depDesc.Name
					}
				}
			}
		}
		if _, ok := m[checkName]; !ok {
			return fmt.Errorf("the add-on '%s' declares a dependency on '%s'; please ddev add-on get %s first", addonName, dep, dep)
		}
	}
	return nil
}

// ResolveDependencyPaths resolves relative paths in dependencies relative to extractedDir
func ResolveDependencyPaths(dependencies []string, extractedDir string, verbose bool) []string {
	resolvedDeps := make([]string, len(dependencies))
	for i, dep := range dependencies {
		if strings.HasPrefix(dep, "../") || strings.HasPrefix(dep, "./") {
			// Resolve relative path relative to extractedDir
			resolvedPath := filepath.Join(extractedDir, dep)
			// Clean the path to resolve .. and . components
			resolvedDeps[i] = filepath.Clean(resolvedPath)
			if verbose {
				util.Success("Resolved relative dependency '%s' to '%s'", dep, resolvedDeps[i])
			}
		} else {
			resolvedDeps[i] = dep
		}
	}
	return resolvedDeps
}

// ProcessRuntimeDependencies handles runtime dependencies generated during pre-install actions
func ProcessRuntimeDependencies(app *DdevApp, addonName, extractedDir string, verbose bool) error {
	runtimeDepsFile := app.GetConfigPath(".runtime-deps-" + addonName)
	if verbose {
		util.Success("Checking for runtime dependencies file: %s", runtimeDepsFile)
		if fileutil.FileExists(runtimeDepsFile) {
			content, _ := fileutil.ReadFileIntoString(runtimeDepsFile)
			util.Success("Runtime dependencies file contents: %q", content)
		} else {
			util.Success("Runtime dependencies file does not exist")
		}
	}

	runtimeDeps, err := ParseRuntimeDependencies(runtimeDepsFile)
	if err != nil {
		return fmt.Errorf("failed to parse runtime dependencies: %w", err)
	}

	if verbose {
		util.Success("Found %d runtime dependencies: %v", len(runtimeDeps), runtimeDeps)
	}

	if len(runtimeDeps) > 0 {
		util.Success("Installing runtime dependencies:")
		// Resolve relative paths for runtime dependencies
		resolvedRuntimeDeps := ResolveDependencyPaths(runtimeDeps, extractedDir, verbose)
		err := InstallDependencies(app, resolvedRuntimeDeps, verbose)
		if err != nil {
			return fmt.Errorf("failed to install runtime dependencies for '%s': %w", addonName, err)
		}
		// Clean up the runtime dependencies file
		_ = os.Remove(runtimeDepsFile)
	}
	return nil
}

// GatherAllManifests searches for all addon manifests and presents the result
// as a map of various names to manifest data
func GatherAllManifests(app *DdevApp) (map[string]AddonManifest, error) {
	metadataDir := app.GetConfigPath(AddonMetadataDir)
	allManifests := make(map[string]AddonManifest)
	err := os.MkdirAll(metadataDir, 0755)
	if err != nil {
		return nil, err
	}

	dirs, err := fileutil.ListFilesInDirFullPath(metadataDir, false)
	if err != nil {
		return nil, err
	}
	for _, d := range dirs {
		if !fileutil.IsDirectory(d) {
			continue
		}

		mPath := filepath.Join(d, "manifest.yaml")
		manifestString, err := fileutil.ReadFileIntoString(mPath)
		if err != nil {
			return nil, err
		}
		var manifestData = &AddonManifest{}
		err = yaml.Unmarshal([]byte(manifestString), manifestData)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling manifest data: %v", err)
		}
		allManifests[manifestData.Name] = *manifestData
		allManifests[manifestData.Repository] = *manifestData

		pathParts := strings.Split(manifestData.Repository, "/")
		if len(pathParts) > 1 {
			shortRepo := pathParts[len(pathParts)-1]
			allManifests[shortRepo] = *manifestData
		}
	}
	return allManifests, nil
}
