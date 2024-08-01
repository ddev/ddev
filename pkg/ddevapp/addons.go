package ddevapp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	github2 "github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/google/go-github/v52/github"
	"gopkg.in/yaml.v3"
)

const AddonMetadataDir = "addon-metadata"

// Format of install.yaml
type InstallDesc struct {
	// Name must be unique in a project; it will overwrite any existing add-on with the same name.
	Name                  string            `yaml:"name"`
	ProjectFiles          []string          `yaml:"project_files"`
	GlobalFiles           []string          `yaml:"global_files,omitempty"`
	DdevVersionConstraint string            `yaml:"ddev_version_constraint,omitempty"`
	Dependencies          []string          `yaml:"dependencies,omitempty"`
	PreInstallActions     []string          `yaml:"pre_install_actions,omitempty"`
	PostInstallActions    []string          `yaml:"post_install_actions,omitempty"`
	RemovalActions        []string          `yaml:"removal_actions,omitempty"`
	YamlReadFiles         map[string]string `yaml:"yaml_read_files"`
}

// format of the add-on manifest file
type AddonManifest struct {
	Name           string   `yaml:"name"`
	Repository     string   `yaml:"repository"`
	Version        string   `yaml:"version"`
	Dependencies   []string `yaml:"dependencies,omitempty"`
	InstallDate    string   `yaml:"install_date"`
	ProjectFiles   []string `yaml:"project_files"`
	GlobalFiles    []string `yaml:"global_files"`
	RemovalActions []string `yaml:"removal_actions"`
}

// GetInstalledAddons returns a list of the installed add-ons
func GetInstalledAddons(app *DdevApp) []AddonManifest {
	metadataDir := app.GetConfigPath(AddonMetadataDir)
	err := os.MkdirAll(metadataDir, 0755)
	if err != nil {
		util.Failed("Error creating metadata directory: %v", err)
	}
	// Read the contents of the .ddev/addon-metadata directory (directories)
	dirs, err := os.ReadDir(metadataDir)
	if err != nil {
		util.Failed("Error reading metadata directory: %v", err)
	}
	manifests := []AddonManifest{}

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
			var manifest AddonManifest
			err = yaml.Unmarshal(manifestBytes, &manifest)
			if err != nil {
				util.Failed("Unable to parse manifest file: %v", err)
			}
			manifests = append(manifests, manifest)
		}
	}
	return manifests
}

// GetInstalledAddonNames returns a list of the names of installed add-ons
func GetInstalledAddonNames(app *DdevApp) []string {
	manifests := GetInstalledAddons(app)
	names := []string{}
	for _, manifest := range manifests {
		names = append(names, manifest.Name)
	}
	return names
}

// GetInstalledAddonProjectFiles returns a list of project files installed by add-ons
func GetInstalledAddonProjectFiles(app *DdevApp) []string {
	manifests := GetInstalledAddons(app)
	uniqueFilesMap := make(map[string]struct{})
	for _, manifest := range manifests {
		for _, file := range manifest.ProjectFiles {
			uniqueFilesMap[filepath.Join(app.AppConfDir(), file)] = struct{}{}
		}
	}
	uniqueFiles := make([]string, 0, len(uniqueFilesMap))
	for file := range uniqueFilesMap {
		uniqueFiles = append(uniqueFiles, file)
	}
	return uniqueFiles
}

// ProcessAddonAction takes a stanza from yaml exec section and executes it.
func ProcessAddonAction(action string, dict map[string]interface{}, bashPath string, verbose bool) error {
	action = "set -eu -o pipefail\n" + action
	t, err := template.New("ProcessAddonAction").Funcs(getTemplateFuncMap()).Parse(action)
	if err != nil {
		return fmt.Errorf("could not parse action '%s': %v", action, err)
	}

	var doc bytes.Buffer
	err = t.Execute(&doc, dict)
	if err != nil {
		return fmt.Errorf("could not parse/execute action '%s': %v", action, err)
	}
	action = doc.String()

	desc := GetAddonDdevDescription(action)
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

// GetAddonDdevDescription returns what follows #ddev-description: in any line in action
func GetAddonDdevDescription(action string) string {
	descLines := nodeps.GrepStringInBuffer(action, `[\r\n]+#ddev-description:.*[\r\n]+`)
	if len(descLines) > 0 {
		d := strings.Split(descLines[0], ":")
		if len(d) > 1 {
			return strings.Trim(d[1], "\r\n\t")
		}
	}
	return ""
}

// ListAvailableAddons lists the add-ons that are listed on github
func ListAvailableAddons(officialOnly bool) ([]*github.Repository, error) {
	client := github2.GetGithubClient(context.Background())
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

// RemoveAddon removes an addon, taking care to respect #ddev-generated
// addonName can be the "Name", or the full "Repository" like ddev/ddev-redis, or
// the final par of the repository name like ddev-redis
func RemoveAddon(app *DdevApp, addonName string, dict map[string]interface{}, bash string, verbose bool) error {
	if addonName == "" {
		return fmt.Errorf("No add-on name specified for removal")
	}

	manifests, err := GatherAllManifests(app)
	if err != nil {
		util.Failed("Unable to gather all manifests: %v", err)
	}

	var manifestData AddonManifest
	var ok bool

	if manifestData, ok = manifests[addonName]; !ok {
		util.Failed("The add-on '%s' does not seem to have a manifest file; please upgrade it.\nUse `ddev get --installed to see installed add-ons.\nIf yours is not there it may have been installed before DDEV v1.22.0.\nUse 'ddev get' to update it.", addonName)
	}

	// Execute any removal actions
	for i, action := range manifestData.RemovalActions {
		err = ProcessAddonAction(action, dict, bash, verbose)
		desc := GetAddonDdevDescription(action)
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

	err = os.RemoveAll(app.GetConfigPath(filepath.Join(AddonMetadataDir, manifestData.Name)))
	if err != nil {
		return fmt.Errorf("Error removing addon metadata directory %s: %v", manifestData.Name, err)
	}
	util.Success("Removed add-on %s", addonName)
	return nil
}

// GatherAllManifests searches for all addon manifests and presents the result
// as a map of various names to manifest data
func GatherAllManifests(app *DdevApp) (map[string]AddonManifest, error) {
	metadataDir := app.GetConfigPath(AddonMetadataDir)
	allManifests := make(map[string]AddonManifest)
	err := os.MkdirAll(metadataDir, 0755)
	if err != nil {
		return nil, err
	}

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
		var manifestData = &AddonManifest{}
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
