package ddevapp

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/mitchellh/go-homedir"

	"fmt"

	"github.com/drud/go-pantheon/pkg/pantheon"
	"gopkg.in/yaml.v2"
)

// PantheonProvider provides pantheon-specific import functionality.
type PantheonProvider struct {
	ProviderType     string                   `yaml:"provider"`
	config           *Config                  `yaml:"-"`
	Sitename         string                   `yaml:"site"`
	site             pantheon.Site            `yaml:"-"`
	siteEnvironments pantheon.EnvironmentList `yaml:"-"`
	Environment      string                   `yaml:"environment"`
	environment      pantheon.Environment     `yaml:"-"`
}

// Init handles loading data from saved config.
func (p *PantheonProvider) Init(config *Config) error {
	var err error
	p.config = config

	configPath := config.GetPath("import.yaml")
	if fileutil.FileExists(configPath) {
		err = p.Read(configPath)
	}

	p.ProviderType = "pantheon"
	return err
}

// ValidateField provides field level validation for config settings. This is used any time a field is set via `ddev config` on the primary app config, and allows
// provider plugins to have additional validation for top level config settings.
func (p *PantheonProvider) ValidateField(field, value string) error {
	switch field {
	case "Name":
		_, err := findPantheonSite(value)
		if err != nil {
			p.Sitename = value
		}
		return err
	}
	return nil
}

// Config provides interactive configuration prompts when running `ddev config pantheon`
func (p *PantheonProvider) Config() error {
	p.Sitename = p.config.Name
	for {
		err := p.environmentPrompt()

		if err == nil {
			break
		}

		fmt.Printf("%v\n", err)
	}

	return nil
}

// GetBackup will download the most recent backup specified by backupType. Valid values for backupType are "database" or "files".
func (p *PantheonProvider) GetBackup(backupType string) (fileLocation string, importPath string, err error) {
	if backupType != "database" && backupType != "files" {
		return "", "", fmt.Errorf("could not get backup: %s is not a valid backup type", backupType)
	}

	// Set the import path blank, to use the root of the archive by default.
	importPath = ""
	err = p.environmentExists()
	if err != nil {
		return "", "", err
	}

	session := getPantheonSession()

	// Find either a files or database backup, depending on what was asked for.
	bl := pantheon.NewBackupList(p.site.ID, p.Environment)
	err = session.Request("GET", bl)
	if err != nil {
		return "", "", err
	}

	backup, err := p.getPantheonBackupLink(backupType, bl, session)
	if err != nil {
		return "", "", err
	}

	p.prepDownloadDir()
	destFile := filepath.Join(p.getDownloadDir(), backup.FileName)

	// Check to see if this file has been downloaded previously.
	// Attempt a new download If we can't stat the file or we get a mismatch on the filesize.
	stat, err := os.Stat(destFile)
	if err != nil || stat.Size() != int64(backup.Size) {
		err = util.DownloadFile(destFile, backup.DownloadURL, true)
		if err != nil {
			return "", "", err
		}
	}

	if backupType == "files" {
		importPath = fmt.Sprintf("files_%s", p.Environment)
	}

	return destFile, importPath, nil
}

// prepDownloadDir ensures the local download cache directories are created and writeable.
func (p *PantheonProvider) prepDownloadDir() {
	destDir := p.getDownloadDir()
	err := os.MkdirAll(destDir, 0755)
	util.CheckErr(err)
}

func (p *PantheonProvider) getDownloadDir() string {
	userDir, err := homedir.Dir()
	util.CheckErr(err)
	destDir := filepath.Join(userDir, ".ddev", "pantheon", p.config.Name)
	return destDir
}

// getPantheonBackupLink will return a URL for the most recent backyp of archiveType that exist with the BackupList specified.
func (p *PantheonProvider) getPantheonBackupLink(archiveType string, bl *pantheon.BackupList, session *pantheon.AuthSession) (*pantheon.Backup, error) {
	latestBackup := pantheon.Backup{}
	for i, backup := range bl.Backups {
		if backup.ArchiveType == archiveType && backup.Timestamp > latestBackup.Timestamp {
			latestBackup = bl.Backups[i]
		}
	}

	if latestBackup.Timestamp != 0 {
		// Get a time-limited backup URL from Pantheon. This requires a POST of the backup type to their API.
		err := session.Request("POST", &latestBackup)
		if err != nil {
			return &pantheon.Backup{}, fmt.Errorf("could not get backup URL: %v", err)
		}

		return &latestBackup, nil
	}

	// If no matches were found, just return an empty backup along with an error.
	return &pantheon.Backup{}, fmt.Errorf("could not find a backup of type %s. please visit your pantheon dashboard and ensure the '%s' environment has a backup available", archiveType, p.Environment)
}

