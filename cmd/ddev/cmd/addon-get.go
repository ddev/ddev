package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	ddevgh "github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/google/go-github/v72/github"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

// AddonGetCmd is the "ddev add-on get" command
var AddonGetCmd = &cobra.Command{
	Use:     "get <addonOrURL>",
	Aliases: []string{"install"},
	Args:    cobra.ExactArgs(1),
	Short:   "Get/Download a 3rd party add-on (service, provider, etc.)",
	Long:    `Get/Download a 3rd party add-on (service, provider, etc.). This can be a GitHub repo, in which case the latest release will be used, or it can be a link to a .tar.gz in the correct format (like a particular release's .tar.gz) or it can be a local directory.`,
	Example: `ddev add-on get ddev/ddev-redis
ddev add-on get ddev/ddev-redis --version v1.0.4
ddev add-on get ddev/ddev-redis --project my-project
ddev add-on get https://github.com/ddev/ddev-drupal-solr/archive/refs/tags/v1.2.3.tar.gz
ddev add-on get /path/to/package
ddev add-on get /path/to/tarball.tar.gz
`,
	Run: func(cmd *cobra.Command, args []string) {
		verbose := false
		bash := util.FindBashPath()
		requestedVersion := ""

		if cmd.Flags().Changed("version") {
			requestedVersion = cmd.Flag("version").Value.String()
		}

		if cmd.Flags().Changed("verbose") {
			verbose = true
		}

		app, err := ddevapp.GetActiveApp(cmd.Flag("project").Value.String())
		if err != nil {
			util.Failed("Unable to get project %v: %v", cmd.Flag("project").Value.String(), err)
		}
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
			defer cleanup()
			if err != nil {
				util.Failed("Unable to download %v: %v", sourceRepoArg, err)
			}
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

		injectedEnv := getInjectedEnv(app.GetConfigPath(".env."+s.Name), verbose)

		yamlMap := make(map[string]interface{})
		for name, f := range s.YamlReadFiles {
			f := os.ExpandEnv(string(f))
			fullpath := filepath.Join(app.GetAppRoot(), f)

			yamlMap[name], err = util.YamlFileToMap(fullpath)
			if err != nil {
				util.Warning("Unable to import yaml file %s: %v", fullpath, err)
			}
		}

		yamlMap["DdevGlobalConfig"], err = util.YamlFileToMap(globalconfig.GetGlobalConfigPath())
		if err != nil {
			util.Warning("Unable to read file %s: %v", globalconfig.GetGlobalConfigPath(), err)
		}

		// Get project config with overrides
		var projectConfigMap map[string]interface{}
		if b, err := yaml.Marshal(app); err != nil {
			util.Warning("Unable to marshal app: %v", err)
		} else if err = yaml.Unmarshal(b, &projectConfigMap); err != nil {
			util.Warning("Unable to unmarshal app: %v", err)
		} else {
			yamlMap["DdevProjectConfig"] = projectConfigMap
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
					util.Failed("The add-on '%s' declares a dependency on '%s'; Please ddev add-on get %s first.", s.Name, dep, dep)
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
			err = ddevapp.ProcessAddonAction(injectedEnv+"; "+action, dict, bash, verbose)
			if err != nil {
				desc := ddevapp.GetAddonDdevDescription(action)
				if err != nil {
					if !verbose {
						util.Failed("Could not process pre-install action (%d) '%s'. For more detail use ddev add-on get --verbose", i, desc)
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
				util.Warning("NOT overwriting %s. The #ddev-generated signature was not found in the file, so it will not be overwritten. You can remove the file and use ddev add-on get again if you want it to be replaced: %v", dest, err)
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
				util.Warning("NOT overwriting %s. The #ddev-generated signature was not found in the file, so it will not be overwritten. You can remove the file and use ddev add-on get again if you want it to be replaced: %v", dest, err)
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
			err = ddevapp.ProcessAddonAction(injectedEnv+"; "+action, dict, bash, verbose)
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

// getInjectedEnv returns bash export string for env variables
// that will be used in PreInstallActions and PostInstallActions
func getInjectedEnv(envFile string, verbose bool) string {
	injectedEnv := "true"
	envMap, _, err := ddevapp.ReadProjectEnvFile(envFile)
	if err != nil && !os.IsNotExist(err) {
		util.Failed("Unable to read %s file: %v", envFile, err)
	}
	if len(envMap) > 0 {
		if verbose {
			util.Warning("Using env file %s", envFile)
		}
		injectedEnv = "export"
		for k, v := range envMap {
			// Escape all spaces and dollar signs
			v = strings.ReplaceAll(strings.ReplaceAll(v, `$`, `\$`), ` `, `\ `)
			injectedEnv = injectedEnv + fmt.Sprintf(" %s=%s ", k, v)
			if verbose {
				util.Warning(`%s=%s`, k, v)
			}
		}
	}
	return injectedEnv
}

func init() {
	AddonGetCmd.Flags().String("version", "", `Specify a particular version of add-on to install`)
	AddonGetCmd.Flags().BoolP("verbose", "v", false, "Extended/verbose output")
	AddonGetCmd.Flags().String("project", "", "Name of the project to install the add-on in")
	_ = AddonGetCmd.RegisterFlagCompletionFunc("project", ddevapp.GetProjectNamesFunc("all", 0))

	AddonCmd.AddCommand(AddonGetCmd)
}
