package cmd

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/styles"
	"github.com/ddev/ddev/pkg/util"
	"github.com/google/go-github/v52/github"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

// AddonListCmd is the "ddev add-on list" command
var AddonListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available or installed DDEV add-ons",
	Long:  `List available or installed DDEV add-ons. Without '--all' it shows only official DDEV add-ons. To list installed add-ons, use '--installed'`,
	Example: `ddev add-on list
ddev add-on list --all
ddev add-on list --installed
`,
	Run: func(cmd *cobra.Command, _ []string) {
		// List installed add-ons
		if cmd.Flags().Changed("installed") {
			app, err := ddevapp.GetActiveApp("")
			if err != nil {
				util.Failed("Unable to find active project: %v", err)
			}

			ListInstalledAddons(app)
			return
		}

		// List available add-ons
		// these do not require an app context
		repos, err := ddevapp.ListAvailableAddons(!cmd.Flags().Changed("all"))
		if err != nil {
			util.Failed("Failed to list available add-ons: %v", err)
		}
		if len(repos) == 0 {
			util.Warning("No DDEV add-ons found with GitHub topic 'ddev-get'.")
			return
		}
		out := renderRepositoryList(repos)
		output.UserOut.WithField("raw", repos).Print(out)
	},
}

// ListInstalledAddons() show the add-ons that have a manifest file
func ListInstalledAddons(app *ddevapp.DdevApp) {

	manifests := ddevapp.GetInstalledAddons(app)

	var out bytes.Buffer
	t := table.NewWriter()
	t.SetOutputMirror(&out)
	styles.SetGlobalTableStyle(t)

	if !globalconfig.DdevGlobalConfig.SimpleFormatting {
		t.SetColumnConfigs([]table.ColumnConfig{
			{
				Name: "Add-on",
			},
			{
				Name: "Version",
			},
			{
				Name: "Repository",
			},
			{
				Name: "Date Installed",
			},
		})
	}
	t.AppendHeader(table.Row{"Add-on", "Version", "Repository", "Date Installed"})

	// Loop through the directories in the .ddev/addon-metadata directory
	for _, addon := range manifests {
		t.AppendRow(table.Row{addon.Name, addon.Version, addon.Repository, addon.InstallDate})
	}
	if t.Length() == 0 {
		output.UserOut.Println("No registered add-ons were found.")
		return
	}
	t.Render()
	output.UserOut.WithField("raw", manifests).Println(out.String())
}

// renderRepositoryList renders the found list of repositories
func renderRepositoryList(repos []*github.Repository) string {
	var out bytes.Buffer

	t := table.NewWriter()
	t.SetOutputMirror(&out)
	styles.SetGlobalTableStyle(t)
	//tWidth, _ := nodeps.GetTerminalWidthHeight()
	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Name: "Service",
		},
		{
			Name: "Description",
		},
	})
	sort.Slice(repos, func(i, j int) bool {
		return strings.Compare(strings.ToLower(repos[i].GetFullName()), strings.ToLower(repos[j].GetFullName())) == -1
	})
	t.AppendHeader(table.Row{"Add-on", "Description"})

	for _, repo := range repos {
		d := repo.GetDescription()
		if repo.GetOwner().GetLogin() == globalconfig.DdevGithubOrg {
			d = d + "*"
		}
		t.AppendRow([]interface{}{repo.GetFullName(), text.WrapSoft(d, 50)})
	}

	t.Render()

	return out.String() + fmt.Sprintf("%d repositories found. Add-ons marked with '*' are officially maintained DDEV add-ons.", len(repos))
}

func init() {
	AddonListCmd.Flags().Bool("all", false, `List unofficial DDEV add-ons for in addition to the official ones`)
	AddonListCmd.Flags().Bool("installed", false, `Show installed DDEV add-ons`)
	// Can't do 'ddev add-on list --all --installed', because the "installed" flag already shows *all* installed add-ons
	AddonListCmd.MarkFlagsMutuallyExclusive("all", "installed")

	AddonCmd.AddCommand(AddonListCmd)
}
