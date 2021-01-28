package ddevapp

import (
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/version"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"fmt"

	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"

	"gopkg.in/yaml.v2"
)

// pantheonEnvironment contains meta-data about a specific environment
type pantheonEnvironment struct {
	Name string
}

// pantheonEnvironmentList provides a list of environments for a given site.
type pantheonEnvironmentList struct {
	SiteID       string
	Environments map[string]pantheonEnvironment
}

// pantheonSite is a representation of a deployed pantheon site.
type pantheonSite struct {
	ID   string
	Site struct {
		ID   string
		Name string
	}
	SiteID string
}

// PantheonProvider provides pantheon-specific import functionality.
type PantheonProvider struct {
	ProviderType     string                  `yaml:"provider"`
	app              *DdevApp                `yaml:"-"`
	Sitename         string                  `yaml:"site"`
	site             pantheonSite            `yaml:"-"`
	siteEnvironments pantheonEnvironmentList `yaml:"-"`
	EnvironmentName  string                  `yaml:"environment"`
	environment      pantheonEnvironment     `yaml:"-"`
}

// Init handles loading data from saved config.
func (p *PantheonProvider) Init(app *DdevApp) error {
	p.app = app
	configPath := app.GetConfigPath("import.yaml")
	if fileutil.FileExists(configPath) {
		err := p.Read(configPath)
		if err != nil {
			return err
		}
	}

	p.ProviderType = nodeps.ProviderPantheon
	return nil
}

// ValidateField provides field level validation for config settings. This is
// used any time a field is set via `ddev config` on the primary app config, and
// allows provider plugins to have additional validation for top level config
// settings.
func (p *PantheonProvider) ValidateField(field, value string) error {
	return nil
}

// SetSiteNameAndEnv sets the environment of the provider (dev/test/live)
func (p *PantheonProvider) SetSiteNameAndEnv(environment string) error {
	_, err := p.findPantheonSite(p.app.Name)
	if err != nil {
		return fmt.Errorf("unable to find siteName %s on Pantheon: %v", p.app.Name, err)
	}
	p.Sitename = p.app.Name
	p.EnvironmentName = environment
	return nil
}

// PromptForConfig provides interactive configuration prompts when running `ddev config pantheon`
func (p *PantheonProvider) PromptForConfig() error {
	for {
		err := p.SetSiteNameAndEnv("dev")
		if err != nil {
			output.UserOut.Errorf("%v\n", err)
			continue
		}
		err = p.environmentPrompt()
		if err != nil {
			output.UserOut.Errorf("%v\n", err)
			return err
		}
		break
	}
	return nil
}

// GetBackup will download the most recent backup specified by backupType in the given environment. If no environment
// is supplied, the configured environment will be used. Valid values for backupType are "database" or "files".
// returns fileURL, importPath, error
func (p *PantheonProvider) GetBackup(backupType, environment string) (string, string, error) {
	var err error
	if backupType != "database" && backupType != "files" {
		return "", "", fmt.Errorf("could not get backup: %s is not a valid backup type", backupType)
	}

	// If the user hasn't defined an environment override, use the configured value.
	if environment == "" {
		environment = p.EnvironmentName
	}

	// Set the import path blank to use the root of the archive by default.
	importPath := ""
	err = p.environmentExists(environment)
	if err != nil {
		return "", "", err
	}

	link, filename, err := p.getBackup(backupType, environment)
	if err != nil {
		return "", "", err
	}

	p.prepDownloadDir()
	destFile := filepath.Join(p.getDownloadDir(), filename)

	// Check to see if this file has been downloaded previously.
	// Attempt a new download If we can't stat the file or we get a mismatch on the filesize.
	_, err = os.Stat(destFile)
	if err != nil {
		err = util.DownloadFile(destFile, link, true)
		if err != nil {
			return "", "", err
		}
	}

	if backupType == "files" {
		importPath = fmt.Sprintf("files_%s", environment)
	}

	return destFile, importPath, nil
}

// prepDownloadDir ensures the download cache directories are created and writeable.
func (p *PantheonProvider) prepDownloadDir() {
	destDir := p.getDownloadDir()
	filesDir := filepath.Join(destDir, "files")
	_ = os.RemoveAll(destDir)
	err := os.MkdirAll(filesDir, 0755)
	util.CheckErr(err)
}

func (p *PantheonProvider) getDownloadDir() string {
	destDir := p.app.GetConfigPath(".downloads")
	return destDir
}

