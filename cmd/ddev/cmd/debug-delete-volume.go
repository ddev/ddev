package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/util"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// deleteVolumeYes skips confirmation prompts when true (--yes/-y).
var deleteVolumeYes bool

// sharedVolumeNames are volumes used by every DDEV project rather than owned
// by one, so deleting them here would affect projects other than the active one.
var sharedVolumeNames = map[string]bool{
	"ddev-global-cache":         true,
	"ddev-ssh-agent_socket_dir": true,
}

// deleteVolumeChoice is one of the current project's deletable volumes.
type deleteVolumeChoice struct {
	shortName string
	longName  string
}

// getDeleteVolumeChoices returns the current project's non-shared volumes
// that actually exist in Docker, sorted alphabetically with "database" last,
// since it's normally the most destructive one to lose.
func getDeleteVolumeChoices(app *ddevapp.DdevApp) []deleteVolumeChoice {
	var choices []deleteVolumeChoice
	if app.ComposeYaml == nil {
		return choices
	}
	for shortName, v := range app.ComposeYaml.Volumes {
		if sharedVolumeNames[shortName] {
			continue
		}
		if !dockerutil.VolumeExists(v.Name) {
			continue
		}
		choices = append(choices, deleteVolumeChoice{shortName: shortName, longName: v.Name})
	}
	sort.Slice(choices, func(i, j int) bool {
		if (choices[i].shortName == "database") != (choices[j].shortName == "database") {
			return choices[j].shortName == "database"
		}
		return choices[i].shortName < choices[j].shortName
	})
	return choices
}

// DebugDeleteVolumeCmd implements the ddev utility delete-volume command
var DebugDeleteVolumeCmd = &cobra.Command{
	Use:   "delete-volume [volume-name]",
	Short: "Delete a Docker volume belonging to the current project",
	Long: `Deletes a Docker volume belonging to the current project, identified by
either its short docker-compose name (e.g. "solr") or its full Docker volume
name (e.g. "ddev-myproject_solr"). If no name is given, choose one from a list
of the project's volumes. Stops the project first if it's running, since a
container-mounted volume can't be removed.`,
	Example: `ddev utility delete-volume
ddev utility delete-volume solr
ddev utility delete-volume ddev-myproject_solr
ddev utility delete-volume solr --yes`,
	Args: cobra.MaximumNArgs(1),
	ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		_ = app.DockerEnv()
		if err = app.WriteDockerComposeYAML(); err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		choices := getDeleteVolumeChoices(app)
		names := make([]string, len(choices))
		for i, c := range choices {
			names[i] = c.shortName
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(_ *cobra.Command, args []string) {
		requestedName := ""
		if len(args) == 1 {
			requestedName = args[0]
		}

		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Can't find active project: %v", err)
		}

		if sharedVolumeNames[requestedName] {
			util.Failed("'%s' is a shared volume used by all DDEV projects, not just %s, so it can't be deleted with this command", requestedName, app.Name)
		}

		_ = app.DockerEnv()
		if err = app.WriteDockerComposeYAML(); err != nil {
			util.Failed("Failed to get Docker Compose config: %v", err)
		}

		choices := getDeleteVolumeChoices(app)
		if len(choices) == 0 {
			util.Failed("No volumes found for project %s", app.Name)
		}

		var volumeName string
		if requestedName == "" {
			items := make([]string, len(choices))
			for i, c := range choices {
				items[i] = fmt.Sprintf("%s (%s)", c.shortName, c.longName)
			}
			prompt := promptui.Select{
				Label: "Volume to delete",
				Items: items,
			}
			idx, _, err := prompt.Run()
			if err != nil {
				util.Failed("Prompt failed: %v", err)
			}
			volumeName = choices[idx].longName
		} else {
			for _, c := range choices {
				if c.shortName == requestedName || c.longName == requestedName {
					volumeName = c.longName
					break
				}
			}
			if volumeName == "" {
				available := make([]string, len(choices))
				for i, c := range choices {
					available[i] = fmt.Sprintf("%s (%s)", c.shortName, c.longName)
				}
				util.Failed("No volume named '%s' found for project %s; known volumes are:\n%s", requestedName, app.Name, strings.Join(available, "\n"))
			}
		}

		status, _ := app.SiteStatus()
		if status == ddevapp.SiteRunning || status == ddevapp.SiteStarting || status == ddevapp.SitePaused {
			if !deleteVolumeYes && !util.Confirm(fmt.Sprintf("Project %s must be stopped to delete volume '%s'. OK to stop it now?", app.Name, volumeName)) {
				util.Failed("Cannot delete volume '%s' while project %s is running", volumeName, app.Name)
			}
			if err := app.Stop(false, false); err != nil {
				util.Failed("Failed to stop project %s: %v", app.Name, err)
			}
		}

		if !deleteVolumeYes && !util.Confirm(fmt.Sprintf("OK to delete Docker volume '%s'? This cannot be undone.", volumeName)) {
			util.Failed("Cancelled")
		}

		if err := dockerutil.RemoveVolume(volumeName); err != nil {
			util.Failed("Failed to delete volume '%s': %v", volumeName, err)
		}

		util.Success("Deleted Docker volume '%s'", volumeName)
	},
}

func registerDebugDeleteVolumeCmd() {
	DebugDeleteVolumeCmd.Flags().BoolVarP(&deleteVolumeYes, "yes", "y", false, "Yes - skip confirmation prompts")
	DebugCmd.AddCommand(DebugDeleteVolumeCmd)
}
