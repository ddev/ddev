package cmd

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/styles"
	"github.com/ddev/ddev/pkg/util"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

// AddonListCmd is the "ddev add-on list" command
var AddonListCmd = &cobra.Command{
	Use:   "list",
	Args:  cobra.NoArgs,
	Short: "List available or installed DDEV add-ons",
	Long:  `List available or installed DDEV add-ons. To list installed add-ons, use '--installed'`,
	Example: `ddev add-on list
ddev add-on list --installed
ddev add-on list --installed --project my-project
`,
	Run: func(cmd *cobra.Command, _ []string) {
		if cmd.Flags().Changed("project") && !cmd.Flags().Changed("installed") {
			util.Failed("--project flag can only be used with --installed flag")
		}

		// List installed add-ons
		if cmd.Flags().Changed("installed") {
			app, err := ddevapp.GetActiveApp(cmd.Flag("project").Value.String())
			if err != nil {
				util.Failed("Unable to get project %v: %v", cmd.Flag("project").Value.String(), err)
			}

			ListInstalledAddons(app)
			return
		}

		// List available add-ons from registry
		// these do not require an app context
		addons, err := ddevapp.ListAvailableAddonsFromRegistry()
		if err != nil {
			util.Failed("Failed to list available add-ons: %v", err)
		}
		if len(addons) == 0 {
			util.Warning("No DDEV add-ons found in registry.")
			return
		}
		wrapTable, _ := cmd.Flags().GetBool("wrap-table")
		out := renderRepositoryList(addons, wrapTable)
		output.UserOut.WithField("raw", addons).Print(out)
	},
}

// ListInstalledAddons show the add-ons that have a manifest file
func ListInstalledAddons(app *ddevapp.DdevApp) {
	manifests := ddevapp.GetInstalledAddons(app)

	var out bytes.Buffer
	t := table.NewWriter()
	t.SetOutputMirror(&out)
	styles.SetGlobalTableStyle(t, false)

	if !globalconfig.DdevGlobalConfig.SimpleFormatting {
		termWidth, _ := nodeps.GetTerminalWidthHeight()
		if termWidth == 0 {
			termWidth = 80
		}
		// Table overhead for 4 columns: | col | col | col | col |
		const tableOverhead = 14
		usableWidth := termWidth - tableOverhead
		// Fixed widths for narrow columns
		versionWidth := 10
		dateWidth := 20
		addonWidth := 20
		repoWidth := max(20, usableWidth-addonWidth-versionWidth-dateWidth)

		snip := func(col string, maxLen int) string { return text.Snip(col, maxLen, "…") }
		t.SetColumnConfigs([]table.ColumnConfig{
			{Name: "Add-on", WidthMax: addonWidth, WidthMaxEnforcer: snip},
			{Name: "Version", WidthMax: versionWidth, WidthMaxEnforcer: snip},
			{Name: "Repository", WidthMax: repoWidth, WidthMaxEnforcer: snip},
			{Name: "Date Installed", WidthMax: dateWidth, WidthMaxEnforcer: snip},
		})
	}
	t.AppendHeader(table.Row{"Add-on", "Version", "Repository", "Date Installed"})

	// Loop through the directories in the .ddev/addon-metadata directory
	for _, addon := range manifests {
		repoDisplay := addon.Repository
		if addon.Repository != "" {
			repoDisplay = output.Hyperlink("https://github.com/"+addon.Repository, addon.Repository)
		}
		t.AppendRow(table.Row{addon.Name, addon.Version, repoDisplay, addon.InstallDate})
	}
	if t.Length() == 0 {
		output.UserOut.Println("No registered add-ons were found.")
		return
	}
	t.Render()
	output.UserOut.WithField("raw", manifests).Println(out.String())
}

// renderRepositoryList renders the found list of addons from the registry
func renderRepositoryList(addons []types.Addon, wrapTable bool) string {
	var out bytes.Buffer

	t := table.NewWriter()
	t.SetOutputMirror(&out)
	styles.SetGlobalTableStyle(t, false)

	termWidth, _ := nodeps.GetTerminalWidthHeight()
	if termWidth == 0 {
		termWidth = 80
	}
	// Table overhead for 2 columns: | col | col |
	const tableOverhead = 7
	usableWidth := termWidth - tableOverhead
	addonWidth := max(30, usableWidth*3/10)
	descWidth := max(30, usableWidth-addonWidth)

	snip := func(col string, maxLen int) string { return text.Snip(col, maxLen, "…") }
	if wrapTable {
		t.SetColumnConfigs([]table.ColumnConfig{
			{Name: "Add-on"},
			{Name: "Description"},
		})
	} else {
		t.SetColumnConfigs([]table.ColumnConfig{
			{Name: "Add-on", WidthMax: addonWidth, WidthMaxEnforcer: snip},
			{Name: "Description", WidthMax: descWidth},
		})
	}

	sort.Slice(addons, func(i, j int) bool {
		return strings.Compare(strings.ToLower(addons[i].Title), strings.ToLower(addons[j].Title)) == -1
	})
	t.AppendHeader(table.Row{"Add-on", "Description"})

	for _, addon := range addons {
		d := addon.Description
		if addon.Type == "official" {
			d = d + "*"
		}
		title := output.Hyperlink(addon.GitHubURL, addon.Title)
		if wrapTable {
			t.AppendRow([]any{title, d})
		} else {
			t.AppendRow([]any{title, text.WrapSoft(d, descWidth)})
		}
	}

	t.Render()

	return out.String() + fmt.Sprintf("%d add-ons found. Those marked with '*' are officially maintained by DDEV.", len(addons))
}

func init() {
	AddonListCmd.Flags().Bool("all", false, `List unofficial DDEV add-ons for in addition to the official ones`)
	_ = AddonListCmd.Flags().MarkDeprecated("all", "this flag no longer has any effect. All add-ons are shown by default.")
	AddonListCmd.Flags().Bool("installed", false, `Show installed DDEV add-ons`)
	AddonListCmd.Flags().String("project", "", "Name of the project to list the add-ons for. Can only be used with `--installed`")
	_ = AddonListCmd.RegisterFlagCompletionFunc("project", ddevapp.GetProjectNamesFunc("all", 0))
	AddonListCmd.Flags().BoolP("wrap-table", "W", false, "Display table with wrapped text instead of truncating.")

	AddonCmd.AddCommand(AddonListCmd)
}
