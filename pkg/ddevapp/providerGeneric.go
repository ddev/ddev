package ddevapp

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"fmt"

	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"

	"gopkg.in/yaml.v2"
)

// ProviderCommand defines the shell command to be run for one of the commands (db pull, etc.)
type ProviderCommand struct {
	Command string `yaml:"command"`
	Service string `yaml:"service,omitempty"`
}

// ProviderInfo defines the provider
type ProviderInfo struct {
	ProjectID        string          `yaml:"project_id"`
	EnvironmentName  string          `yaml:"environment_name"`
	AuthCommand      ProviderCommand `yaml:"auth_command"`
	DBPullCommand    ProviderCommand `yaml:"db_pull_command"`
	FilesPullCommand ProviderCommand `yaml:"files_pull_command"`
	CodePullCommand  ProviderCommand `yaml:"code_pull_command,omitempty"`
}

// GenericProvider provides generic-specific import functionality.
type GenericProvider struct {
	ProviderType string   `yaml:"provider"`
	app          *DdevApp `yaml:"-"`
	ProviderInfo `yaml:"providers"`
}

// Init handles loading data from saved config.
func (p *GenericProvider) Init(app *DdevApp) error {
	p.app = app
	configPath := app.GetConfigPath(filepath.Join("providers", app.Provider+".yaml"))
	if !fileutil.FileExists(configPath) {
		return fmt.Errorf("no configuration exists for %s provider - it should be at %s", app.Provider, configPath)
	}
	err := p.Read(configPath)
	if err != nil {
		return err
	}

	p.ProviderType = app.Provider
	app.ProviderInstance = p
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

	if p.AuthCommand.Command != "" {
		err := p.app.ExecOnHostOrService(p.AuthCommand.Service, p.AuthCommand.Command)
		if err != nil {
			return "", "", err
		}
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
	_ = os.RemoveAll(destDir)
	err := os.MkdirAll(filesDir, 0755)
	util.CheckErr(err)
}

func (p *GenericProvider) getDownloadDir() string {
	destDir := p.app.GetConfigPath(".downloads")
	return destDir
}

func (p *GenericProvider) getFilesBackup() (filename string, error error) {

	destDir := filepath.Join(p.getDownloadDir(), "files")
	_ = os.RemoveAll(destDir)
	_ = os.MkdirAll(destDir, 0755)

	s := p.FilesPullCommand.Service
	if s == "" {
		s = "web"
	}

	err := p.app.ExecOnHostOrService(s, p.FilesPullCommand.Command)
	if err != nil {
		util.Failed("Failed to exec %s on %s: %v", p.DBPullCommand.Command, s, err)
	}

	return filepath.Join(p.getDownloadDir(), "files"), nil
}

// getDatabaseBackup retrieves database using `generic backup database`, then
// describe until it appears, then download it.
func (p *GenericProvider) getDatabaseBackup() (filename string, error error) {
	_ = os.RemoveAll(p.getDownloadDir())
	_ = os.Mkdir(p.getDownloadDir(), 0755)

	if p.DBPullCommand.Command == "" {
		util.Warning("No DBPullCommand is defined for provider %s", p.app.Provider)
		return "", nil
	}

	s := p.DBPullCommand.Service
	if s == "" {
		s = "web"
	}
	err := p.app.ExecOnHostOrService(s, p.DBPullCommand.Command)
	if err != nil {
		util.Failed("Failed to exec %s on %s: %v", p.DBPullCommand.Command, s, err)
	}
	return filepath.Join(p.getDownloadDir(), "db.sql.gz"), nil
}

// Write the generic provider configuration to a specified location on disk.
func (p *GenericProvider) Write(configPath string) error {
	return nil
}

// Read generic provider configuration from a specified location on disk.
func (p *GenericProvider) Read(configPath string) error {
	source, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}

	// Read config values from file.
	err = yaml.Unmarshal(source, &p.ProviderInfo)
	if err != nil {
		return err
	}

	return nil
}

// Validate ensures that the current configuration is valid (i.e. the configured pantheon site/environment exists)
func (p *GenericProvider) Validate() error {
	return nil
}
