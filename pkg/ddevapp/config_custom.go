package ddevapp

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
)

// customConfigCheck defines a custom configuration check.
type customConfigCheck struct {
	collectFiles      func() ([]string, error) // returns candidate files before custom config filtering
	expectedDdevFiles func() []string          // whitelisted DDEV-managed files (optional, returns nil to check all files)
	checkOnlyWhen     func() bool              // when to run this check (optional, returns true if nil)
	displayName       string                   // category name for grouped display (e.g., "Router (global)")
}

// CheckCustomConfig checks for custom configuration files and returns a message.
// If showAll is true, files with #ddev-silent-no-warn marker are included in the output.
// Returns the message and a bool indicating if warnings were found (true = warnings, false = success).
func (app *DdevApp) CheckCustomConfig(showAll bool) (message string, hasWarnings bool) {
	ddevDir := filepath.Dir(app.ConfigPath)

	// Define all configuration checks
	routerEnabled := func() bool { return !slices.Contains(app.OmitContainersGlobal, RouterComposeProjectName) }
	sshAgentEnabled := func() bool { return !slices.Contains(app.OmitContainersGlobal, SSHAuthName) }

	checks := []customConfigCheck{
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(globalconfig.GetGlobalDdevDir(), "router-compose.*.yaml"))
			},
			checkOnlyWhen: routerEnabled,
			displayName:   "Router (global)",
		},
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(globalconfig.GetGlobalDdevDir(), "ssh-auth-compose.*.yaml"))
			},
			checkOnlyWhen: sshAgentEnabled,
			displayName:   "SSH agent (global)",
		},
		{
			collectFiles: func() ([]string, error) {
				return fileutil.ListFilesWithDepth(filepath.Join(globalconfig.GetGlobalDdevDir(), "commands"), 2)
			},
			expectedDdevFiles: func() []string {
				return GetAssetFiles("global_dotddev_assets/commands", filepath.Join(globalconfig.GetGlobalDdevDir(), "commands"))
			},
			displayName: "Commands (global)",
		},
		{
			collectFiles: func() ([]string, error) {
				return fileutil.ListFilesWithDepth(filepath.Join(globalconfig.GetGlobalDdevDir(), "homeadditions"), 2)
			},
			expectedDdevFiles: func() []string {
				return GetAssetFiles("global_dotddev_assets/homeadditions", filepath.Join(globalconfig.GetGlobalDdevDir(), "homeadditions"))
			},
			displayName: "Home additions (global)",
		},
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik", "static_config.*.yaml"))
			},
			checkOnlyWhen: routerEnabled,
			displayName:   "Router (global)",
		},
		{
			collectFiles: func() ([]string, error) {
				return fileutil.ListFilesInDirFullPath(filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik", "custom-global-config"), true)
			},
			expectedDdevFiles: func() []string {
				return GetAssetFiles("global_dotddev_assets/traefik/custom-global-config", filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik", "custom-global-config"))
			},
			checkOnlyWhen: routerEnabled,
			displayName:   "Router (global)",
		},
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(ddevDir, "apache", "*.conf"))
			},
			expectedDdevFiles: func() []string {
				return []string{app.GetConfigPath("apache/apache-site.conf")}
			},
			checkOnlyWhen: func() bool { return app.WebserverType == nodeps.WebserverApacheFPM },
			displayName:   "Web server",
		},
		{
			collectFiles: func() ([]string, error) {
				allFiles, err := filepath.Glob(filepath.Join(ddevDir, "db-build", "*Dockerfile*"))
				if err != nil {
					return nil, err
				}
				// Filter to only valid Dockerfile patterns
				return slices.DeleteFunc(allFiles, func(file string) bool {
					base := filepath.Base(file)
					return !strings.HasPrefix(base, "Dockerfile") &&
						!strings.HasPrefix(base, "pre.Dockerfile") &&
						!strings.HasPrefix(base, "prepend.Dockerfile")
				}), nil
			},
			displayName: "Database",
		},
		{
			collectFiles: func() ([]string, error) {
				return []string{app.GetConfigPath("mutagen/mutagen.yml")}, nil
			},
			expectedDdevFiles: func() []string {
				return []string{app.GetConfigPath("mutagen/mutagen.yml")}
			},
			checkOnlyWhen: func() bool { return app.IsMutagenEnabled() },
			displayName:   "Mutagen",
		},
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(ddevDir, "mysql", "*.cnf"))
			},
			checkOnlyWhen: func() bool {
				return !slices.Contains(app.OmitContainers, "db") &&
					slices.Contains([]string{nodeps.MariaDB, nodeps.MySQL}, app.Database.Type)
			},
			displayName: "Database",
		},
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(ddevDir, "nginx", "*.conf"))
			},
			checkOnlyWhen: func() bool { return app.WebserverType == nodeps.WebserverNginxFPM },
			displayName:   "Web server",
		},
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(ddevDir, "nginx_full", "*.conf"))
			},
			expectedDdevFiles: func() []string {
				return []string{app.GetConfigPath("nginx_full/nginx-site.conf")}
			},
			checkOnlyWhen: func() bool { return app.WebserverType == nodeps.WebserverNginxFPM },
			displayName:   "Web server",
		},
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(ddevDir, "php", "*.ini"))
			},
			displayName: "PHP",
		},
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(ddevDir, "postgres", "*.conf"))
			},
			expectedDdevFiles: func() []string {
				return []string{app.GetConfigPath("postgres/postgresql.conf")}
			},
			checkOnlyWhen: func() bool {
				return !slices.Contains(app.OmitContainers, "db") &&
					app.Database.Type == nodeps.Postgres
			},
			displayName: "Database",
		},
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(ddevDir, "providers", "*.yaml"))
			},
			expectedDdevFiles: func() []string {
				return GetAssetFiles("dotddev_assets/providers", app.GetConfigPath("providers"))
			},
			displayName: "Hosting providers",
		},
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(ddevDir, "share-providers", "*.sh"))
			},
			expectedDdevFiles: func() []string {
				return GetAssetFiles("dotddev_assets/share-providers", app.GetConfigPath("share-providers"))
			},
			displayName: "Share providers",
		},
		{
			collectFiles: func() ([]string, error) {
				crtFiles, err := filepath.Glob(filepath.Join(app.GetConfigPath("traefik/certs"), "*.crt"))
				if err != nil {
					return nil, err
				}
				keyFiles, err := filepath.Glob(filepath.Join(app.GetConfigPath("traefik/certs"), "*.key"))
				if err != nil {
					return nil, err
				}
				return append(crtFiles, keyFiles...), nil
			},
			expectedDdevFiles: func() []string {
				return []string{
					filepath.Join(app.GetConfigPath("traefik/certs"), app.Name+".crt"),
					filepath.Join(app.GetConfigPath("traefik/certs"), app.Name+".key"),
				}
			},
			displayName: "Router",
		},
		{
			collectFiles: func() ([]string, error) {
				customCertsDir := app.GetConfigPath("custom_certs")
				crtFiles, err := filepath.Glob(filepath.Join(customCertsDir, "*.crt"))
				if err != nil {
					return nil, err
				}
				keyFiles, err := filepath.Glob(filepath.Join(customCertsDir, "*.key"))
				if err != nil {
					return nil, err
				}
				return append(crtFiles, keyFiles...), nil
			},
			displayName: "Router",
		},
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(app.GetConfigPath("traefik/config"), "*.yaml"))
			},
			expectedDdevFiles: func() []string {
				return []string{filepath.Join(app.GetConfigPath("traefik/config"), app.Name+".yaml")}
			},
			displayName: "Router",
		},
		{
			collectFiles: func() ([]string, error) {
				allFiles, err := filepath.Glob(filepath.Join(ddevDir, "web-build", "*Dockerfile*"))
				if err != nil {
					return nil, err
				}
				// Filter to only valid Dockerfile patterns
				return slices.DeleteFunc(allFiles, func(file string) bool {
					base := filepath.Base(file)
					return !strings.HasPrefix(base, "Dockerfile") &&
						!strings.HasPrefix(base, "pre.Dockerfile") &&
						!strings.HasPrefix(base, "prepend.Dockerfile")
				}), nil
			},
			displayName: "Web server",
		},
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(ddevDir, "web-entrypoint.d", "*.sh"))
			},
			displayName: "Web server",
		},
		{
			collectFiles: func() ([]string, error) {
				commandsDir := filepath.Join(ddevDir, "commands")
				files, err := fileutil.ListFilesWithDepth(commandsDir, 2)
				if err != nil {
					return nil, nil
				}
				return files, nil
			},
			expectedDdevFiles: func() []string {
				return GetAssetFiles("dotddev_assets/commands", app.GetConfigPath("commands"))
			},
			displayName: "Commands",
		},
		{
			collectFiles: func() ([]string, error) {
				homeadditionsDir := filepath.Join(ddevDir, "homeadditions")
				files, err := fileutil.ListFilesWithDepth(homeadditionsDir, 2)
				if err != nil {
					return nil, nil
				}
				return files, nil
			},
			expectedDdevFiles: func() []string {
				return GetAssetFiles("dotddev_assets/homeadditions", app.GetConfigPath("homeadditions"))
			},
			displayName: "Home additions",
		},
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(ddevDir, "config.*.y*ml"))
			},
			displayName: "Config",
		},
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(ddevDir, "docker-compose.*.y*ml"))
			},
			displayName: "Docker Compose",
		},
		{
			collectFiles: func() ([]string, error) {
				envDotFiles, err := filepath.Glob(filepath.Join(ddevDir, ".env.*"))
				if err != nil {
					return nil, err
				}
				return append([]string{filepath.Join(ddevDir, ".env")}, envDotFiles...), nil
			},
			displayName: "Environment",
		},
	}

	// Gather expected add-on files from the manifest
	// When showAll=false: use this to filter out add-on files (don't show them)
	// When showAll=true: use this to label add-on files with (add-on <name>)
	addonFileMap := make(map[string]string) // file path -> add-on name
	manifest, err := GatherAllManifests(app)
	if err == nil {
		for _, addon := range manifest {
			for _, relPath := range addon.ProjectFiles {
				addonFileMap[app.GetConfigPath(relPath)] = addon.Name
			}
			for _, relPath := range addon.GlobalFiles {
				addonFileMap[filepath.Join(globalconfig.GetGlobalDdevDir(), relPath)] = addon.Name
			}
		}
	}
	addonFileSlice := slices.Collect(maps.Keys(addonFileMap))

	// Execute all checks and collect findings
	type finding struct {
		category string
		files    []fileInfo
	}
	var findings []finding

	for _, check := range checks {
		if check.checkOnlyWhen != nil && !check.checkOnlyWhen() {
			continue
		}
		files, err := check.collectFiles()
		if err != nil {
			util.WarningOnce("%v", err)
			continue
		}
		if len(files) == 0 {
			continue
		}

		// Filter files to find custom config files
		var expectedFiles []string
		if check.expectedDdevFiles != nil {
			expectedFiles = check.expectedDdevFiles()
		}
		// Add-on files are considered expected/standard for the purpose of filtering out from custom config.
		// but only if showAll is false
		if !showAll {
			expectedFiles = append(expectedFiles, addonFileSlice...)
		}
		customFiles := filterCustomConfigFiles(files, expectedFiles, addonFileMap, showAll)

		if len(customFiles) > 0 {
			findings = append(findings, finding{
				category: check.displayName,
				files:    customFiles,
			})
		}
	}

	// Build message for all findings
	if len(findings) > 0 {
		// Group findings by category
		categoryFiles := make(map[string][]fileInfo)
		var categoryOrder []string
		for _, f := range findings {
			if _, exists := categoryFiles[f.category]; !exists {
				categoryOrder = append(categoryOrder, f.category)
				categoryFiles[f.category] = f.files
			} else {
				categoryFiles[f.category] = append(categoryFiles[f.category], f.files...)
			}
		}

		var msgBuilder strings.Builder
		_, _ = fmt.Fprintf(&msgBuilder, "Custom configuration detected in project '%s':\n", app.Name)
		hasUnexpectedFiles := false
		for _, category := range categoryOrder {
			files := categoryFiles[category]
			if len(files) == 1 {
				_, _ = fmt.Fprintf(&msgBuilder, "  • %s: %s\n", category, fileLabel(files[0]))
				if files[0].ddevGenerated && files[0].addonName == "" {
					hasUnexpectedFiles = true
				}
			} else {
				_, _ = fmt.Fprintf(&msgBuilder, "  • %s:\n", category)
				for _, file := range files {
					_, _ = fmt.Fprintf(&msgBuilder, "    - %s\n", fileLabel(file))
					if file.ddevGenerated && file.addonName == "" {
						hasUnexpectedFiles = true
					}
				}
			}
		}
		msgBuilder.WriteString("\nCustom configuration is updated on restart. Run 'ddev restart' if changes don't take effect.")
		if hasUnexpectedFiles {
			msgBuilder.WriteString("\nRemove unexpected '#ddev-generated' comments from files to avoid possible overrides.")
		}
		msgBuilder.WriteString("\nAdd '#ddev-silent-no-warn' comment to files if you don't want to see these warnings.")
		return msgBuilder.String(), true
	}
	return fmt.Sprintf("No custom configuration detected in project '%s'.", app.Name), false
}

