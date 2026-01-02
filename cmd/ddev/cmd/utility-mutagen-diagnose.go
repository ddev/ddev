package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/version"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/spf13/cobra"
)

// MutagenDiagnoseCmd implements the ddev utility mutagen-diagnose command
var MutagenDiagnoseCmd = &cobra.Command{
	Use:   "mutagen-diagnose",
	Short: "Diagnose Mutagen sync configuration and performance",
	Long: `Analyze Mutagen sync status, volume sizes, and configuration issues.

This command checks:
- Volume sizes and disk usage
- Upload_dirs configuration
- Sync session status and problems
- Performance issues (node_modules, large files, etc.)

Use the --all flag to analyze all Mutagen volumes system-wide.`,
	Example: `ddev utility mutagen-diagnose
ddev ut mutagen-diagnose
ddev utility mutagen-diagnose --all  # Show all Mutagen volumes`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			util.Failed("This command takes no additional arguments")
		}

		showAll, _ := cmd.Flags().GetBool("all")
		exitCode := runMutagenDiagnose(showAll)
		os.Exit(exitCode)
	},
}

func init() {
	MutagenDiagnoseCmd.Flags().BoolP("all", "a", false, "Show all Mutagen volumes system-wide")
	DebugCmd.AddCommand(MutagenDiagnoseCmd)
}

// runMutagenDiagnose performs the diagnostic checks and outputs results
// Returns exit code: 0 if no issues, 1 if issues found
func runMutagenDiagnose(showAll bool) int {
	hasIssues := false

	// Try to load app from current directory (optional)
	app, _ := ddevapp.GetActiveApp("")

	// If showAll is true or we're not in a project, show system-wide analysis
	if showAll || app.AppRoot == "" {
		hasIssues = showAllMutagenVolumes()
		if app.AppRoot == "" {
			// Not in a project directory and not using --all
			if !showAll {
				output.UserOut.Println()
				util.Warning("Not in a DDEV project directory.")
				util.Warning("Used --all flag to analyze all Mutagen volumes system-wide.")
			}
			if hasIssues {
				return 1
			}
			return 0
		}
	}

	// If we have an app, analyze it
	if app.AppRoot != "" {
		// Check if Mutagen is enabled for this project
		if !app.IsMutagenEnabled() {
			util.Warning("Mutagen is not enabled for project '%s'\n", app.Name)
			util.Warning("To enable Mutagen:")
			util.Warning("  ddev config --performance-mode=mutagen")
			util.Warning("  ddev restart")
			return 0
		}

		// Check if project is running - start it if needed
		status, _ := app.SiteStatus()
		if status != ddevapp.SiteRunning {
			output.UserOut.Printf("Project '%s' is not running. Starting project to enable sync diagnostics...", app.Name)
			output.UserOut.Println()
			err := app.Start()
			if err != nil {
				util.Failed("Failed to start project: %v", err)
				return 1
			}
		}

		projectHasIssues := showProjectDiagnostics(app)
		if projectHasIssues {
			hasIssues = true
		}
	}

	if hasIssues {
		return 1
	}
	return 0
}

// showAllMutagenVolumes displays information about all Mutagen volumes on the system
// Returns true if issues were found
func showAllMutagenVolumes() bool {
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("System-wide Mutagen Volume Analysis")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println()

	volumes, totalSize, err := ddevapp.GetAllMutagenVolumes()
	if err != nil {
		util.Error("Failed to get Mutagen volumes: %v", err)
		return true
	}

	if len(volumes) == 0 {
		output.UserOut.Println("  No Mutagen volumes found on this system.")
		output.UserOut.Println()
		return false
	}

	hasIssues := false

	// Display each volume
	for _, vol := range volumes {
		sizeWarning := ""
		fiveGB := int64(5 * 1024 * 1024 * 1024)
		tenGB := int64(10 * 1024 * 1024 * 1024)

		if vol.SizeBytes > tenGB {
			sizeWarning = " ✗ CRITICAL: Very large volume"
			hasIssues = true
		} else if vol.SizeBytes > fiveGB {
			sizeWarning = " ⚠ WARNING: Large volume"
		} else {
			sizeWarning = " ✓"
		}

		output.UserOut.Printf("  %s Project: %s - Volume: %s (%s)", sizeWarning, vol.Project, vol.Name, vol.SizeHuman)
	}

	output.UserOut.Println()
	totalSizeHuman := util.FormatBytes(totalSize)
	output.UserOut.Printf("  Total Mutagen disk usage: %s across %d project(s)", totalSizeHuman, len(volumes))

	if hasIssues {
		output.UserOut.Println()
		output.UserOut.Println("  Recommendations:")
		output.UserOut.Println("    - Review large volumes and consider excluding unnecessary directories")
		output.UserOut.Println("    - Configure upload_dirs to exclude user-generated files")
		output.UserOut.Println("    - Exclude node_modules and other large dependency directories")
	}

	return hasIssues
}

