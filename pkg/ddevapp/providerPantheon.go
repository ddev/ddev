package ddevapp

import (
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"fmt"

	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"

	"github.com/drud/go-pantheon/pkg/pantheon"
	"gopkg.in/yaml.v2"
)

// PantheonProvider provides pantheon-specific import functionality.
type PantheonProvider struct {
	ProviderType     string                   `yaml:"provider"`
	app              *DdevApp                 `yaml:"-"`
	Sitename         string                   `yaml:"site"`
	site             pantheon.Site            `yaml:"-"`
	siteEnvironments pantheon.EnvironmentList `yaml:"-"`
	EnvironmentName  string                   `yaml:"environment"`
	environment      pantheon.Environment     `yaml:"-"`
}

// Init handles loading data from saved config.
func (p *PantheonProvider) Init(app *DdevApp) error {
	p.app = app
	configPath := app.GetConfigPath("import.yaml")
	if fileutil.FileExists(configPath) {
		err := p.Read(configPath)
		return err
	}

	p.ProviderType = nodeps.ProviderPantheon
	err := p.authPantheon()
	return err
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
	_, err := findPantheonSite(p.app.Name)
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
			continue
		}
		break
	}
	return nil
}

// GetBackup will download the most recent backup specified by backupType in the given environment. If no environment
// is supplied, the configured environment will be used. Valid values for backupType are "database" or "files".
func (p *PantheonProvider) GetBackup(backupType, environment string) (fileURL string, importPath string, err error) {
	if backupType != "database" && backupType != "files" {
		return "", "", fmt.Errorf("could not get backup: %s is not a valid backup type", backupType)
	}

	// If the user hasn't defined an environment override, use the configured value.
	if environment == "" {
		environment = p.EnvironmentName
	}

	// Set the import path blank to use the root of the archive by default.
	importPath = ""
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
	err := os.MkdirAll(destDir, 0755)
	util.CheckErr(err)
}

func (p *PantheonProvider) getDownloadDir() string {
	globalDir := globalconfig.GetGlobalDdevDir()
	destDir := filepath.Join(globalDir, "pantheon", p.app.Name)

	return destDir
}

// getBackup will return a URL for the most recent backup of archiveType.
func (p *PantheonProvider) getBackup(archiveType string, environment string) (link string, filename string, error error) {

	element := "files"
	if archiveType == "database" {
		element = "db" // how it's specified in terminus
	}

	result, stderr, err := p.app.Exec(&ExecOpts{
		Cmd: fmt.Sprintf("terminus backup:info --format=string --fields=Filename,URL --element=%s %s.%s", element, p.site.ID, environment),
	})
	if err != nil {
		return "", "", fmt.Errorf("unable to get terminus backup: %v stderr=%v", err, stderr)
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
	_, err := p.GetEnvironments()
	if err != nil {
		return err
	}

	if p.EnvironmentName == "" {
		p.EnvironmentName = "dev"
	}

	fmt.Println("\nConfigure import environment:")

	keys := make([]string, 0, len(p.siteEnvironments.Environments))
	for k := range p.siteEnvironments.Environments {
		keys = append(keys, k)
	}
	fmt.Println("\n\t- " + strings.Join(keys, "\n\t- ") + "\n")
	var environmentPrompt = "Type the name to select an environment to pull from"
	if p.EnvironmentName != "" {
		environmentPrompt = fmt.Sprintf("%s (%s)", environmentPrompt, p.EnvironmentName)
	}

	fmt.Print(environmentPrompt + ": ")
	envName := util.GetInput(p.EnvironmentName)

	_, ok := p.siteEnvironments.Environments[envName]

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
	// TODO: Cache the list of environments

	if p.site.ID == "" {
		id, err := p.findPantheonSite(p.Sitename)
		if err != nil {
			return []string{}, err
		}

		p.site.ID = id
	}

	// Get a list of all active environments for the current site.
	envs, stderr, err := p.app.Exec(&ExecOpts{
		Cmd: "terminus env:list --field=id " + p.site.ID,
	})
	if err != nil {
		return []string{}, fmt.Errorf("unable to get environments: %v stderr=%v", err, stderr)
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

	id, stderr, err := p.app.Exec(&ExecOpts{
		Cmd: "terminus site:info --field=id " + name,
	})
	if err != nil {
		return "", fmt.Errorf("could not find a pantheon site named %s: %v", name, stderr)
	}
	// TODO: If more than one site, error out and somehow require the hash instead.

	id = strings.Trim(id, "\n\r ")
	return id, nil
}

// authPantheon does a terminus login; it only needs to be done once in life of container.
func (p *PantheonProvider) authPantheon() error {
	token := os.Getenv("DDEV_PANTHEON_API_TOKEN")
	if token == "" {
		return fmt.Errorf("environment variable DDEV_PANTHEON_API_TOKEN not found")
	}

	_, _, err := p.app.Exec(&ExecOpts{
		Cmd: "terminus auth:login --machine-token=" + token,
	})

	return err
}
