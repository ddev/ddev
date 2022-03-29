package cmd

import (
	"bytes"
	"context"
	"fmt"
	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/styles"
	"github.com/drud/ddev/pkg/util"
	"github.com/google/go-github/github"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type installDesc struct {
	Name               string   `yaml:"name"`
	ProjectFiles       []string `yaml:"project_files"`
	GlobalFiles        []string `yaml:"global_files,omitempty"`
	PreInstallActions  []string `yaml:"pre_install_actions,omitempty"`
	PostInstallActions []string `yaml:"post_install_actions,omitempty"`
}

// Get implements the ddev get command
var Get = &cobra.Command{
	Use:   "get <addonOrURL> [project]",
	Short: "Get/Download a 3rd party add-on (service, provider, etc.)",
	Long:  `Get/Download a 3rd party add-on (service, provider, etc.). This can be a github repo, in which case the latest release will be used, or it can be a link to a .tar.gz in the correct format (like a particular release's .tar.gz) or it can be a local directory. Use 'ddev get --list' or 'ddev get --list --all' to see a list of available add-ons. Without --all it shows only official ddev add-ons.`,
	Example: `ddev get drud/ddev-drupal9-solr
ddev get https://github.com/drud/ddev-drupal9-solr/archive/refs/tags/v0.0.5.tar.gz
ddev get /path/to/package
ddev get /path/to/tarball.tar.gz
ddev get --list
ddev get --list --all
`,
	Run: func(cmd *cobra.Command, args []string) {
		officialOnly := true
		if cmd.Flag("list").Changed {
			if cmd.Flag("all").Changed {
				officialOnly = false
			}
			repos, err := listAvailable(officialOnly)
			if err != nil {
				util.Failed("Failed to list available add-ons: %v", err)
			}
			if len(repos) == 0 {
				util.Warning("No ddev add-ons found with GitHub topic 'ddev-get'.")
				return
			}
			out := renderRepositoryList(repos)
			output.UserOut.WithField("raw", repos).Print(out)
			return
		}

		if len(args) < 1 {
			util.Failed("You must specify an add-on to download")
		}
		bash := util.FindBashPath()
		apps, err := getRequestedProjects(args[1:], false)
		if err != nil {
			util.Failed("Unable to get project(s) %v: %v", args, err)
		}
		if len(apps) == 0 {
			util.Failed("No project(s) found")
		}
		app := apps[0]
		app.DockerEnv()
		sourceRepoArg := args[0]
		extractedDir := ""
		parts := strings.Split(sourceRepoArg, "/")
		tarballURL := ""
		var cleanup func()
		argType := ""
		owner := ""
		repo := ""
		switch {
		// If the provided sourceRepoArg is a directory, then we will use that as the source
		case fileutil.IsDirectory(sourceRepoArg):
			// Use the directory as the source
			extractedDir = sourceRepoArg
			argType = "directory"

		// if sourceRepoArg is a tarball on local filesystem, we can use that
		case fileutil.FileExists(sourceRepoArg) && (strings.HasSuffix(filepath.Base(sourceRepoArg), "tar.gz") || strings.HasSuffix(filepath.Base(sourceRepoArg), "tar") || strings.HasSuffix(filepath.Base(sourceRepoArg), "tgz")):
			// If the provided sourceRepoArg is a file, then we will use that as the source
			extractedDir, cleanup, err = archive.ExtractTarballWithCleanup(sourceRepoArg, true)
			if err != nil {
				util.Failed("Unable to extract %s: %v", sourceRepoArg, err)
			}
			argType = "tarball"
			defer cleanup()

		// If the provided sourceRepoArg is a github sourceRepoArg, then we will use that as the source
		case len(parts) == 2: // github.com/owner/sourceRepoArg
			owner = parts[0]
			repo = parts[1]
			ctx := context.Background()

			client := getGithubClient(ctx)
			releases, resp, err := client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{})
			if err != nil {
				var rate github.Rate
				if resp != nil {
					rate = resp.Rate
				}
				util.Failed("Unable to get releases for %v: %v\nresp.Rate=%v", repo, err, rate)
			}
			if len(releases) == 0 {
				util.Failed("No releases found for %v", repo)
			}
			tarballURL = releases[0].GetTarballURL()
			argType = "github"
			fallthrough

		// Otherwise, use the provided source as a URL to a tarball
		default:
			if tarballURL == "" {
				tarballURL = sourceRepoArg
			}
			extractedDir, cleanup, err = archive.DownloadAndExtractTarball(tarballURL, true)
			if err != nil {
				util.Failed("Unable to download %v: %v", sourceRepoArg, err)
			}
			defer cleanup()
		}
		yamlFile := filepath.Join(extractedDir, "install.yaml")
		yamlContent, err := fileutil.ReadFileIntoString(yamlFile)
		if err != nil {
			util.Failed("Unable to read %v: %v", yamlFile, err)
		}
		var s installDesc
		err = yaml.Unmarshal([]byte(yamlContent), &s)
		if err != nil {
			util.Failed("Unable to parse %v: %v", yamlFile, err)
		}

		for _, action := range s.PreInstallActions {
			out, err := exec.RunHostCommand(bash, "-c", action)
			if err != nil {
				util.Failed("Unable to run action %v: %v, output=%s", action, err, out)
			}
			output.UserOut.Printf("Executed pre-install action %v, output=%s.", action, out)
		}

		for _, file := range s.ProjectFiles {
			src := filepath.Join(extractedDir, file)
			dest := app.GetConfigPath(file)

			err = copy.Copy(src, dest)
			if err != nil {
				util.Failed("Unable to copy %v to %v: %v", src, dest, err)
			}
			output.UserOut.Printf("Installed file %s", dest)
		}
		globalDotDdev := filepath.Join(globalconfig.GetGlobalDdevDir())
		for _, file := range s.GlobalFiles {
			src := filepath.Join(extractedDir, file)
			dest := filepath.Join(globalDotDdev, file)
			err = copy.Copy(src, dest)
			if err != nil {
				util.Failed("Unable to copy %v to %v: %v", src, dest, err)
			}
			output.UserOut.Printf("Installed file %s", dest)
		}
		origDir, _ := os.Getwd()

		//nolint: errcheck
		defer os.Chdir(origDir)
		err = os.Chdir(app.GetConfigPath(""))
		if err != nil {
			util.Failed("Unable to chdir to %v: %v", app.GetConfigPath(""), err)
		}

		for _, action := range s.PostInstallActions {
			out, err := exec.RunHostCommand(bash, "-c", action)
			if err != nil {
				util.Failed("Unable to run action %v: %v, output=%s", action, err, out)
			}
			output.UserOut.Printf("Executed post-install action %v.", action)
		}
		util.Success("Downloaded add-on %s, use `ddev restart` to enable.", sourceRepoArg)
		if argType == "github" {
			util.Success("Please read instructions for this addon at the source repo at\nhttps://github.com/%v/%v\nPlease file issues and create pull requests there to improve it.", owner, repo)
		}
	},
}