// getBackup will return a URL for the most recent backup of archiveType.
func (p *PantheonProvider) getBackup(archiveType string, environment string) (link string, filename string, error error) {

	element := "files"
	if archiveType == "database" {
		element = "db" // how it's specified in terminus
	}

	uid, _, _ := util.GetContainerUIDGid()
	cmd := fmt.Sprintf("disable_xdebug >/dev/null && terminus backup:info --format=string --fields=Filename,URL --element=%s %s.%s", element, p.site.ID, environment)
	_, result, err := dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", cmd}, nil, []string{"HOME=/tmp", "DDEV_XDEBUG_ENABLED=false"}, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, uid, true, false)

	if err != nil {
		return "", "", fmt.Errorf("unable to get terminus backup: %v (%v)", err, result)
	}

	result = strings.Trim(result, "\n\r ")
	results := strings.Split(result, "\t")
	if len(results) != 2 {
		return "", "", fmt.Errorf("terminus result does not provide filename and url: %v", result)
	}
	filename = results[0]
	link = results[1]
	return link, filename, nil
}

// environmentPrompt contains the user prompts for interactive configuration of the pantheon environment.
func (p *PantheonProvider) environmentPrompt() error {
	envs, err := p.GetEnvironments()
	if err != nil {
		return err
	}

	if p.EnvironmentName == "" {
		p.EnvironmentName = "dev"
	}

	fmt.Println("\nConfigure Pantheon environment:")

	fmt.Println("\n\t- " + strings.Join(envs, "\n\t- ") + "\n")
	var environmentPrompt = "Type the name to select an environment to pull from"
	if p.EnvironmentName != "" {
		environmentPrompt = fmt.Sprintf("%s (%s)", environmentPrompt, p.EnvironmentName)
	}

	fmt.Print(environmentPrompt + ": ")
	envName := util.GetInput(p.EnvironmentName)

	ok := nodeps.ArrayContainsString(envs, envName)

	if !ok {
		return fmt.Errorf("could not find an environment named '%s'", envName)
	}
	err = p.SetSiteNameAndEnv(envName)
	if err != nil {
		return err
	}
	return nil
}

// Write the pantheon provider configuration to a specified location on disk.
func (p *PantheonProvider) Write(configPath string) error {
	err := PrepDdevDirectory(filepath.Dir(configPath))
	if err != nil {
		return err
	}

	cfgbytes, err := yaml.Marshal(p)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(configPath, cfgbytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Read pantheon provider configuration from a specified location on disk.
func (p *PantheonProvider) Read(configPath string) error {
	source, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}

	// Read config values from file.
	err = yaml.Unmarshal(source, p)
	if err != nil {
		return err
	}

	return nil
}

// GetEnvironments will return a list of environments for the currently configured pantheon site.
func (p *PantheonProvider) GetEnvironments() ([]string, error) {
	if p.site.ID == "" {
		id, err := p.findPantheonSite(p.Sitename)
		if err != nil {
			return []string{}, err
		}

		p.site.ID = id
	}

	// Get a list of all active environments for the current site.
	cmd := "disable_xdebug >/dev/null && terminus env:list --field=id " + p.site.ID
	uid, _, _ := util.GetContainerUIDGid()
	_, envs, err := dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", cmd}, nil, []string{"HOME=/tmp", "DDEV_XDEBUG_ENABLED=false"}, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, uid, true, false)

	if err != nil {
		return []string{}, fmt.Errorf("unable to get Pantheon environments for project %s - Have you authenticated with `ddev auth pantheon`? Does the ddev project name match the pantheon project name? ('%v' failed)", p.Sitename, cmd)
	}

	envs = strings.Trim(envs, "\n\r ")
	return strings.Split(envs, "\n"), err
}

// Validate ensures that the current configuration is valid (i.e. the configured pantheon site/environment exists)
func (p *PantheonProvider) Validate() error {
	return p.environmentExists(p.EnvironmentName)
}

// environmentExists ensures the currently configured pantheon site & environment exists.
func (p *PantheonProvider) environmentExists(environment string) error {
	el, err := p.GetEnvironments()
	if err != nil {
		return err
	}
	if !nodeps.ArrayContainsString(el, environment) {
		return fmt.Errorf("could not find an environment named '%s'", environment)
	}

	return nil
}

// findPantheonSite ensures the pantheon site specified by name exists, and the current user has access to it.
func (p *PantheonProvider) findPantheonSite(name string) (id string, e error) {

	uid, _, _ := util.GetContainerUIDGid()
	_, id, err := dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", "disable_xdebug >/dev/null && terminus site:info --fields=id --format=json " + name + " | jq -r .id"}, nil, []string{"HOME=/tmp", "DDEV_XDEBUG_ENABLED=false"}, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, uid, true, false)

	if err != nil {
		return "", fmt.Errorf("could not find a pantheon site named %s: %v (%v)", name, err, id)
	}

	id = strings.Trim(id, "\n\r ")
	return id, nil
}