// showProjectDiagnostics displays detailed diagnostics for a specific project
// Returns true if issues were found
func showProjectDiagnostics(app *ddevapp.DdevApp) bool {
	output.UserOut.Printf("Mutagen Diagnostics for Project: %s", app.Name)
	output.UserOut.Println()

	// Run comprehensive diagnostics
	result := ddevapp.DiagnoseMutagenConfiguration(app)
	hasIssues := result.IssueCount > 0

	// Show Sync Status section
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Sync Status")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Session status
	statusSymbol := getStatusSymbol(result.SyncStatus)
	statusDisplay := result.SyncStatus
	if result.SyncStatusDetail != "" && result.SyncStatusDetail != result.SyncStatus {
		statusDisplay = fmt.Sprintf("%s (%s)", result.SyncStatus, result.SyncStatusDetail)
	}
	output.UserOut.Printf("  %s Session status: %s\n", statusSymbol, statusDisplay)

	// Labels match
	if result.LabelsMatch {
		output.UserOut.Println("  ✓ Session and volume labels match")
	} else {
		output.UserOut.Println("  ✗ Session and volume labels DO NOT match")
		output.UserOut.Println("    → Run: ddev mutagen reset && ddev restart")
	}

	// Mutagen version
	mutagenVersion, _ := version.GetLiveMutagenVersion()
	requiredVersion := versionconstants.RequiredMutagenVersion
	versionMatch := mutagenVersion == requiredVersion
	versionSymbol := "✓"
	if !versionMatch {
		versionSymbol = "⚠"
	}
	output.UserOut.Printf("  %s Mutagen version: %s", versionSymbol, mutagenVersion)
	if !versionMatch {
		output.UserOut.Printf(" (required: %s)", requiredVersion)
	}
	output.UserOut.Println()

	// Show problems if any
	if len(result.Problems) > 0 {
		output.UserOut.Println("  Problems detected:")
		for _, problem := range result.Problems {
			output.UserOut.Printf("    ✗ %s", problem)
			output.UserOut.Println()
		}
	}

	// Show Volume Size Analysis section
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Volume Size Analysis")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	volumeName := ddevapp.GetMutagenVolumeName(app)
	if result.VolumeCritical {
		output.UserOut.Printf("  ✗ Project volume: %s (%s)\n", volumeName, result.VolumeSizeHuman)
		output.UserOut.Println("    CRITICAL: Very large volume detected (>10GB)")
		output.UserOut.Println("    Consider excluding unnecessary directories from sync.")
	} else if result.VolumeWarning {
		output.UserOut.Printf("  ⚠ Project volume: %s (%s)\n", volumeName, result.VolumeSizeHuman)
		output.UserOut.Println("    Large volume detected (>5GB)")
		output.UserOut.Println("    Consider excluding unnecessary directories from sync.")
	} else {
		output.UserOut.Printf("  ✓ Project volume: %s (%s)\n", volumeName, result.VolumeSizeHuman)
	}

	// Show total across all projects
	volumes, totalSize, err := ddevapp.GetAllMutagenVolumes()
	if err == nil && len(volumes) > 1 {
		totalSizeHuman := util.FormatBytes(totalSize)
		output.UserOut.Printf("  ℹ Total Mutagen volumes: %s across %d project(s)", totalSizeHuman, len(volumes))
	}

	output.UserOut.Println()

	// Show Upload Directories Configuration section
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Upload Directories Configuration")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if result.UploadDirsConfigured {
		output.UserOut.Printf("  ✓ upload_dirs configured: %s", strings.Join(result.UploadDirs, ", "))
	} else {
		if result.UploadDirsSuggestion != "" {
			output.UserOut.Printf("  ✗ No upload_dirs configured for %s project", app.Type)
			output.UserOut.Printf("    → Suggestion: Add 'upload_dirs: [\"%s\"]' to .ddev/config.yaml", result.UploadDirsSuggestion)
			output.UserOut.Println("    → Then run: ddev mutagen reset && ddev restart")
		} else {
			output.UserOut.Println("  ℹ No upload_dirs configured (may not be needed for this project type)")
		}
	}

	output.UserOut.Println()

	// Show Performance Analysis section
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Performance Analysis")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Show ignore pattern issues/warnings
	if len(result.IgnoreIssues) > 0 {
		for _, issue := range result.IgnoreIssues {
			output.UserOut.Printf("  ✗ %s\n", issue)
		}
	}

	// Show warnings from pattern checks with specific recommendations
	// Group by directory type (e.g., all node_modules together)
	warningsByType := make(map[string][]ddevapp.IgnorePatternWarning)
	for _, patternWarn := range result.IgnorePatternWarnings {
		warningsByType[patternWarn.Directory] = append(warningsByType[patternWarn.Directory], patternWarn)
	}

	for dirType, warnings := range warningsByType {
		if len(warnings) == 1 {
			output.UserOut.Printf("  ⚠ %s directory exists but is not excluded from sync (%s)", dirType, warnings[0].Reason)
		} else {
			output.UserOut.Printf("  ⚠ %d %s directories exist but are not excluded from sync (%s)", len(warnings), dirType, warnings[0].Reason)
		}

		output.UserOut.Println("    → Add to .ddev/config.yaml (include existing upload_dirs if any):")
		output.UserOut.Println("      upload_dirs:")
		// Show existing upload_dirs first
		if len(result.UploadDirs) > 0 {
			for _, dir := range result.UploadDirs {
				output.UserOut.Printf("        - %s", dir)
			}
		}
		// Show all paths for this directory type
		for _, warn := range warnings {
			output.UserOut.Printf("        - %s", warn.UploadDirsPath)
		}
		output.UserOut.Println("    → Then run: ddev restart")
		output.UserOut.Println("    → (Files in upload_dirs are not synced by Mutagen but available via Docker bind-mount)")
	}

	// Show other warnings without specific recommendations
	for _, warning := range result.IgnoreWarnings {
		// Skip pattern warnings as they're shown above
		skipWarning := false
		for _, patternWarn := range result.IgnorePatternWarnings {
			if strings.Contains(warning, patternWarn.Directory) {
				skipWarning = true
				break
			}
		}
		if !skipWarning {
			output.UserOut.Printf("  ⚠ %s", warning)
		}
	}

	if len(result.IgnoreIssues) == 0 && len(result.IgnoreWarnings) == 0 {
		output.UserOut.Println("  ✓ No obvious performance issues detected")
	}

	// Show configuration status
	if result.MutagenYmlCustomized {
		output.UserOut.Println("  ℹ mutagen.yml has been customized (no #ddev-generated marker)")
	} else {
		output.UserOut.Println("  ✓ mutagen.yml using default configuration")
	}

	output.UserOut.Println()

	// Show Summary section
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Summary")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if result.WarningCount > 0 {
		output.UserOut.Printf("  ⚠ %d warning(s) found", result.WarningCount)
	}
	if result.IssueCount > 0 {
		output.UserOut.Printf("  ✗ %d issue(s) found", result.IssueCount)
	}
	if result.IssueCount == 0 && result.WarningCount == 0 {
		output.UserOut.Println("  ✓ No issues found - Mutagen configuration looks good!")
	}

	output.UserOut.Println()

	// Show recommendations if there are issues
	if result.IssueCount > 0 || result.WarningCount > 0 {
		output.UserOut.Println("Recommendations:")
		recommendations := []string{}

		if !result.UploadDirsConfigured && result.UploadDirsSuggestion != "" {
			recommendations = append(recommendations, "Configure upload_dirs for better performance")
		}
		if len(result.IgnoreWarnings) > 0 {
			for _, warning := range result.IgnoreWarnings {
				if strings.Contains(warning, "node_modules") {
					recommendations = append(recommendations, "Exclude node_modules from Mutagen sync")
				}
			}
		}
		if result.VolumeWarning || result.VolumeCritical {
			recommendations = append(recommendations, "Consider excluding large directories to reduce volume size")
		}
		if !result.LabelsMatch {
			recommendations = append(recommendations, "Run 'ddev mutagen reset' to fix compatibility issues")
		}

		for i, rec := range recommendations {
			output.UserOut.Printf("  %d. %s", i+1, rec)
		}

		output.UserOut.Println()
		output.UserOut.Println("Run 'ddev mutagen reset && ddev restart' after making configuration changes.")
		output.UserOut.Println()
	}

	return hasIssues
}

// getStatusSymbol returns the appropriate symbol for a sync status
func getStatusSymbol(status string) string {
	switch status {
	case "ok":
		return "✓"
	case "paused":
		return "ℹ"
	case "problems":
		return "⚠"
	case "failing":
		return "✗"
	case "nosession":
		return "✗"
	case "not enabled":
		return "ℹ"
	default:
		return "?"
	}
}
