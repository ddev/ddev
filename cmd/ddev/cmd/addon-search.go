package cmd

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/globalconfig"
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
ddev add-on search redis commander
ddev add-on search "redis commander"`,
	Run: func(cmd *cobra.Command, args []string) {
		searchTerms := strings.Join(args, " ")

		// Get available add-ons using the same source as list command
		repos, err := ddevapp.ListAvailableAddons()
		if err != nil {
			util.Failed("Failed to search available add-ons: %v", err)
		}

		// Filter repositories based on search keywords
		var filteredRepos []*github.Repository

		// Extract all keywords from all arguments
		var keywords []string
		for _, arg := range args {
			// Split each argument by spaces to handle both quoted and unquoted cases
			words := strings.Fields(strings.ToLower(arg))
			keywords = append(keywords, words...)
		}

		for _, repo := range repos {
			repoName := strings.ToLower(repo.GetFullName())
			repoDesc := strings.ToLower(repo.GetDescription())
			searchText := repoName + " " + repoDesc

			// Check if all keywords appear in name or description
			matches := true
			for _, keyword := range keywords {
				if !strings.Contains(searchText, keyword) {
					matches = false
					break
				}
			}

			if matches {
				filteredRepos = append(filteredRepos, repo)
			}
		}

		if len(filteredRepos) == 0 {
			output.UserOut.Printf("No add-ons found matching '%s'\n", searchTerms)
			return
		}

		out := renderSearchResults(filteredRepos, searchTerms)
		output.UserOut.WithField("raw", filteredRepos).Print(out)
	},
}

// renderSearchResults renders the filtered list of repositories
func renderSearchResults(repos []*github.Repository, searchTerm string) string {
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
	sort.Slice(repos, func(i, j int) bool {
		return strings.Compare(strings.ToLower(repos[i].GetFullName()), strings.ToLower(repos[j].GetFullName())) == -1
	})
	t.AppendHeader(table.Row{"Add-on", "Description"})

	for _, repo := range repos {
		d := repo.GetDescription()
		if repo.GetOwner().GetLogin() == globalconfig.DdevGithubOrg {
			d = d + "*"
		}
		t.AppendRow([]any{repo.GetFullName(), text.WrapSoft(d, 50)})
	}

	t.Render()

	return out.String() + fmt.Sprintf("%d repositories found matching '%s'. Add-ons marked with '*' are officially maintained DDEV add-ons.", len(repos), searchTerm)
}

func init() {
	AddonCmd.AddCommand(AddonSearchCmd)
}
