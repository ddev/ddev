package ddevapp

import (
	"fmt"
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

// CheckCustomConfig warns the user if any custom configuration files are in use.
// If showAll is true, files with #ddev-silent-no-warn marker are included in the output.
func (app *DdevApp) CheckCustomConfig(showAll bool) {
	ddevDir := filepath.Dir(app.ConfigPath)

	// Gather expected add-on files from the manifest
	// When showAll=false: use this to filter out add-on files (don't show them)
	// When showAll=true: use this to mark add-on files with (#ddev-generated)
	var expectedAddonFiles []string
	manifest, err := GatherAllManifests(app)
	if err == nil {
		for _, addon := range manifest {
			for _, relPath := range addon.ProjectFiles {
				expectedAddonFiles = append(expectedAddonFiles, app.GetConfigPath(relPath))
			}
		}
	}

	// Define all configuration checks
	checks := []customConfigCheck{
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(globalconfig.GetGlobalDdevDir(), "router-compose.*.yaml"))
			},
			checkOnlyWhen: func() bool { return !slices.Contains(app.OmitContainersGlobal, RouterComposeProjectName) },
			displayName:   "Router (global)",
		},
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(globalconfig.GetGlobalDdevDir(), "ssh-auth-compose.*.yaml"))
			},
			checkOnlyWhen: func() bool { return !slices.Contains(app.OmitContainersGlobal, SSHAuthName) },
			displayName:   "Global SSH agent",
		},
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik", "static_config.*.yaml"))
			},
			checkOnlyWhen: func() bool { return !slices.Contains(app.OmitContainersGlobal, RouterComposeProjectName) },
			displayName:   "Router (global)",
		},
		{
			collectFiles: func() ([]string, error) {
				customGlobalConfigDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik", "custom-global-config")
				if !fileutil.IsDirectory(customGlobalConfigDir) {
					return nil, nil
				}
				return fileutil.ListFilesInDirFullPath(customGlobalConfigDir, true)
			},
			expectedDdevFiles: func() []string {
				return []string{filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik", "custom-global-config", "README.txt")}
			},
			checkOnlyWhen: func() bool { return !slices.Contains(app.OmitContainersGlobal, RouterComposeProjectName) },
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
			expectedDdevFiles: func() []string {
				if showAll {
					return nil
				}
				return expectedAddonFiles
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
				return []string{
					app.GetConfigPath("providers/acquia.yaml"),
					app.GetConfigPath("providers/lagoon.yaml"),
					app.GetConfigPath("providers/pantheon.yaml"),
					app.GetConfigPath("providers/platform.yaml"),
					app.GetConfigPath("providers/upsun.yaml"),
				}
			},
			displayName: "Hosting providers",
		},
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(ddevDir, "share-providers", "*.sh"))
			},
			expectedDdevFiles: func() []string {
				return []string{
					app.GetConfigPath("share-providers/cloudflared.sh"),
					app.GetConfigPath("share-providers/ngrok.sh"),
				}
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
			expectedDdevFiles: func() []string {
				if showAll {
					return nil
				}
				return expectedAddonFiles
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
				return filepath.Glob(filepath.Join(ddevDir, "config.*.y*ml"))
			},
			expectedDdevFiles: func() []string {
				if showAll {
					return nil
				}
				return expectedAddonFiles
			},
			displayName: "Config",
		},
		{
			collectFiles: func() ([]string, error) {
				return filepath.Glob(filepath.Join(ddevDir, "docker-compose.*.y*ml"))
			},
			expectedDdevFiles: func() []string {
				if showAll {
					return nil
				}
				return expectedAddonFiles
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
		customFiles := filterCustomConfigFiles(files, expectedFiles, expectedAddonFiles, showAll)

		if len(customFiles) > 0 {
			findings = append(findings, finding{
				category: check.displayName,
				files:    customFiles,
			})
		}
	}

	// Display all findings as a single grouped message
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

		var message strings.Builder
		_, _ = fmt.Fprintf(&message, "Custom configuration detected in project '%s':\n", app.Name)
		for _, category := range categoryOrder {
			files := categoryFiles[category]
			if len(files) == 1 {
				filePath := files[0].path
				if files[0].ddevGenerated {
					filePath += " (#ddev-generated)"
				}
				if files[0].silentNoWarn {
					filePath += " (#ddev-silent-no-warn)"
				}
				_, _ = fmt.Fprintf(&message, "  • %s: %s\n", category, filePath)
			} else {
				_, _ = fmt.Fprintf(&message, "  • %s:\n", category)
				for _, file := range files {
					filePath := file.path
					if file.ddevGenerated {
						filePath += " (#ddev-generated)"
					}
					if file.silentNoWarn {
						filePath += " (#ddev-silent-no-warn)"
					}
					_, _ = fmt.Fprintf(&message, "    - %s\n", filePath)
				}
			}
		}
		if !showAll {
			message.WriteString("\nCustom configuration is updated on restart. Run 'ddev restart' if changes don't take effect.")
			message.WriteString("\nAdd '#ddev-silent-no-warn' comment to files if you don't want to see these warnings.")
		}
		util.Warning(message.String())
	} else if showAll {
		// Only show success message when explicitly checking (showAll=true)
		util.Success("No custom configuration detected in project '%s'.", app.Name)
	}
}

