package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	ddevgh "github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/styles"
	"github.com/ddev/ddev/pkg/util"
	"github.com/google/go-github/v52/github"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Get implements the ddev get command
var Get = &cobra.Command{
	Use:   "get <addonOrURL> [project]",
	Short: "Get/Download a 3rd party add-on (service, provider, etc.)",
	Long:  `Get/Download a 3rd party add-on (service, provider, etc.). This can be a GitHub repo, in which case the latest release will be used, or it can be a link to a .tar.gz in the correct format (like a particular release's .tar.gz) or it can be a local directory. Use 'ddev get --list' or 'ddev get --list --all' to see a list of available add-ons. Without --all it shows only official DDEV add-ons. To list installed add-ons, 'ddev get --installed', to remove an add-on 'ddev get --remove <add-on>'.`,
	Example: `ddev get ddev/ddev-redis
ddev get ddev/ddev-redis --version v1.0.4
ddev get https://github.com/ddev/ddev-drupal-solr/archive/refs/tags/v1.2.3.tar.gz
ddev get /path/to/package
ddev get /path/to/tarball.tar.gz
ddev get --list
ddev get --list --all
ddev get --installed
ddev get --remove someaddonname,
ddev get --remove someowner/ddev-someaddonname,
ddev get --remove ddev-someaddonname
`,
	Run: func(cmd *cobra.Command, args []string) {
		officialOnly := true
		verbose := false
		bash := util.FindBashPath()
		requestedVersion := ""

		if cmd.Flags().Changed("version") {
			requestedVersion = cmd.Flag("version").Value.String()
		}

		if cmd.Flags().Changed("verbose") {
			verbose = true
		}

		// Handle ddev get --list and ddev get --list --all
		// these do not require an app context
		if cmd.Flags().Changed("list") {
			if cmd.Flag("all").Changed {
				officialOnly = false
			}
			repos, err := ddevapp.ListAvailableAddons(officialOnly)
			if err != nil {
				util.Failed("Failed to list available add-ons: %v", err)
			}
			if len(repos) == 0 {
				util.Warning("No DDEV add-ons found with GitHub topic 'ddev-get'.")
				return
			}
			out := renderRepositoryList(repos)
			output.UserOut.WithField("raw", repos).Print(out)
			return
		}

		// Handle ddev get --installed
		if cmd.Flags().Changed("installed") {
			app, err := ddevapp.GetActiveApp("")
			if err != nil {
				util.Failed("Unable to find active project: %v", err)
			}

			ListInstalledAddons(app)
			return
		}

		// Handle ddev get --remove
		if cmd.Flags().Changed("remove") {
			app, err := ddevapp.GetActiveApp("")
			if err != nil {
				util.Failed("Unable to find active project: %v", err)
			}

			origDir, _ := os.Getwd()

			defer func() {
				err = os.Chdir(origDir)
				if err != nil {
					util.Failed("Unable to chdir to %v: %v", origDir, err)
				}
			}()

			err = os.Chdir(app.GetConfigPath(""))
			if err != nil {
				util.Failed("Unable to chdir to %v: %v", app.GetConfigPath(""), err)
			}

			app.DockerEnv()

			err = ddevapp.RemoveAddon(app, cmd.Flag("remove").Value.String(), nil, bash, verbose)
			if err != nil {
				util.Failed("Unable to remove add-on: %v", err)
			}
			return
		}

		if len(args) < 1 {
			util.Failed("You must specify an add-on to download")
		}
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
		downloadedRelease := ""
		switch {
		// If the provided sourceRepoArg is a directory, then we will use that as the source
		case fileutil.IsDirectory(sourceRepoArg):
			// Use the directory as the source
			extractedDir = sourceRepoArg
			argType = "directory"

		// If sourceRepoArg is a tarball on local filesystem, we can use that
		case fileutil.FileExists(sourceRepoArg) && (strings.HasSuffix(filepath.Base(sourceRepoArg), "tar.gz") || strings.HasSuffix(filepath.Base(sourceRepoArg), "tar") || strings.HasSuffix(filepath.Base(sourceRepoArg), "tgz")):
			// If the provided sourceRepoArg is a file, then we will use that as the source
			extractedDir, cleanup, err = archive.ExtractTarballWithCleanup(sourceRepoArg, true)
			if err != nil {
				util.Failed("Unable to extract %s: %v", sourceRepoArg, err)
			}
			argType = "tarball"
			defer cleanup()

		// If the provided sourceRepoArg is a GitHub sourceRepoArg, then we will use that as the source
		case len(parts) == 2: // github.com/owner/repo
			argType = "github"
			owner = parts[0]
			repo = parts[1]
			ctx := context.Background()

			client := ddevgh.GetGithubClient(ctx)
			releases, resp, err := client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{PerPage: 100})
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
			releaseItem := 0
			releaseFound := false
			if requestedVersion != "" {
				for i, release := range releases {
					if release.GetTagName() == requestedVersion {
						releaseItem = i
						releaseFound = true
						break
					}
				}
				if !releaseFound {
					util.Failed("No release found for %v with tag %v", repo, requestedVersion)
				}
			}
			tarballURL = releases[releaseItem].GetTarballURL()
			downloadedRelease = releases[releaseItem].GetTagName()
			util.Success("Installing %s/%s:%s", owner, repo, downloadedRelease)
			fallthrough

		// Otherwise, use the provided source as a URL to a tarball
		default:
			if tarballURL == "" {
				tarballURL = sourceRepoArg
				argType = "tarball"
			}
			extractedDir, cleanup, err = archive.DownloadAndExtractTarball(tarballURL, true)
			if err != nil {
				util.Failed("Unable to download %v: %v", sourceRepoArg, err)
			}
			defer cleanup()
		}

		// 20220811: Don't auto-start because it auto-creates the wrong database in some situations, leading to a
		// chicken-egg problem in getting database configured. See https://github.com/ddev/ddev-platformsh/issues/24
		// Automatically start, as we don't want to be taking actions with Mutagen off, for example.
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
		var s ddevapp.InstallDesc
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
				util.Warning("Unable to import yaml file %s: %v", fullpath, err)
			}
		}
		for k, v := range map[string]string{"DdevGlobalConfig": globalconfig.GetGlobalConfigPath(), "DdevProjectConfig": app.GetConfigPath("config.yaml")} {
			yamlMap[k], err = util.YamlFileToMap(v)
			if err != nil {
				util.Warning("Unable to read file %s", v)
			}
		}

		dict, err := util.YamlToDict(yamlMap)
		if err != nil {
			util.Failed("Unable to YamlToDict: %v", err)
		}
		// Check to see if any dependencies are missing
		if len(s.Dependencies) > 0 {
			// Read in full existing registered config
			m, err := ddevapp.GatherAllManifests(app)
			if err != nil {
				util.Failed("Unable to gather manifests: %v", err)
			}
			for _, dep := range s.Dependencies {
				if _, ok := m[dep]; !ok {
					util.Failed("The add-on '%s' declares a dependency on '%s'; Please ddev get %s first.", s.Name, dep, dep)
				}
			}
		}

		if s.DdevVersionConstraint != "" {
			err := ddevapp.CheckDdevVersionConstraint(s.DdevVersionConstraint, fmt.Sprintf("Unable to install the '%s' add-on", s.Name), "")
			if err != nil {
				util.Failed(err.Error())
			}
		}

		if len(s.PreInstallActions) > 0 {
			util.Success("\nExecuting pre-install actions:")
		}
		for i, action := range s.PreInstallActions {
			err = ddevapp.ProcessAddonAction(action, dict, bash, verbose)
			if err != nil {
				desc := ddevapp.GetAddonDdevDescription(action)
				if err != nil {
					if !verbose {
						util.Failed("Could not process pre-install action (%d) '%s'. For more detail use ddev get --verbose", i, desc)
					} else {
						util.Failed("Could not process pre-install action (%d) '%s'; error=%v\n action=%s", i, desc, err, action)
					}
				}
			}
		}

		if len(s.ProjectFiles) > 0 {
			util.Success("\nInstalling project-level components:")
		}

		projectFiles, err := fileutil.ExpandFilesAndDirectories(extractedDir, s.ProjectFiles)
		if err != nil {
			util.Failed("Unable to expand files and directories: %v", err)
		}
		for _, file := range projectFiles {
			src := filepath.Join(extractedDir, file)
			dest := app.GetConfigPath(file)
			if err = fileutil.CheckSignatureOrNoFile(dest, nodeps.DdevFileSignature); err == nil {
				err = copy.Copy(src, dest)
				if err != nil {
					util.Failed("Unable to copy %v to %v: %v", src, dest, err)
				}
				util.Success("%c %s", '\U0001F44D', file)
			} else {
				util.Warning("NOT overwriting %s. The #ddev-generated signature was not found in the file, so it will not be overwritten. You can remove the file and use DDEV get again if you want it to be replaced: %v", dest, err)
			}
		}
		globalDotDdev := filepath.Join(globalconfig.GetGlobalDdevDir())
		if len(s.GlobalFiles) > 0 {
			util.Success("\nInstalling global components:")
		}

		globalFiles, err := fileutil.ExpandFilesAndDirectories(extractedDir, s.GlobalFiles)
		if err != nil {
			util.Failed("Unable to expand global files and directories: %v", err)
		}
		for _, file := range globalFiles {
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
				util.Warning("NOT overwriting %s. The #ddev-generated signature was not found in the file, so it will not be overwritten. You can remove the file and use DDEV get again if you want it to be replaced: %v", dest, err)
			}
		}
		origDir, _ := os.Getwd()

		defer func() {
			err = os.Chdir(origDir)
			if err != nil {
				util.Failed("Unable to chdir to %v: %v", origDir, err)
			}
		}()

		err = os.Chdir(app.GetConfigPath(""))
		if err != nil {
			util.Failed("Unable to chdir to %v: %v", app.GetConfigPath(""), err)
		}

		if len(s.PostInstallActions) > 0 {
			util.Success("\nExecuting post-install actions:")
		}
		for i, action := range s.PostInstallActions {
			err = ddevapp.ProcessAddonAction(action, dict, bash, verbose)
			desc := ddevapp.GetAddonDdevDescription(action)
			if err != nil {
				if !verbose {
					util.Failed("Could not process post-install action (%d) '%s'", i, desc)
				} else {
					util.Failed("Could not process post-install action (%d) '%s': %v", i, desc, err)
				}
			}
		}

		repository := ""
		switch argType {
		case "github":
			repository = fmt.Sprintf("%s/%s", owner, repo)
		case "directory":
			fallthrough
		case "tarball":
			repository = sourceRepoArg
		}
		manifest, err := createManifestFile(app, s.Name, repository, downloadedRelease, s)
		if err != nil {
			util.Failed("Unable to create manifest file: %v", err)
		}

		util.Success("\nInstalled DDEV add-on %s, use `ddev restart` to enable.", sourceRepoArg)
		if argType == "github" {
			util.Success("Please read instructions for this add-on at the source repo at\nhttps://github.com/%v/%v\nPlease file issues and create pull requests there to improve it.", owner, repo)
		}
		output.UserOut.WithField("raw", manifest).Printf("Installed %s:%s from %s", manifest.Name, manifest.Version, manifest.Repository)
	},
}

