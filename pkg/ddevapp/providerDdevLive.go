package ddevapp

import (
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/version"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"fmt"

	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"

	"gopkg.in/yaml.v2"
)

// ddevLiveSite is a representation of a deployed ddevLive site.
type ddevLiveSite struct {
	ID   string
	Site struct {
		Name string
	}
	SiteID string
}

// DdevLiveProvider provides ddevLive-specific import functionality.
type DdevLiveProvider struct {
	ProviderType string       `yaml:"provider"`
	app          *DdevApp     `yaml:"-"`
	Sitename     string       `yaml:"site"`
	site         ddevLiveSite `yaml:"-"`
}

// Init handles loading data from saved config.
func (p *DdevLiveProvider) Init(app *DdevApp) error {
	p.app = app
	configPath := app.GetConfigPath("import.yaml")
	if fileutil.FileExists(configPath) {
		err := p.Read(configPath)
		if err != nil {
			return err
		}
	}

	p.ProviderType = nodeps.ProviderDdevLive
	return nil
}

// ValidateField provides field level validation for config settings. This is
// used any time a field is set via `ddev config` on the primary app config, and
// allows provider plugins to have additional validation for top level config
// settings.
func (p *DdevLiveProvider) ValidateField(field, value string) error {
	return nil
}

// SetSiteNameAndEnv sets the environment of the provider (dev/test/live)
func (p *DdevLiveProvider) SetSiteNameAndEnv(environment string) error {
	_, err := p.findddevLiveSite(p.app.Name)
	if err != nil {
		return fmt.Errorf("unable to find siteName %s on ddevLive: %v", p.app.Name, err)
	}
	p.Sitename = p.app.Name
	return nil
}

// PromptForConfig provides interactive configuration prompts when running `ddev config ddevLive`
func (p *DdevLiveProvider) PromptForConfig() error {
	return nil
}

// GetBackup will download the most recent backup specified by backupType in the given environment. If no environment
// is supplied, the configured environment will be used. Valid values for backupType are "database" or "files".
// returns fileURL, importPath, error
func (p *DdevLiveProvider) GetBackup(backupType, environment string) (string, string, error) {
	var err error
	if backupType != "database" && backupType != "files" {
		return "", "", fmt.Errorf("could not get backup: %s is not a valid backup type", backupType)
	}

	// Set the import path blank to use the root of the archive by default.
	importPath := ""

	link, filename, err := p.getBackup(backupType, "")
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
func (p *DdevLiveProvider) prepDownloadDir() {
	destDir := p.getDownloadDir()
	err := os.MkdirAll(destDir, 0755)
	util.CheckErr(err)
}

func (p *DdevLiveProvider) getDownloadDir() string {
	globalDir := globalconfig.GetGlobalDdevDir()
	destDir := filepath.Join(globalDir, "ddevLive", p.app.Name)

	return destDir
}

// getBackup will return a URL for the most recent backup of archiveType.
func (p *DdevLiveProvider) getBackup(archiveType string, environment string) (link string, filename string, error error) {

	element := "files"
	if archiveType == "database" {
		element = "db" // how it's specified in terminus
	}

	uid, _, _ := util.GetContainerUIDGid()
	cmd := fmt.Sprintf("terminus backup:info --format=string --fields=Filename,URL --element=%s %s.%s", element, p.site.ID, environment)
	_, result, err := dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", cmd}, nil, []string{"HOME=/tmp"}, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, uid, true)

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

// Write the ddevLive provider configuration to a specified location on disk.
func (p *DdevLiveProvider) Write(configPath string) error {
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

// Read ddevLive provider configuration from a specified location on disk.
func (p *DdevLiveProvider) Read(configPath string) error {
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

// findddevLiveSite ensures the ddevLive site specified by name exists, and the current user has access to it.
func (p *DdevLiveProvider) findddevLiveSite(name string) (id string, e error) {
	uid, _, _ := util.GetContainerUIDGid()
	_, id, err := dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", "ddev-live list sites -o json " + name + " | jq -r .id"}, nil, []string{"HOME=/tmp"}, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, uid, true)

	if err != nil {
		return "", fmt.Errorf("could not find a ddevLive site named %s: %v (%v)", name, err, id)
	}

	id = strings.Trim(id, "\n\r ")
	return id, nil
}

// Validate ensures that the current configuration is valid (i.e. the configured pantheon site/environment exists)
func (p *DdevLiveProvider) Validate() error {
	return nil
}
