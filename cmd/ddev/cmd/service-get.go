package cmd

import (
	"context"
	"fmt"
	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/google/go-github/github"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"strings"
)

type serviceDesc struct {
	Name               string   `yaml:"name"`
	Files              []string `yaml:"files"`
	PreInstallActions  []string `yaml:"pre_install_actions,omitempty"`
	PostInstallActions []string `yaml:"post_install_actions,omitempty"`
}

// ServiceGet implements the ddev service get command
var ServiceGet = &cobra.Command{
	Use:   "get servicename [project]",
	Short: "Get/Download a 3rd party service",
	Long:  `Get/Download a 3rd party service. This can be a github repo, in which case the latest release will be used, or it can be a local directory.`,
	Example: `ddev service get rfay/solr
ddev service get /path/to/service`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			util.Failed("You must specify a service to enable")
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
		serviceRepo := args[0]
		removeDir := ""
		srcDest := ""
		parts := strings.Split(serviceRepo, "/")

		switch {
		case fileutil.IsDirectory(serviceRepo):
			// Use the directory as the source
			srcDest = serviceRepo

		case len(parts) == 2: // github.com/user/repo
			// assume it's a github repo
			if len(parts) != 2 {
				util.Failed("Invalid service name %s", serviceRepo)
			}
			owner := parts[0]
			repo := parts[1]
			client := github.NewClient(nil)
			ctx := context.Background()
			releases, _, err := client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{})
			if err != nil {
				util.Failed("Unable to get releases for %v: %v", serviceRepo, err)
			}
			if len(releases) == 0 {
				util.Failed("No releases found for %v", serviceRepo)
			}
			f, err := os.CreateTemp("", fmt.Sprintf("%s-%s*.tar.gz", owner, repo))
			if err != nil {
				util.Failed("Unable to create temp file: %v", err)
			}
			defer func() {
				_ = f.Close()
			}()

			util.Success("Downloading latest release from github.com/%v (%s)", serviceRepo, releases[0].GetTarballURL())
			tarball := f.Name()
			defer os.RemoveAll(tarball)
			err = util.DownloadFile(tarball, releases[0].GetTarballURL(), true)
			if err != nil {
				util.Failed("Unable to download %v: %v", releases[0].GetTarballURL(), err)
			}
			srcDest, err = os.MkdirTemp("", "service_repo_")
			if err != nil {
				util.Failed("Unable to create temp dir: %v", err)
			}
			defer os.RemoveAll(srcDest)

			err = archive.Untar(tarball, srcDest, "")
			if err != nil {
				util.Failed("Unable to untar %v: %v", srcDest, err)
			}

			list, err := fileutil.ListFilesInDir(srcDest)
			if err != nil {
				util.Failed("Unable to list files in %v: %v", srcDest, err)
			}
			if len(list) == 0 {
				util.Failed("No files found in %v", srcDest)
			}
			removeDir = list[0]
		default:
			util.Failed("Invalid service repo or path '%s'", serviceRepo)
		}
		yamlFile := filepath.Join(srcDest, removeDir, "install.yaml")
		yamlContent, err := fileutil.ReadFileIntoString(yamlFile)
		if err != nil {
			util.Failed("Unable to read %v: %v", yamlFile, err)
		}
		var s serviceDesc
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

		for _, file := range s.Files {
			src := filepath.Join(srcDest, removeDir, file)
			//TODO: How dangerous is this? Rethink.
			_ = os.RemoveAll(app.GetConfigPath(file))
			if fileutil.IsDirectory(src) {
				err = fileutil.CopyDir(src, app.GetConfigPath(file))
			} else {
				err = fileutil.CopyFile(src, app.GetConfigPath(file))
			}
			if err != nil {
				util.Failed("Unable to copy %v to %v: %v", src, app.GetConfigPath(file), err)
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

		util.Success("Downloaded and enabled service %s, use `ddev restart` to turn it on.", serviceRepo)
	},
}

func init() {
	ServiceCmd.AddCommand(ServiceGet)
}
