package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
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

const addonMetadataDir = "addon-metadata"

// Format of install.yaml
type installDesc struct {
	// Name must be unique in a project; it will overwrite any existing add-on with the same name.
	Name               string            `yaml:"name"`
	ProjectFiles       []string          `yaml:"project_files"`
	GlobalFiles        []string          `yaml:"global_files,omitempty"`
	Dependencies       []string          `yaml:"dependencies,omitempty"`
	PreInstallActions  []string          `yaml:"pre_install_actions,omitempty"`
	PostInstallActions []string          `yaml:"post_install_actions,omitempty"`
	RemovalActions     []string          `yaml:"removal_actions,omitempty"`
	YamlReadFiles      map[string]string `yaml:"yaml_read_files"`
}

// format of the add-on manifest file
type addonManifest struct {
	Name           string   `yaml:"name"`
	Repository     string   `yaml:"repository"`
	Version        string   `yaml:"version"`
	Dependencies   []string `yaml:"dependencies,omitempty"`
	InstallDate    string   `yaml:"install_date"`
	ProjectFiles   []string `yaml:"project_files"`
	GlobalFiles    []string `yaml:"global_files"`
	RemovalActions []string `yaml:"removal_actions"`
}

// Get implements the ddev get command
var Get = &cobra.Command{
	Use:   "get <addonOrURL> [project]",
	Short: "Get/Download a 3rd party add-on (service, provider, etc.)",
	Long:  `Get/Download a 3rd party add-on (service, provider, etc.). This can be a github repo, in which case the latest release will be used, or it can be a link to a .tar.gz in the correct format (like a particular release's .tar.gz) or it can be a local directory. Use 'ddev get --list' or 'ddev get --list --all' to see a list of available add-ons. Without --all it shows only official ddev add-ons. To list installed add-ons, 'ddev get --installed', to remove an add-on 'ddev get --remove <add-on>'.`,
	Example: `ddev get ddev/ddev-redis
ddev get ddev/ddev-redis --version v1.0.4
ddev get https://github.com/ddev/ddev-drupal9-solr/archive/refs/tags/v0.0.5.tar.gz
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

		// handle ddev get --list and ddev get --list --all
		// these do not require an app context
		if cmd.Flags().Changed("list") {
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

		// handle ddev get --installed
		if cmd.Flags().Changed("installed") {
			app, err := ddevapp.GetActiveApp("")
			if err != nil {
				util.Failed("unable to find active project: %v", err)
			}

			listInstalledAddons(app)
			return
		}

		// handle ddev get --remove
		if cmd.Flags().Changed("remove") {
			app, err := ddevapp.GetActiveApp("")
			if err != nil {
				util.Failed("unable to find active project: %v", err)
			}
			app.DockerEnv()

			err = removeAddon(app, cmd.Flag("remove").Value.String(), nil, bash, verbose)
			if err != nil {
				util.Failed("unable to remove add-on: %v", err)
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
				util.Warning("unable to import yaml file %s: %v", fullpath, err)
			}
		}
		for k, v := range map[string]string{"DdevGlobalConfig": globalconfig.GetGlobalConfigPath(), "DdevProjectConfig": app.GetConfigPath("config.yaml")} {
			yamlMap[k], err = util.YamlFileToMap(v)
			if err != nil {
				util.Warning("unable to read file %s", v)
			}
		}

		dict, err := util.YamlToDict(yamlMap)
		if err != nil {
			util.Failed("Unable to YamlToDict: %v", err)
		}
		// Check to see if any dependencies are missing
		if len(s.Dependencies) > 0 {
			// Read in full existing registered config
			m, err := gatherAllManifests(app)
			if err != nil {
				util.Failed("Unable to gather manifests: %v", err)
			}
			for _, dep := range s.Dependencies {
				if _, ok := m[dep]; !ok {
					util.Failed("The add-on '%s' declares a dependency on '%s'; Please ddev get %s first.", s.Name, dep, dep)
				}
			}
		}
		if len(s.PreInstallActions) > 0 {
			util.Success("\nExecuting pre-install actions:")
		}
		for i, action := range s.PreInstallActions {
			err = processAction(action, dict, bash, verbose)
			if err != nil {
				desc := getDdevDescription(action)
				if err != nil {
					if !verbose {
						util.Failed("could not process pre-install action (%d) '%s'. For more detail use ddev get --verbose", i, desc)
					} else {
						util.Failed("could not process pre-install action (%d) '%s'; error=%v\n action=%s", i, desc, err, action)
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
				util.Warning("NOT overwriting %s. The #ddev-generated signature was not found in the file, so it will not be overwritten. You can just remove the file and use ddev get again if you want it to be replaced: %v", dest, err)
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
				util.Warning("NOT overwriting %s. The #ddev-generated signature was not found in the file, so it will not be overwritten. You can just remove the file and use ddev get again if you want it to be replaced: %v", dest, err)
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
			err = processAction(action, dict, bash, verbose)
			desc := getDdevDescription(action)
			if err != nil {
				if !verbose {
					util.Failed("could not process post-install action (%d) '%s'", i, desc)
				} else {
					util.Failed("could not process post-install action (%d) '%s': %v", i, desc, err)
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
			util.Success("Please read instructions for this addon at the source repo at\nhttps://github.com/%v/%v\nPlease file issues and create pull requests there to improve it.", owner, repo)
		}
		output.UserOut.WithField("raw", manifest).Printf("Installed %s:%s from %s", manifest.Name, manifest.Version, manifest.Repository)
	},
}

// createManifestFile creates a manifest file for the addon
func createManifestFile(app *ddevapp.DdevApp, addonName string, repository string, downloadedRelease string, desc installDesc) (addonManifest, error) {
	// Create a manifest file
	manifest := addonManifest{
		Name:           addonName,
		Repository:     repository,
		Version:        downloadedRelease,
		Dependencies:   desc.Dependencies,
		InstallDate:    time.Now().Format(time.RFC3339),
		ProjectFiles:   desc.ProjectFiles,
		GlobalFiles:    desc.GlobalFiles,
		RemovalActions: desc.RemovalActions,
	}
	manifestFile := app.GetConfigPath(fmt.Sprintf("%s/%s/manifest.yaml", addonMetadataDir, addonName))
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

// listInstalledAddons() show the add-ons that have a manifest file
func listInstalledAddons(app *ddevapp.DdevApp) {
	metadataDir := app.GetConfigPath(addonMetadataDir)
	err := os.MkdirAll(metadataDir, 0755)
	if err != nil {
		util.Failed("Error creating metadata directory: %v", err)
	}
	// Read the contents of the .ddev/addon-metadata directory (directories)
	dirs, err := os.ReadDir(metadataDir)
	if err != nil {
		util.Failed("Error reading metadata directory: %v", err)
	}
	manifests := []addonManifest{}

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
	for _, d := range dirs {
		// Check if the file is a directory
		if d.IsDir() {
			// Read the contents of the manifest file
			manifestFile := filepath.Join(metadataDir, d.Name(), "manifest.yaml")
			manifestBytes, err := os.ReadFile(manifestFile)
			if err != nil {
				util.Warning("No manifest file found at %s: %v", manifestFile, err)
				continue
			}

			// Parse the manifest file
			var manifest addonManifest
			err = yaml.Unmarshal(manifestBytes, &manifest)
			if err != nil {
				util.Failed("Unable to parse manifest file: %v", err)
			}
			manifests = append(manifests, manifest)
			t.AppendRow(table.Row{manifest.Name, manifest.Version, manifest.Repository, manifest.InstallDate})
		}
	}
	if t.Length() == 0 {
		output.UserOut.Println("No registered add-ons were found. Add-ons installed before DDEV v1.22.0 will not be listed.\nUpdate them with `ddev get` so they'll be shown.")
		return
	}
	t.Render()
	output.UserOut.WithField("raw", manifests).Println(out.String())
}

// processAction takes a stanza from yaml exec section and executes it.
func processAction(action string, dict map[string]interface{}, bashPath string, verbose bool) error {
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
	if verbose {
		action = "set -x; " + action
	}
	out, err := exec.RunHostCommand(bashPath, "-c", action)
	if len(out) > 0 {
		util.Warning(out)
	}
	if err != nil {
		util.Warning("%c %s", '\U0001F44E', desc)
		return fmt.Errorf("Unable to run action %v: %v, output=%s", action, err, out)
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

// listAvailable lists the services that are listed on github
func listAvailable(officialOnly bool) ([]*github.Repository, error) {
	client := ddevgh.GetGithubClient(context.Background())
	q := "topic:ddev-get fork:true"
	if officialOnly {
		q = q + " org:" + globalconfig.DdevGithubOrg
	}

	opts := &github.SearchOptions{Sort: "updated", Order: "desc", ListOptions: github.ListOptions{PerPage: 200}}
	var allRepos []*github.Repository
	for {

		repos, resp, err := client.Search.Repositories(context.Background(), q, opts)
		if err != nil {
			msg := fmt.Sprintf("Unable to get list of available services: %v", err)
			if resp != nil {
				msg = msg + fmt.Sprintf(" rateinfo=%v", resp.Rate)
			}
			return nil, fmt.Errorf(msg)
		}
		allRepos = append(allRepos, repos.Repositories...)
		if resp.NextPage == 0 {
			break
		}

		// Set the next page number for the next request
		opts.ListOptions.Page = resp.NextPage
	}
	out := ""
	for _, r := range allRepos {
		out = out + fmt.Sprintf("%s: %s\n", r.GetFullName(), r.GetDescription())
	}
	if len(allRepos) == 0 {
		return nil, fmt.Errorf("No add-ons found")
	}
	return allRepos, nil
}

// removeAddon removes an addon, taking care to respect #ddev-generated
// addonName can be the "Name", or the full "Repository" like ddev/ddev-redis, or
// the final par of the repository name like ddev-redis
func removeAddon(app *ddevapp.DdevApp, addonName string, dict map[string]interface{}, bash string, verbose bool) error {
	if addonName == "" {
		return fmt.Errorf("No add-on name specified for removal")
	}

	manifests, err := gatherAllManifests(app)
	if err != nil {
		util.Failed("Unable to gather all manifests: %v", err)
	}

	var manifestData addonManifest
	var ok bool

	if manifestData, ok = manifests[addonName]; !ok {
		util.Failed("The add-on '%s' does not seem to have a manifest file; please upgrade it.\nUse `ddev get --installed to see installed add-ons.\nIf yours is not there it may have been installed before DDEV v1.22.0.\nUse 'ddev get' to update it.", addonName)
	}

	// Execute any removal actions
	for i, action := range manifestData.RemovalActions {
		err = processAction(action, dict, bash, verbose)
		desc := getDdevDescription(action)
		if err != nil {
			util.Warning("could not process removal action (%d) '%s': %v", i, desc, err)
		}
	}

	// Remove any project files
	for _, f := range manifestData.ProjectFiles {
		p := app.GetConfigPath(f)
		err = fileutil.CheckSignatureOrNoFile(p, nodeps.DdevFileSignature)
		if err == nil {
			_ = os.RemoveAll(p)
		} else {
			util.Warning("Unwilling to remove '%s' because it does not have #ddev-generated in it: %v; you can manually delete it if it is safe to delete.", p, err)
		}
	}

	// Remove any global files
	globalDotDdev := filepath.Join(globalconfig.GetGlobalDdevDir())
	for _, f := range manifestData.GlobalFiles {
		p := filepath.Join(globalDotDdev, f)
		err = fileutil.CheckSignatureOrNoFile(p, nodeps.DdevFileSignature)
		if err == nil {
			_ = os.RemoveAll(p)
		} else {
			util.Warning("Unwilling to remove '%s' because it does not have #ddev-generated in it: %v; you can manually delete it if it is safe to delete.", p, err)
		}
	}
	if len(manifestData.Dependencies) > 0 {
		for _, dep := range manifestData.Dependencies {
			if m, ok := manifests[dep]; ok {
				util.Warning("The add-on you're removing ('%s') declares a dependency on '%s', which is not being removed. You may want to remove it manually if it is no longer needed.", addonName, m.Name)
			}
		}
	}

	err = os.RemoveAll(app.GetConfigPath(filepath.Join(addonMetadataDir, manifestData.Name)))
	if err != nil {
		return fmt.Errorf("Error removing addon metadata directory %s: %v", manifestData.Name, err)
	}
	util.Success("Removed add-on %s", addonName)
	return nil
}

