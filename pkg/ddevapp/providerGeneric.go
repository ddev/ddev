package ddevapp

import (
	"github.com/drud/ddev/pkg/output"
	"io/ioutil"
	"os"
	"path/filepath"

	"fmt"

	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"

	"gopkg.in/yaml.v2"
)

// GenericProvider provides generic-specific import functionality.
type GenericProvider struct {
	ProviderType string   `yaml:"provider"`
	app          *DdevApp `yaml:"-"`
	SiteID       string   `yaml:"site_id"`
}

// Init handles loading data from saved config.
func (p *GenericProvider) Init(app *DdevApp) error {
	p.app = app
	configPath := app.GetConfigPath("import.yaml")
	if fileutil.FileExists(configPath) {
		err := p.Read(configPath)
		if err != nil {
			return err
		}
	}

	p.ProviderType = app.Provider
	return nil
}

// ValidateField provides field level validation for config settings. This is
// used any time a field is set via `ddev config` on the primary app config, and
// allows provider plugins to have additional validation for top level config
// settings.
func (p *GenericProvider) ValidateField(field, value string) error {
	return nil
}

// PromptForConfig provides interactive configuration prompts when running `ddev config generic`
func (p *GenericProvider) PromptForConfig() error {
	return nil
}

// SiteNamePrompt prompts for the generic site name.
func (p *GenericProvider) SiteNamePrompt() error {
	return nil
}

func (p *GenericProvider) GetSites() ([]string, error) {
	return nil, nil
}

// GetBackup will create and download a backup
// Valid values for backupType are "database" or "files".
// returns fileURL, importPath, error
func (p *GenericProvider) GetBackup(backupType, environment string) (string, string, error) {
	var err error
	var filePath string
	if backupType != "database" && backupType != "files" {
		return "", "", fmt.Errorf("could not get backup: %s is not a valid backup type", backupType)
	}

	// Set the import path blank to use the root of the archive by default.
	importPath := ""

	p.prepDownloadDir()

	switch backupType {
	case "database":
		filePath, err = p.getDatabaseBackup()
	case "files":
		filePath, err = p.getFilesBackup()
	default:
		return "", "", fmt.Errorf("could not get backup: %s is not a valid backup type", backupType)
	}
	if err != nil {
		return "", "", err
	}

	return filePath, importPath, nil
}

// prepDownloadDir ensures the download cache directories are created and writeable.
func (p *GenericProvider) prepDownloadDir() {
	destDir := p.getDownloadDir()
	filesDir := filepath.Join(destDir, "files")
	_ = os.RemoveAll(filesDir)
	err := os.MkdirAll(filesDir, 0755)
	util.CheckErr(err)
}

func (p *GenericProvider) getDownloadDir() string {
	destDir := p.app.GetConfigPath(".generic-downloads")
	return destDir
}

func (p *GenericProvider) getFilesBackup() (filename string, error error) {

	destDir := filepath.Join(p.getDownloadDir(), "files")
	_ = os.RemoveAll(destDir)
	_ = os.MkdirAll(destDir, 0755)

	// Create a files backup first so we can pull
	if os.Getenv("DDEV_DEBUG") != "" {
		output.UserOut.Printf("backup files %s", p.app.Name)
	}

	s := p.app.Providers[p.app.Provider].FilesPullCommand.Service
	if s == "" {
		s = "web"
	}

	err := p.app.ExecOnHostOrService(s, p.app.Providers[p.app.Provider].FilesPullCommand.Command)
	if err != nil {
		util.Failed("Failed to exec %s on %s", p.app.Providers[p.app.Provider].DBPullCommand.Command, s)
	}

	return filepath.Join(p.getDownloadDir(), "files"), nil
}

// getDatabaseBackup retrieves database using `generic backup database`, then
// describe until it appears, then download it.
func (p *GenericProvider) getDatabaseBackup() (filename string, error error) {
	_ = os.RemoveAll(p.getDownloadDir())
	_ = os.Mkdir(p.getDownloadDir(), 0755)

	if p.app.Providers[p.app.Provider].DBPullCommand.Command == "" {
		util.Warning("No DBPullCommand is defined for provider %s", p.app.Provider)
		return "", nil
	}
	// First, kick off the database backup
	if os.Getenv("DDEV_DEBUG") != "" {
		output.UserOut.Print("generic backup database")
	}

	s := p.app.Providers[p.app.Provider].DBPullCommand.Service
	if s == "" {
		s = "web"
	}
	err := p.app.ExecOnHostOrService(s, p.app.Providers[p.app.Provider].DBPullCommand.Command)
	if err != nil {
		util.Failed("Failed to exec %s on %s", p.app.Providers[p.app.Provider].DBPullCommand.Command, s)
	}
	return filepath.Join(p.getDownloadDir(), "db.sql.gz"), nil
}

// Write the generic provider configuration to a specified location on disk.
func (p *GenericProvider) Write(configPath string) error {
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

// Read generic provider configuration from a specified location on disk.
func (p *GenericProvider) Read(configPath string) error {
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

// Validate ensures that the current configuration is valid (i.e. the configured pantheon site/environment exists)
func (p *GenericProvider) Validate() error {
	return nil
}
