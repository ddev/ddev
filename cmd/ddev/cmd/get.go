package cmd

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Masterminds/sprig/v3"
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
	"text/template"
)

type installDesc struct {
	Name               string            `yaml:"name"`
	ProjectFiles       []string          `yaml:"project_files"`
	GlobalFiles        []string          `yaml:"global_files,omitempty"`
	PreInstallActions  []string          `yaml:"pre_install_actions,omitempty"`
	PostInstallActions []string          `yaml:"post_install_actions,omitempty"`
	YamlReadFiles      map[string]string `yaml:"yaml_read_files"`
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
		err = os.Chdir(app.AppRoot)
		if err != nil {
			util.Failed("Unable to change directory to project root %s: %v", app.AppRoot, err)
		}
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

		// 20220811: Don't auto-start because it auto-creates the wrong database in some situations, leading to a
		// chicken-egg problem in getting database configured. See https://github.com/platformsh/ddev-platformsh/issues/24
		// Automatically start, as we don't want to be taking actions with mutagen off, for example.
		//if status, _ := app.SiteStatus(); status != ddevapp.SiteRunning {
		//	err = app.Start()
		//	if err != nil {
		//		util.Failed("Failed to start app %s to ddev-get: %v", app.Name, err)
		//	}
		//}

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

		yamlMap := make(map[string]interface{})
		for name, f := range s.YamlReadFiles {
			f := os.ExpandEnv(string(f))
			fullpath := filepath.Join(app.GetAppRoot(), f)

			yamlMap[name], err = util.YamlFileToMap(fullpath)
			if err != nil {
				util.Failed("unable to import yaml file %s: %v", fullpath, err)
			}
		}
		for k, v := range map[string]string{"DdevGlobalConfig": globalconfig.GetGlobalConfigPath(), "DdevProjectConfig": app.GetConfigPath("config.yaml")} {
			yamlMap[k], err = util.YamlFileToMap(v)
			if err != nil {
				util.Failed("unable to read file %s", v)
			}
		}

		dict, err := util.YamlToDict(yamlMap)
		if err != nil {
			util.Failed("Unable to YamlToDict: %v", err)
		}
		if len(s.PreInstallActions) > 0 {
			util.Success("\nExecuting pre-install actions:")
		}
		for _, action := range s.PreInstallActions {
			err = processAction(action, dict, bash)
			if err != nil {
				util.Failed("could not process pre-install action '%s': %v", action, err)
			}
		}

		if len(s.ProjectFiles) > 0 {
			util.Success("\nInstalling project-level components:")
		}
		for _, file := range s.ProjectFiles {
			file := os.ExpandEnv(file)
			src := filepath.Join(extractedDir, file)
			dest := app.GetConfigPath(file)
			if err = fileutil.CheckSignatureOrNoFile(dest, nodeps.DdevFileSignature); err == nil {
				err = copy.Copy(src, dest)
				if err != nil {
					util.Failed("Unable to copy %v to %v: %v", src, dest, err)
				}
				util.Success("%c %s", '\U0001F44D', file)
			} else {
				util.Warning("NOT overwriting file/directory %s. The #ddev-generated signature was not found in the file, so it will not be overwritten. You can just remove the file and use ddev get again if you want it to be replaced: %v", dest, err)
			}
		}
		globalDotDdev := filepath.Join(globalconfig.GetGlobalDdevDir())
		if len(s.GlobalFiles) > 0 {
			util.Success("\nInstalling global components:")
		}
		for _, file := range s.GlobalFiles {
			file := os.ExpandEnv(file)
			src := filepath.Join(extractedDir, file)
			dest := filepath.Join(globalDotDdev, file)

			// If the file existed and had #ddev-generated OR if it did not exist, copy it in.
			if err = fileutil.CheckSignatureOrNoFile(dest, nodeps.DdevFileSignature); err == nil {
				err = copy.Copy(src, dest)
				if err != nil {
					util.Failed("Unable to copy %v to %v: %v", src, dest, err)
				}
				util.Success("%c %s", '\U0001F44D', file)
			} else {
				util.Warning("NOT overwriting file/directory %s. The #ddev-generated signature was not found in the file, so it will not be overwritten. You can just remove the file and use ddev get again if you want it to be replaced: %v", dest, err)
			}
		}
		origDir, _ := os.Getwd()

		//nolint: errcheck
		defer os.Chdir(origDir)
		err = os.Chdir(app.GetConfigPath(""))
		if err != nil {
			util.Failed("Unable to chdir to %v: %v", app.GetConfigPath(""), err)
		}

		if len(s.PostInstallActions) > 0 {
			util.Success("\nExecuting post-install actions:")
		}
		for _, action := range s.PostInstallActions {
			err = processAction(action, dict, bash)
			if err != nil {
				util.Failed("could not process post-install action '%s': %v", action, err)
			}
		}

		util.Success("\nInstalled DDEV add-on %s, use `ddev restart` to enable.", sourceRepoArg)
		if argType == "github" {
			util.Success("Please read instructions for this addon at the source repo at\nhttps://github.com/%v/%v\nPlease file issues and create pull requests there to improve it.", owner, repo)
		}

	},
}

// processAction takes a stanza from yaml exec section and executes it.
func processAction(action string, dict map[string]interface{}, bashPath string) error {
	action = "set -eu -o pipefail\n" + action
	t, err := template.New("processAction").Funcs(sprig.TxtFuncMap()).Parse(action)
	if err != nil {
		return fmt.Errorf("could not parse action '%s': %v", action, err)
	}

	var doc bytes.Buffer
	err = t.Execute(&doc, dict)
	if err != nil {
		return fmt.Errorf("could not parse/execute action '%s': %v", action, err)
	}
	action = doc.String()

	desc := getDdevDescription(action)
	out, err := exec.RunHostCommand(bashPath, "-c", action)
	if err != nil {
		util.Warning("%c %s", '\U0001F44E', desc)
		return fmt.Errorf("Unable to run action %v: %v, output=%s", action, err, out)
	}
	if len(out) > 0 {
		output.UserOut.Print(out)
	}
	if desc != "" {
		util.Success("%c %s", '\U0001F44D', desc)
	}
	return nil
}

// getDdevDescription returns what follows #ddev-description: in any line in action
func getDdevDescription(action string) string {
	descLines := nodeps.GrepStringInBuffer(action, `[\r\n]+#ddev-description:.*[\r\n]+`)
	if len(descLines) > 0 {
		d := strings.Split(descLines[0], ":")
		if len(d) > 1 {
			return strings.Trim(d[1], "\r\n\t")
		}
	}
	return ""
}

func renderRepositoryList(repos []github.Repository) string {
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
		msg := fmt.Sprintf("Unable to get list of available services: %v", err)
		if resp != nil {
			msg = msg + fmt.Sprintf(" rateinfo=%v", resp.Rate)
		}
		return nil, fmt.Errorf(msg)
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
