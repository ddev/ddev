package ddevapp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	goexec "os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	github2 "github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/google/go-github/v72/github"
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

// ProcessAddonAction takes a stanza from yaml exec section and executes it.
func ProcessAddonAction(action string, dict map[string]interface{}, bashPath string, verbose bool) error {
	return ProcessAddonActionWithImage(action, dict, bashPath, verbose, "", nil)
}

// ProcessAddonActionWithImage takes a stanza from yaml exec section and executes it, optionally in a container.
func ProcessAddonActionWithImage(action string, dict map[string]interface{}, bashPath string, verbose bool, image string, app *DdevApp) error {
	// Check if the action starts with <?php
	if strings.HasPrefix(strings.TrimSpace(action), "<?php") {
		if app == nil {
			return fmt.Errorf("app is required for PHP actions")
		}
		return processPHPAction(action, dict, image, verbose, app)
	}

	// Default behavior for bash actions
	action = "set -eu -o pipefail\n" + action
	t, err := template.New("ProcessAddonAction").Funcs(getTemplateFuncMap()).Parse(action)
	if err != nil {
		return fmt.Errorf("could not parse action '%s': %v", action, err)
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
	out, err := exec.RunHostCommand(bashPath, "-c", action)
	if err != nil {
		warningCode := GetAddonDdevWarningExitCode(action)
		if warningCode > 0 {
			var exitErr *goexec.ExitError
			if errors.As(err, &exitErr) {
				// Get the exit code
				exitCode := exitErr.ExitCode()
				if exitCode == warningCode {
					if desc != "" {
						util.Warning("%s %s", "\U000026A0\U0000FE0F", desc)
					}
					err = nil
				}
			}
		}
		if err != nil {
			if desc != "" {
				util.Warning("%c %s", '\U0001F44E', desc)
			}
			err = fmt.Errorf("unable to run action %v: %v, output=%s", action, err, out)
		}
	} else {
		if desc != "" {
			util.Success("%c %s", '\U0001F44D', desc)
		}
	}
	if len(out) > 0 {
		util.Warning(out)
	}
	return err
}

// processPHPAction executes a PHP action in a container
func processPHPAction(action string, dict map[string]interface{}, image string, verbose bool, app *DdevApp) error {
	// Extract description before processing
	desc := GetAddonDdevDescription(action)

	// Store the original action for validation
	originalAction := action

	// Make sure the standard ddev_default network is available
	dockerutil.EnsureDdevNetwork()

	// Validate included/required files on the host (since we need to read from filesystem)
	err := validatePHPIncludesAndRequires(action, app, image)
	if err != nil {
		return fmt.Errorf("PHP include/require validation error: %v", err)
	}

	// Inject PHP strict error handling
	action = injectPHPStrictMode(action)
	// Use the default ddev-webserver as the image if none is specified
	if image == "" {
		image = docker.GetWebImage()
	}

	// Create configuration files for PHP action access (optional for tests and removal actions)
	err = createConfigurationFiles(app)
	if err != nil {
		return fmt.Errorf("failed to create configuration files for PHP action: %w", err)
	}

	// Create a shell script that validates original PHP syntax first, then executes with strict mode
	shellScript := fmt.Sprintf(phpActionShellScriptTemplate, originalAction, action)

	cmd := []string{"sh", "-c", shellScript}

	// Bind mount the .ddev directory and the project root into the container
	binds := []string{fmt.Sprintf("%s:/var/www/html", app.AppRoot)}

	// Build environment variables array with standard DDEV variables
	env := buildPHPActionEnvironment(app, verbose)

	// Create in-memory docker-compose project for PHP action execution
	phpProject, err := dockerutil.CreateComposeProject("name: ddev-php-action")
	if err != nil {
		return fmt.Errorf("failed to create compose project: %v", err)
	}

	// Create service configuration for PHP action
	serviceName := "php-runner"
	serviceConfig := composeTypes.ServiceConfig{
		Name:        serviceName,
		Image:       image,
		Command:     cmd,
		WorkingDir:  "/var/www/html/.ddev",
		Environment: buildEnvironmentMap(env),
		Volumes:     buildVolumeConfigs(binds),
		Networks: map[string]*composeTypes.ServiceNetworkConfig{
			"default":          {},
			dockerutil.NetName: {},
		},
		Labels: composeTypes.Labels{
			"com.ddev.site-name": app.Name + "-php-action",
			"com.ddev.approot":   app.AppRoot,
		},
	}

	// Add host.docker.internal setup using DDEV's standard logic
	hostDockerInternalIP, err := dockerutil.GetHostDockerInternalIP()
	if err != nil {
		util.Warning("Could not determine host.docker.internal IP address: %v", err)
	}

	// Set up host.docker.internal based on DDEV's standard approach
	if hostDockerInternalIP != "" {
		// Use specific IP address for host.docker.internal
		extraHosts, err := composeTypes.NewHostsList([]string{
			"host.docker.internal:" + hostDockerInternalIP,
		})
		if err == nil {
			serviceConfig.ExtraHosts = extraHosts
		}
	} else if (runtime.GOOS == "linux" && !nodeps.IsWSL2() && !dockerutil.IsColima()) ||
		(nodeps.IsWSL2() && globalconfig.DdevGlobalConfig.XdebugIDELocation == globalconfig.XdebugIDELocationWSL2) {
		// Use host-gateway for modern Docker on Linux
		extraHosts, err := composeTypes.NewHostsList([]string{
			"host.docker.internal:host-gateway",
		})
		if err == nil {
			serviceConfig.ExtraHosts = extraHosts
		}
	}

	phpProject.Services[serviceName] = serviceConfig

	// Add network definitions matching DDEV's standard networking
	if phpProject.Networks == nil {
		phpProject.Networks = composeTypes.Networks{}
	}
	phpProject.Networks[dockerutil.NetName] = composeTypes.NetworkConfig{
		Name:     dockerutil.NetName,
		External: true,
	}
	phpProject.Networks["default"] = composeTypes.NetworkConfig{
		Name:   "php-action-" + app.GetComposeProjectName() + "_default",
		Labels: composeTypes.Labels{"com.ddev.platform": "ddev"},
	}
	defer func() {
		dockerutil.RemoveNetworkWithWarningOnError(phpProject.Networks["default"].Name)
	}()

	// Execute PHP action using docker-compose run
	// Use ComposeCmd to capture output for error reporting, ComposeWithStreams for streaming
	stdout, stderr, err := dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
		ComposeYaml: phpProject,
		Action:      []string{"run", "--rm", "--no-deps", serviceName},
		ProjectName: "php-action-" + app.GetComposeProjectName(),
	})

	if err != nil {
		if desc != "" {
			util.Warning("%c %s", '\U0001F44E', desc) // 👎 error emoji
		}
		// Include output in error message for debugging, similar to original implementation
		combinedOutput := stdout
		if stderr != "" {
			if combinedOutput != "" {
				combinedOutput += "\n"
			}
			combinedOutput += stderr
		}
		if combinedOutput != "" {
			return fmt.Errorf("PHP script failed: %v - Output: %s", err, combinedOutput)
		}
		return fmt.Errorf("PHP script failed: %v", err)
	}

	// Display captured output
	if stdout != "" {
		util.Success(stdout)
	}

	// Show description on success
	if desc != "" {
		util.Success("%c %s", '\U0001F44D', desc) // 👍 success emoji
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

// buildEnvironmentMap converts environment variable slice to compose environment format
func buildEnvironmentMap(envSlice []string) composeTypes.MappingWithEquals {
	envMap := composeTypes.MappingWithEquals{}
	for _, envVar := range envSlice {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = &parts[1]
		}
	}
	return envMap
}