func renderRepositoryList(repos []github.Repository) string {
	var out bytes.Buffer

	t := table.NewWriter()
	t.SetOutputMirror(&out)
	styles.SetGlobalTableStyle(t)
	tWidth, _ := nodeps.GetTerminalWidthHeight()
	if !globalconfig.DdevGlobalConfig.SimpleFormatting {
		t.SetAllowedRowLength(tWidth)
	}
	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Name: "Service",
		},
		{
			Name: "Description",
		},
	})
	sort.Slice(repos, func(i, j int) bool {
		if repos[i].GetOwner().GetLogin() == "drud" {
			return true
		}
		return false
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

	return out.String() + "Add-ons marked with '*' are official, maintained DDEV add-ons."
}

func init() {
	Get.Flags().Bool("list", true, fmt.Sprintf(`List available add-ons for 'ddev get'`))
	Get.Flags().Bool("all", true, fmt.Sprintf(`List unofficial add-ons for 'ddev get' in addition to the official ones`))
	RootCmd.AddCommand(Get)
}

// getGithubClient creates the required github client
func getGithubClient(ctx context.Context) *github.Client {
	client := github.NewClient(nil)

	// Use authenticated client for higher rate limit, normally only needed for tests
	githubToken := os.Getenv("DDEV_GITHUB_TOKEN")
	if githubToken != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: githubToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	}
	return client
}

// listAvailable lists the services that are listed on github
func listAvailable(officialOnly bool) ([]github.Repository, error) {
	client := getGithubClient(context.Background())
	q := "topic:ddev-get fork:true"
	if officialOnly {
		q = q + " org:" + globalconfig.DdevGithubOrg
	}

	repos, resp, err := client.Search.Repositories(context.Background(), q, nil)
	if err != nil {
		return nil, fmt.Errorf("Unable to get list of available services: %v\nresp.Rate=%v", err, resp.Rate)
	}
	out := ""
	for _, r := range repos.Repositories {
		out = out + fmt.Sprintf("%s: %s\n", r.GetFullName(), r.GetDescription())
	}
	if len(repos.Repositories) == 0 {
		return nil, fmt.Errorf("No add-ons found")
	}
	return repos.Repositories, err
}
