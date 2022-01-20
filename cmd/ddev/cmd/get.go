package cmd

import (
	"context"
	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/util"
	"github.com/google/go-github/github"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
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
	Long:  `Get/Download a 3rd party add-on (service, provider, etc.). This can be a github repo, in which case the latest release will be used, or it can be a link to a .tar.gz in the correct format (like a particular release's .tar.gz) or it can be a local directory.`,
	Example: `ddev get drud/ddev-drupal9-solr
ddev get https://github.com/drud/ddev-drupal9-solr/archive/refs/tags/v0.0.5.tar.gz
ddev get /path/to/package
ddev get /path/to/tarball.tar.gz`,
	Run: func(cmd *cobra.Command, args []string) {
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

		switch {
		// If the provided sourceRepoArg is a directory, then we will use that as the source
		case fileutil.IsDirectory(sourceRepoArg):
			// Use the directory as the source
			extractedDir = sourceRepoArg

		// if sourceRepoArg is a tarball on local filesystem, we can use that
		case fileutil.FileExists(sourceRepoArg) && (strings.HasSuffix(filepath.Base(sourceRepoArg), "tar.gz") || strings.HasSuffix(filepath.Base(sourceRepoArg), "tar") || strings.HasSuffix(filepath.Base(sourceRepoArg), "tgz")):
			// If the provided sourceRepoArg is a file, then we will use that as the source
			extractedDir, cleanup, err = archive.ExtractTarballWithCleanup(sourceRepoArg, true)
			if err != nil {
				util.Failed("Unable to extract %s: %v", sourceRepoArg, err)
			}
			defer cleanup()

		// If the provided sourceRepoArg is a github sourceRepoArg, then we will use that as the source
		case len(parts) == 2: // github.com/owner/sourceRepoArg
			owner := parts[0]
			repo := parts[1]
			client := github.NewClient(nil)
			ctx := context.Background()
			releases, _, err := client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{})
			if err != nil {
				util.Failed("Unable to get releases for %v: %v", repo, err)
			}
			if len(releases) == 0 {
				util.Failed("No releases found for %v", repo)
			}
			tarballURL = releases[0].GetTarballURL()
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
			util.Success("%v\n%s", action, out)
		}

		for _, file := range s.ProjectFiles {
			src := filepath.Join(extractedDir, file)
			dest := app.GetConfigPath(file)

			err = copy.Copy(src, dest)
			if err != nil {
				util.Failed("Unable to copy %v to %v: %v", src, dest, err)
			}
		}
		globalDotDdev := filepath.Join(globalconfig.GetGlobalDdevDir())
		for _, file := range s.GlobalFiles {
			src := filepath.Join(extractedDir, file)
			dest := filepath.Join(globalDotDdev, file)
			err = copy.Copy(src, dest)
			if err != nil {
				util.Failed("Unable to copy %v to %v: %v", src, dest, err)
			}
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
		}

		util.Success("Downloaded add-on %s, use `ddev restart` to enable.", sourceRepoArg)
	},
}

func init() {
	RootCmd.AddCommand(Get)
}