// createManifestFile creates a manifest file for the addon
func createManifestFile(app *ddevapp.DdevApp, addonName string, repository string, downloadedRelease string, desc ddevapp.InstallDesc) (ddevapp.AddonManifest, error) {
	// Create a manifest file
	manifest := ddevapp.AddonManifest{
		Name:           addonName,
		Repository:     repository,
		Version:        downloadedRelease,
		Dependencies:   desc.Dependencies,
		InstallDate:    time.Now().Format(time.RFC3339),
		ProjectFiles:   desc.ProjectFiles,
		GlobalFiles:    desc.GlobalFiles,
		RemovalActions: desc.RemovalActions,
	}
	manifestFile := app.GetConfigPath(fmt.Sprintf("%s/%s/manifest.yaml", ddevapp.AddonMetadataDir, addonName))
	if fileutil.FileExists(manifestFile) {
		util.Warning("Overwriting existing manifest file %s", manifestFile)
	}
	manifestData, err := yaml.Marshal(manifest)
	if err != nil {
		util.Failed("Error marshaling manifest data: %v", err)
	}
	err = os.MkdirAll(filepath.Dir(manifestFile), 0755)
	if err != nil {
		util.Failed("Error creating manifest directory: %v", err)
	}
	if err = fileutil.TemplateStringToFile(string(manifestData), nil, manifestFile); err != nil {
		util.Failed("Error writing manifest file: %v", err)
	}
	return manifest, nil
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
		output.UserOut.Println("No registered add-ons were found. Add-ons installed before DDEV v1.22.0 will not be listed.\nUpdate them with `ddev get` so they'll be shown.")
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
	Get.Flags().Bool("list", true, `List available add-ons for 'ddev get'`)
	Get.Flags().Bool("all", true, `List unofficial add-ons for 'ddev get' in addition to the official ones`)
	Get.Flags().Bool("installed", true, `Show installed ddev-get add-ons`)
	Get.Flags().String("remove", "", `Remove a ddev-get add-on`)
	Get.Flags().String("version", "", `Specify a particular version of add-on to install`)
	Get.Flags().BoolP("verbose", "v", false, "Extended/verbose output for ddev get")
	RootCmd.AddCommand(Get)
}