// gatherAllManifests searches for all addon manifests and presents the result
// as a map of various names to manifest data
func gatherAllManifests(app *ddevapp.DdevApp) (map[string]addonManifest, error) {
	metadataDir := app.GetConfigPath(addonMetadataDir)
	allManifests := make(map[string]addonManifest)

	dirs, err := fileutil.ListFilesInDirFullPath(metadataDir)
	if err != nil {
		return nil, err
	}
	for _, d := range dirs {
		if !fileutil.IsDirectory(d) {
			continue
		}

		mPath := filepath.Join(d, "manifest.yaml")
		manifestString, err := fileutil.ReadFileIntoString(mPath)
		if err != nil {
			return nil, err
		}
		var manifestData = &addonManifest{}
		err = yaml.Unmarshal([]byte(manifestString), manifestData)
		if err != nil {
			return nil, fmt.Errorf("Error unmarshalling manifest data: %v", err)
		}
		allManifests[manifestData.Name] = *manifestData
		allManifests[manifestData.Repository] = *manifestData

		pathParts := strings.Split(manifestData.Repository, "/")
		if len(pathParts) > 1 {
			shortRepo := pathParts[len(pathParts)-1]
			allManifests[shortRepo] = *manifestData
		}
	}
	return allManifests, nil
}

func init() {
	Get.Flags().Bool("list", true, fmt.Sprintf(`List available add-ons for 'ddev get'`))
	Get.Flags().Bool("all", true, fmt.Sprintf(`List unofficial add-ons for 'ddev get' in addition to the official ones`))
	Get.Flags().Bool("installed", true, fmt.Sprintf(`Show installed ddev-get add-ons`))
	Get.Flags().String("remove", "", fmt.Sprintf(`Remove a ddev-get add-on`))
	Get.Flags().String("version", "", fmt.Sprintf(`Specify a particular version of add-on to install`))
	Get.Flags().BoolP("verbose", "v", false, "Extended/verbose output for ddev get")
	RootCmd.AddCommand(Get)
}