// environmentPrompt contains the user prompts for interactive configuration of the pantheon environment.
func (p *PantheonProvider) environmentPrompt() error {
	_, err := p.GetEnvironments()
	if err != nil {
		return err
	}

	if p.Environment == "" {
		p.Environment = "dev"
	}

	fmt.Println("\nConfigure import environment:")

	keys := make([]string, 0, len(p.siteEnvironments.Environments))
	for k := range p.siteEnvironments.Environments {
		keys = append(keys, k)
	}
	fmt.Println("\n\t- " + strings.Join(keys, "\n\t- ") + "\n")
	var environmentPrompt = "Type the name to select an environment to import from"
	if p.Environment != "" {
		environmentPrompt = fmt.Sprintf("%s (%s)", environmentPrompt, p.Environment)
	}

	fmt.Print(environmentPrompt + ": ")
	p.Environment = util.GetInput(p.Environment)

	environment, ok := p.siteEnvironments.Environments[p.Environment]
	p.environment = environment

	if !ok {
		return fmt.Errorf("could not find an environment named '%s'", p.Environment)
	}
	return nil
}

// Write the pantheon provider configuration to a spcified location on disk.
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

// GetEnvironments will return a list of environments for the currently configured upstream pantheon site.
func (p *PantheonProvider) GetEnvironments() (pantheon.EnvironmentList, error) {
	var el *pantheon.EnvironmentList
	// If we've got an already populated environment list, then just use that.
	if len(p.siteEnvironments.Environments) > 0 {
		return p.siteEnvironments, nil
	}

	// Otherwise we need to find our environments.
	session := getPantheonSession()

	if p.site.ID == "" {
		site, err := findPantheonSite(p.Sitename)
		if err != nil {
			return p.siteEnvironments, err
		}

		p.site = site
	}

	// Get a list of all active environments for the current site.
	el = pantheon.NewEnvironmentList(p.site.ID)
	err := session.Request("GET", el)
	p.siteEnvironments = *el
	return *el, err
}

// Validate ensures that the current configuration is valid (i.e. the configured pantheon site/environment exists)
func (p *PantheonProvider) Validate() error {
	return p.environmentExists()
}

// environmentExists ensures the currently configured pantheon site & environment exists.
func (p *PantheonProvider) environmentExists() error {
	_, err := p.GetEnvironments()
	if err != nil {
		return err
	}

	_, ok := p.siteEnvironments.Environments[p.Environment]

	if !ok {
		return fmt.Errorf("could not find an environment named '%s'", p.Environment)
	}

	return nil
}

// findPantheonSite ensures the pantheon site specified by name exists, and the current user has access to it.
func findPantheonSite(name string) (pantheon.Site, error) {
	session := getPantheonSession()

	// Get a list of all sites the current user has access to. Ensure we can find the site which was used in the CLI arguments in that list.
	sl := &pantheon.SiteList{}
	err := session.Request("GET", sl)
	if err != nil {
		return pantheon.Site{}, err
	}

	// Get a list of environments for a given site.
	for i, site := range sl.Sites {
		if site.Site.Name == name {
			return sl.Sites[i], nil
		}
	}

	return pantheon.Site{}, fmt.Errorf("could not find a pantheon site named %s", name)
}

// getPantheonSession loads the pantheon API config from disk and returns a pantheon session struct.
func getPantheonSession() *pantheon.AuthSession {
	userDir, err := homedir.Dir()
	util.CheckErr(err)
	sessionLocation := filepath.Join(userDir, ".ddev", "pantheonconfig.json")

	// Generate a session object based on the DDEV_PANTHEON_API_TOKEN environment var.
	session := &pantheon.AuthSession{}

	// Read a previously saved session.
	err = session.Read(sessionLocation)

	if err != nil {
		// If we can't read a previous session fall back to using the API token.
		apiToken := os.Getenv("DDEV_PANTHEON_API_TOKEN")
		if apiToken == "" {
			util.Failed("No saved session could be found and the environment variable DDEV_PANTHEON_API_TOKEN is not set. Please use ddev auth-pantheon or set a DDEV_PANTHEON_API_TOKEN. https://pantheon.io/docs/machine-tokens/ provides instructions on creating a token.")
		}
		session = pantheon.NewAuthSession(os.Getenv("DDEV_PANTHEON_API_TOKEN"))
	}

	err = session.Auth()
	if err != nil {
		log.Fatalf("Could not authenticate with pantheon: %v", err)
	}

	err = session.Write(sessionLocation)
	util.CheckErr(err)

	return session
}
