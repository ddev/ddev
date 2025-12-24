package cmd

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/styles"
	"github.com/ddev/ddev/pkg/util"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

// AddonSearchCmd is the "ddev add-on search" command
var AddonSearchCmd = &cobra.Command{
	Use:   "search <search-term> [additional-terms...]",
	Args:  cobra.MinimumNArgs(1),
	Short: "Search available DDEV add-ons",
	Long:  `Search available DDEV add-ons by name or description.`,
	Example: `ddev add-on search redis
ddev add-on search database
ddev add-on search redis insight
ddev add-on search "redis insight"`,
	Run: func(cmd *cobra.Command, args []string) {
		searchTerms := strings.Join(args, " ")

		// Get available add-ons from the registry
		addons, err := ddevapp.ListAvailableAddonsFromRegistry()
		if err != nil {
			util.Failed("Failed to search available add-ons: %v", err)
		}

		// Filter addons based on search keywords
		var filteredAddons []types.Addon

		// Extract all keywords from all arguments
		var keywords []string
		for _, arg := range args {
			// Split each argument by spaces to handle both quoted and unquoted cases
			words := strings.Fields(strings.ToLower(arg))
			keywords = append(keywords, words...)
		}

		for _, addon := range addons {
			addonTitle := strings.ToLower(addon.Title)
			addonDesc := strings.ToLower(addon.Description)
			searchText := addonTitle + " " + addonDesc

			// Check if all keywords appear in name or description
			matches := true
			for _, keyword := range keywords {
				if !strings.Contains(searchText, keyword) {
					matches = false
					break
				}
			}

			if matches {
				filteredAddons = append(filteredAddons, addon)
			}
		}

		if len(filteredAddons) == 0 {
			output.UserOut.Printf("No add-ons found matching '%s'\n", searchTerms)
			return
		}

		out := renderSearchResults(filteredAddons, searchTerms)
		output.UserOut.WithField("raw", filteredAddons).Print(out)
	},
}

// renderSearchResults renders the filtered list of addons from the registry
func renderSearchResults(addons []types.Addon, searchTerm string) string {
	var out bytes.Buffer

	t := table.NewWriter()
	t.SetOutputMirror(&out)
	styles.SetGlobalTableStyle(t, false)
	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Name: "Service",
		},
		{
			Name: "Description",
		},
	})
	sort.Slice(addons, func(i, j int) bool {
		return strings.Compare(strings.ToLower(addons[i].Title), strings.ToLower(addons[j].Title)) == -1
	})
	t.AppendHeader(table.Row{"Add-on", "Description"})

	for _, addon := range addons {
		d := addon.Description
		if addon.Type == "official" {
			d = d + "*"
		}
		t.AppendRow([]any{addon.Title, text.WrapSoft(d, 50)})
	}

	t.Render()

	return out.String() + fmt.Sprintf("%d add-ons found matching '%s'. Add-ons marked with '*' are officially maintained DDEV add-ons.", len(addons), searchTerm)
}

func init() {
	AddonCmd.AddCommand(AddonSearchCmd)
}