// isCustomConfigFile returns true if the file exists and is not marked with
// the standard DDEV signature, OR if the file has DDEV markers but is not in
// the expectedDdevFiles list (indicating an unexpected DDEV-generated file).
// Files with the #ddev-silent-no-warn marker are excluded unless showAll is true.
func isCustomConfigFile(filePath string, expectedDdevFiles []string, hasDdevSig, hasSilentNoWarn bool, showAll bool) bool {
	// Exclude example files
	if strings.HasSuffix(filePath, ".example") {
		return false
	}

	// Files with #ddev-silent-no-warn marker are only excluded if showAll is false
	if !showAll && hasSilentNoWarn {
		return false
	}

	// If file has DDEV marker, check if it's in the expected list
	if hasDdevSig {
		if slices.Contains(expectedDdevFiles, filePath) {
			return false // Expected DDEV file, not custom
		}
		// Has DDEV marker but not in expected list = unexpected = custom
		return true
	}

	// No DDEV marker = custom
	return true
}

// fileInfo represents a config file with metadata
type fileInfo struct {
	path          string
	addonName     string // non-empty if file belongs to an add-on
	ddevGenerated bool
	silentNoWarn  bool
}

// filterCustomConfigFiles returns only files that qualify as custom config files.
// expectedDdevFiles is an optional list of files that are expected to have DDEV markers (core DDEV files).
// addonFileMap maps file paths to their add-on name for files managed by installed add-ons.
// Files with DDEV markers that are NOT in expectedDdevFiles list are considered unexpected and flagged as custom.
// If showAll is true, add-on files are shown with (addon <name>) and silenced files with (#ddev-silent-no-warn).
func filterCustomConfigFiles(files []string, expectedDdevFiles []string, addonFileMap map[string]string, showAll bool) []fileInfo {
	var out []fileInfo
	for _, f := range files {
		// Read file once and check for both markers
		content, err := os.ReadFile(f)
		if err != nil {
			// Skip files that don't exist or can't be read
			continue
		}

		contentStr := string(content)
		hasDdevSig := strings.Contains(contentStr, nodeps.DdevFileSignature)
		hasSilentNoWarn := strings.Contains(contentStr, nodeps.DdevSilentNoWarn)

		addonName := addonFileMap[f] // empty string if not an add-on file

		if isCustomConfigFile(f, expectedDdevFiles, hasDdevSig, hasSilentNoWarn, showAll) {
			out = append(out, fileInfo{
				path:          f,
				addonName:     addonName,
				ddevGenerated: hasDdevSig,
				silentNoWarn:  hasSilentNoWarn,
			})
		}
	}
	return out
}

// fileLabel returns the display string for a file, appending annotation tags as needed.
func fileLabel(f fileInfo) string {
	label := f.path
	if f.addonName != "" {
		label += " (add-on " + f.addonName + ")"
	}
	if f.ddevGenerated {
		if f.addonName == "" {
			label += " (unexpected #ddev-generated)"
		} else {
			label += " (#ddev-generated)"
		}
	}
	if f.silentNoWarn {
		label += " (#ddev-silent-no-warn)"
	}
	return label
}