// buildVolumeConfigs converts bind mount slice to compose volume format
func buildVolumeConfigs(binds []string) []composeTypes.ServiceVolumeConfig {
	var volumes []composeTypes.ServiceVolumeConfig
	for _, bind := range binds {
		parts := strings.Split(bind, ":")
		if len(parts) >= 2 {
			volumes = append(volumes, composeTypes.ServiceVolumeConfig{
				Type:   "bind",
				Source: parts[0],
				Target: parts[1],
			})
		}
	}
	return volumes
}

// buildPHPActionEnvironment creates the environment variables for PHP actions
func buildPHPActionEnvironment(app *DdevApp, verbose bool) []string {
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

	return env
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
func validatePHPSyntax(phpCode string, app *DdevApp, image string) error {
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

	cmd := []string{"sh", "-c", shellScript}

	// Create in-memory docker-compose project for PHP validation
	phpProject, err := dockerutil.CreateComposeProject("name: ddev-php-validate")
	if err != nil {
		return fmt.Errorf("failed to create validation compose project: %v", err)
	}

	// Create service configuration for PHP validation
	serviceName := "php-validator"
	phpProject.Services[serviceName] = composeTypes.ServiceConfig{
		Name:    serviceName,
		Image:   image,
		Command: cmd,
	}

	// Execute PHP validation using docker-compose run
	stdout, stderr, err := dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
		ComposeYaml: phpProject,
		Action:      []string{"run", "--rm", "--no-deps", serviceName},
		ProjectName: "php-validate-" + app.GetComposeProjectName(),
	})

	if err != nil {
		// Include validation output in error message for debugging
		combinedOutput := stdout
		if stderr != "" {
			if combinedOutput != "" {
				combinedOutput += "\n"
			}
			combinedOutput += stderr
		}
		if combinedOutput != "" {
			return fmt.Errorf("PHP syntax validation failed: %s", combinedOutput)
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
	includePattern := `(include|include_once|require|require_once)[[:space:]]+.*\.(php|inc)`
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
			if end := strings.IndexByte(match[start+1:], byte(quote)); end != -1 {
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
	dynamicPatterns := []string{"$", "__DIR__", "__FILE__", "dirname", "realpath", "getcwd", "."}

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
		err = validatePHPSyntax(content, app, image)
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
func RemoveAddon(app *DdevApp, addonName string, dict map[string]interface{}, bash string, verbose bool, skipRemovalActions bool) error {
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
			// For PHP actions, we need to pass the app, but it's okay if the project isn't running
			trimmedAction := strings.TrimSpace(action)
			if strings.HasPrefix(trimmedAction, "<?php") {
				err = ProcessAddonActionWithImage(action, dict, bash, verbose, "", app)
			} else {
				err = ProcessAddonAction(action, dict, bash, verbose)
			}
			desc := GetAddonDdevDescription(action)
			if err != nil {
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

	err = os.RemoveAll(app.GetConfigPath(filepath.Join(AddonMetadataDir, manifestData.Name)))
	if err != nil {
		return fmt.Errorf("error removing addon metadata directory %s: %v", manifestData.Name, err)
	}
	util.Success("Removed add-on %s", addonName)
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
