package cmd

import (
	"os"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// addonUpdateResult describes the outcome of checking/updating a single add-on,
// used to build the `raw` JSON output.
type addonUpdateResult struct {
	Name       string `json:"name"`
	Repository string `json:"repository"`
	OldVersion string `json:"old_version"`
	NewVersion string `json:"new_version,omitempty"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

// AddonUpdateCmd is the "ddev add-on update" command
var AddonUpdateCmd = &cobra.Command{
	Use:   "update [addonName...]",
	Args:  cobra.ArbitraryArgs,
	Short: "Update installed add-ons to their latest available version",
	Long: `Check installed add-ons for a newer release and install it. If no add-on names
are provided, all installed add-ons are checked.

Only add-ons installed from a GitHub repository can be checked automatically; add-ons
installed from a local directory or from a non-GitHub tarball URL are skipped.

Use '--dry-run' to see what would be updated without installing anything.`,
	Example: `ddev add-on update
ddev add-on update redis
ddev add-on update ddev/ddev-redis
ddev add-on update --dry-run
ddev add-on update --project my-project
`,
	Run: func(cmd *cobra.Command, args []string) {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		verbose, _ := cmd.Flags().GetBool("verbose")

		app, err := ddevapp.GetActiveApp(cmd.Flag("project").Value.String())
		if err != nil {
			util.Failed("Unable to get project %v: %v", cmd.Flag("project").Value.String(), err)
		}
		err = os.Chdir(app.AppRoot)
		if err != nil {
			util.Failed("Unable to change directory to project root %s: %v", app.AppRoot, err)
		}
		_ = app.DockerEnv()

		manifests := ddevapp.GetInstalledAddons(app)
		if len(manifests) == 0 {
			util.Warning("No add-ons are installed in project '%s'.", app.Name)
			return
		}

		if len(args) > 0 {
			allManifests, err := ddevapp.GatherAllManifests(app)
			if err != nil {
				util.Failed("Unable to gather add-on manifests: %v", err)
			}
			seen := map[string]bool{}
			var filtered []ddevapp.AddonManifest
			for _, arg := range args {
				manifest, ok := allManifests[arg]
				if !ok {
					util.Warning("Add-on '%s' is not installed in project '%s'", arg, app.Name)
					continue
				}
				if !seen[manifest.Name] {
					seen[manifest.Name] = true
					filtered = append(filtered, manifest)
				}
			}
			manifests = filtered
		}

		var results []addonUpdateResult
		var toUpdate []addonUpdateResult

		for _, manifest := range manifests {
			result := addonUpdateResult{Name: manifest.Name, Repository: manifest.Repository, OldVersion: manifest.Version}

			if !ddevapp.IsGithubRef(manifest.Repository) {
				util.Warning("Skipping '%s': not installed from a GitHub repository, cannot check for updates", manifest.Name)
				result.Status = "skipped"
				results = append(results, result)
				continue
			}

			_, latestVersion, err := ddevapp.GetAddonTarballURL(manifest.Repository, "", false, 0)
			if err != nil {
				util.Warning("Skipping '%s': unable to determine the latest version: %v", manifest.Name, err)
				result.Status = "skipped"
				result.Error = err.Error()
				results = append(results, result)
				continue
			}

			result.NewVersion = latestVersion
			if latestVersion == manifest.Version {
				if verbose {
					util.Success("'%s' is already up to date (%s)", manifest.Name, manifest.Version)
				}
				result.Status = "up-to-date"
				results = append(results, result)
				continue
			}

			result.Status = "outdated"
			toUpdate = append(toUpdate, result)
		}

		if len(toUpdate) == 0 {
			output.UserOut.WithField("raw", results).Println("All add-ons are up to date.")
			return
		}

		if dryRun {
			for _, r := range toUpdate {
				util.Success("'%s' can be updated: %s -> %s", r.Name, r.OldVersion, r.NewVersion)
			}
			results = append(results, toUpdate...)
			output.UserOut.WithField("raw", results).Printf("%d add-on(s) can be updated.", len(toUpdate))
			return
		}

		var updatedNames []string
		var failedNames []string
		for _, r := range toUpdate {
			util.Success("\nUpdating '%s': %s -> %s", r.Name, r.OldVersion, r.NewVersion)
			err := ddevapp.InstallAddonFromGitHub(app, r.Repository, "", verbose)
			if err != nil {
				util.Warning("Unable to update '%s': %v", r.Name, err)
				r.Status = "failed"
				r.Error = err.Error()
				failedNames = append(failedNames, r.Name)
				results = append(results, r)
				continue
			}

			err = ddevapp.ProcessRuntimeDependencies(app, r.Name, verbose)
			if err != nil {
				util.Warning("Unable to process runtime dependencies for '%s': %v", r.Name, err)
			}

			r.Status = "updated"
			updatedNames = append(updatedNames, r.Name)
			results = append(results, r)
		}

		err = app.CleanupConfigurationFiles()
		if err != nil {
			util.Warning("Unable to clean up temporary configuration files: %v", err)
		}

		if len(updatedNames) > 0 {
			output.UserOut.WithField("raw", results).Printf("\nUpdated %d add-on(s): %s\nUse `ddev restart` to enable updates.", len(updatedNames), strings.Join(updatedNames, ", "))
		}
		if len(failedNames) > 0 {
			util.Failed("Failed to update %d add-on(s): %s", len(failedNames), strings.Join(failedNames, ", "))
		}
	},
}

func init() {
	AddonUpdateCmd.Flags().Bool("dry-run", false, "Show which add-ons would be updated without installing anything")
	_ = AddonUpdateCmd.RegisterFlagCompletionFunc("dry-run", configCompletionFunc([]string{"true", "false"}))
	AddonUpdateCmd.Flags().BoolP("verbose", "v", false, "Extended/verbose output")
	_ = AddonUpdateCmd.RegisterFlagCompletionFunc("verbose", configCompletionFunc([]string{"true", "false"}))
	AddonUpdateCmd.Flags().String("project", "", "Name of the project to update the add-ons for")
	_ = AddonUpdateCmd.RegisterFlagCompletionFunc("project", ddevapp.GetProjectNamesFunc("all", 0))

	AddonCmd.AddCommand(AddonUpdateCmd)
}