// isCustomConfigFile returns true if the file exists and is not marked with
// the standard DDEV signature, OR if the file has DDEV markers but is not in
// the expectedDdevFiles list (indicating a fake/suspicious DDEV-generated file).
// Files with the #ddev-silent-no-warn marker are excluded unless showAll is true.
func isCustomConfigFile(filePath string, expectedDdevFiles []string, showAll bool) bool {
	if !fileutil.FileExists(filePath) {
		return false
	}

	// Exclude example files
	if strings.HasSuffix(filePath, ".example") {
		return false
	}

	// Files with #ddev-silent-no-warn marker are only excluded if showAll is false
	if !showAll {
		silentNoWarnFound, _ := fileutil.FgrepStringInFile(filePath, nodeps.DdevSilentNoWarn)
		if silentNoWarnFound {
			return false
		}
	}

	sigFound, _ := fileutil.FgrepStringInFile(filePath, nodeps.DdevFileSignature)

	// If file has DDEV marker, check if it's in the expected list
	if sigFound {
		for _, expected := range expectedDdevFiles {
			if filePath == expected {
				return false // Expected DDEV file, not custom
			}
		}
		// Has DDEV marker but not in expected list = fake/suspicious = custom
		return true
	}

	// No DDEV marker = custom
	return true
}

// fileInfo represents a config file with metadata
type fileInfo struct {
	path          string
	ddevGenerated bool
	silentNoWarn  bool
}

// filterCustomConfigFiles returns only files that qualify as custom config files.
// expectedDdevFiles is an optional list of files that are expected to have DDEV markers (core DDEV files).
// expectedAddonFiles is a list of addon-generated files that should be marked when showAll=true.
// Files with DDEV markers that are NOT in expectedDdevFiles list are considered suspicious and flagged as custom.
// If showAll is true, add-on files are shown with (#ddev-generated) marker and silenced files with (#ddev-silent-no-warn) marker.
func filterCustomConfigFiles(files []string, expectedDdevFiles []string, expectedAddonFiles []string, showAll bool) []fileInfo {
	var out []fileInfo
	for _, f := range files {
		// Check if file is an add-on file
		isAddonFile := slices.Contains(expectedAddonFiles, f)

		// Check if file has markers
		isSilenced := false
		isDdevGenerated := false
		if showAll {
			silentNoWarnFound, _ := fileutil.FgrepStringInFile(f, nodeps.DdevSilentNoWarn)
			isSilenced = silentNoWarnFound

			// Mark add-on files with #ddev-generated
			if isAddonFile {
				ddevGenFound, _ := fileutil.FgrepStringInFile(f, nodeps.DdevFileSignature)
				isDdevGenerated = ddevGenFound
			}
		}

		if isCustomConfigFile(f, expectedDdevFiles, showAll) {
			out = append(out, fileInfo{
				path:          f,
				ddevGenerated: isDdevGenerated,
				silentNoWarn:  isSilenced,
			})
		}
	}
	return out
}
