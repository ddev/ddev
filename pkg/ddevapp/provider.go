package ddevapp

import (
	"github.com/drud/ddev/pkg/output"
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
	EnvironmentVariables map[string]string `yaml:"environment_variables"`
	AuthCommand          ProviderCommand   `yaml:"auth_command"`
	DBPullCommand        ProviderCommand   `yaml:"db_pull_command"`
	DBImportCommand      ProviderCommand   `yaml:"db_import_command"`
	FilesPullCommand     ProviderCommand   `yaml:"files_pull_command"`
	FilesImportCommand   ProviderCommand   `yaml:"files_import_command"`
	CodePullCommand      ProviderCommand   `yaml:"code_pull_command,omitempty"`
	DBPushCommand        ProviderCommand   `yaml:"db_push_command"`
	FilesPushCommand     ProviderCommand   `yaml:"files_push_command"`
}

// Provider provides generic-specific import functionality.
type Provider struct {
	ProviderType string   `yaml:"provider"`
	app          *DdevApp `yaml:"-"`
	ProviderInfo `yaml:"providers"`
}

// Init handles loading data from saved config.
func (p *Provider) Init(pType string, app *DdevApp) error {
	p.app = app
	configPath := app.GetConfigPath(filepath.Join("providers", pType+".yaml"))
	if !fileutil.FileExists(configPath) {
		return fmt.Errorf("no configuration exists for %s provider - it should be at %s", pType, configPath)
	}
	err := p.Read(configPath)
	if err != nil {
		return err
	}

	p.ProviderType = pType
	app.ProviderInstance = p
	return nil
}

// Pull performs an import of db and files
func (app *DdevApp) Pull(provider *Provider, skipDbArg bool, skipFilesArg bool, skipImportArg bool) error {
	var err error
	err = app.ProcessHooks("pre-pull")
	if err != nil {
		return fmt.Errorf("Failed to process pre-pull hooks: %v", err)
	}

	if app.SiteStatus() != SiteRunning {
		util.Warning("Project is not currently running. Starting project before performing pull.")
		err = app.Start()
		if err != nil {
			return err
		}
	}

	if provider.AuthCommand.Command != "" {
		output.UserOut.Print("Authenticating...")
		err = provider.app.ExecOnHostOrService(provider.AuthCommand.Service, provider.injectedEnvironment()+"; "+provider.AuthCommand.Command)
		if err != nil {
			return err
		}
	}

	if skipDbArg {
		output.UserOut.Println("Skipping database pull.")
	} else {
		output.UserOut.Println("Obtaining database...")
		fileLocation, importPath, err := provider.GetBackup("database")
		if err != nil {
			return err
		}
		err = app.MutagenSyncFlush()
		if err != nil {
			return err
		}

		if skipImportArg {
			output.UserOut.Println("Skipping database import.")
		} else {
			err = app.MutagenSyncFlush()
			if err != nil {
				return err
			}
			output.UserOut.Println("Importing database...")
			err = provider.importDatabaseBackup(fileLocation, importPath)

			if err != nil {
				return err
			}
		}
	}

	if skipFilesArg {
		output.UserOut.Println("Skipping files pull.")
	} else {
		output.UserOut.Println("Obtaining files...")
		fileLocation, importPath, err := provider.GetBackup("files")
		if err != nil {
			return err
		}

		err = app.MutagenSyncFlush()
		if err != nil {
			return err
		}

		if skipImportArg {
			output.UserOut.Println("Skipping files import.")
		} else {
			output.UserOut.Println("Importing files...")
			err = provider.importFilesBackup(fileLocation, importPath)
			if err != nil {
				return err
			}
		}
	}
	err = app.ProcessHooks("post-pull")
	if err != nil {
		return fmt.Errorf("Failed to process post-pull hooks: %v", err)
	}

	return nil
}

// Push pushes db and files up to upstream hosting provider
func (app *DdevApp) Push(provider *Provider, skipDbArg bool, skipFilesArg bool) error {
	var err error
	err = app.ProcessHooks("pre-push")
	if err != nil {
		return fmt.Errorf("Failed to process pre-push hooks: %v", err)
	}

	if app.SiteStatus() != SiteRunning {
		util.Warning("Project is not currently running. Starting project before performing push.")
		err = app.Start()
		if err != nil {
			return err
		}
	}

	if provider.AuthCommand.Command != "" {
		output.UserOut.Print("Authenticating...")
		err := provider.app.ExecOnHostOrService(provider.AuthCommand.Service, provider.injectedEnvironment()+"; "+provider.AuthCommand.Command)
		if err != nil {
			return err
		}
	}

	if skipDbArg {
		output.UserOut.Println("Skipping database push.")
	} else {
		output.UserOut.Println("Uploading database...")
		err = provider.UploadDB()
		if err != nil {
			return err
		}

		output.UserOut.Printf("Database uploaded")
	}

	if skipFilesArg {
		output.UserOut.Println("Skipping files push.")
	} else {
		output.UserOut.Println("Uploading files...")
		err = provider.UploadFiles()
		if err != nil {
			return err
		}

		output.UserOut.Printf("Files uploaded")
	}
	err = app.ProcessHooks("post-push")
	if err != nil {
		return fmt.Errorf("Failed to process post-push hooks: %v", err)
	}

	return nil
}

// GetBackup will create and download a backup
// Valid values for backupType are "database" or "files".
// returns fileURL, importPath, error
func (p *Provider) GetBackup(backupType string) (string, string, error) {
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

// UploadDB is used by Push to push the database to hosting provider
func (p *Provider) UploadDB() error {
	_ = os.RemoveAll(p.getDownloadDir())
	_ = os.Mkdir(p.getDownloadDir(), 0755)

	if p.DBPushCommand.Command == "" {
		util.Warning("No DBPushCommand is defined for provider %s", p.ProviderType)
		return nil
	}

	err := p.app.ExportDB(p.app.GetConfigPath(".downloads/db.sql.gz"), "gzip", "")
	if err != nil {
		return err
	}
	err = p.app.MutagenSyncFlush()
	if err != nil {
		return err
	}

	s := p.DBPushCommand.Service
	if s == "" {
		s = "web"
	}
	err = p.app.ExecOnHostOrService(s, p.injectedEnvironment()+"; "+p.DBPushCommand.Command)
	if err != nil {
		return fmt.Errorf("Failed to exec %s on %s: %v", p.DBPushCommand.Command, s, err)
	}
	return nil
}

// UploadFiles is used by Push to push the user-generated files to the hosting provider
func (p *Provider) UploadFiles() error {
	_ = os.RemoveAll(p.getDownloadDir())
	_ = os.Mkdir(p.getDownloadDir(), 0755)

	if p.FilesPushCommand.Command == "" {
		util.Warning("No FilesPushCommand is defined for provider %s", p.ProviderType)
		return nil
	}

	s := p.FilesPushCommand.Service
	if s == "" {
		s = "web"
	}
	err := p.app.ExecOnHostOrService(s, p.injectedEnvironment()+"; "+p.FilesPushCommand.Command)
	if err != nil {
		return fmt.Errorf("Failed to exec %s on %s: %v", p.FilesPushCommand.Command, s, err)
	}
	return nil
}

// prepDownloadDir ensures the download cache directories are created and writeable.
func (p *Provider) prepDownloadDir() {
	destDir := p.getDownloadDir()
	filesDir := filepath.Join(destDir, "files")
	_ = os.RemoveAll(destDir)
	err := os.MkdirAll(filesDir, 0755)
	util.CheckErr(err)
}

func (p *Provider) getDownloadDir() string {
	destDir := p.app.GetConfigPath(".downloads")
	return destDir
}

func (p *Provider) getFilesBackup() (filename string, error error) {

	destDir := filepath.Join(p.getDownloadDir(), "files")
	_ = os.RemoveAll(destDir)
	_ = os.MkdirAll(destDir, 0755)

	if p.FilesPullCommand.Command == "" {
		util.Warning("No FilesPullCommand is defined for provider %s", p.ProviderType)
		return "", nil
	}
	s := p.FilesPullCommand.Service
	if s == "" {
		s = "web"
	}

	err := p.app.ExecOnHostOrService(s, p.injectedEnvironment()+"; "+p.FilesPullCommand.Command)
	if err != nil {
		return "", fmt.Errorf("Failed to exec %s on %s: %v", p.FilesPullCommand.Command, s, err)
	}

	return filepath.Join(p.getDownloadDir(), "files"), nil
}

// getDatabaseBackup retrieves database using `generic backup database`, then
// describe until it appears, then download it.
func (p *Provider) getDatabaseBackup() (filename string, error error) {
	_ = os.RemoveAll(p.getDownloadDir())
	_ = os.Mkdir(p.getDownloadDir(), 0755)

	if p.DBPullCommand.Command == "" {
		util.Warning("No DBPullCommand is defined for provider")
		return "", nil
	}

	s := p.DBPullCommand.Service
	if s == "" {
		s = "web"
	}
	err := p.app.MutagenSyncFlush()
	if err != nil {
		return "", err
	}
	err = p.app.ExecOnHostOrService(s, p.injectedEnvironment()+"; "+p.DBPullCommand.Command)
	if err != nil {
		return "", fmt.Errorf("Failed to exec %s on %s: %v", p.DBPullCommand.Command, s, err)
	}
	return filepath.Join(p.getDownloadDir(), "db.sql.gz"), nil
}

// importDatabaseBackup will import a downloaded database
// If a custom importer is provided, that will be used, otherwise
// the default is app.ImportDB()
func (p *Provider) importDatabaseBackup(fileLocation string, importPath string) error {
	var err error
	if p.DBImportCommand.Command == "" {
		err = p.app.ImportDB(fileLocation, importPath, true, false, "db")
	} else {
		s := p.DBImportCommand.Service
		if s == "" {
			s = "web"
		}
		output.UserOut.Printf("Importing database via custom db_import_command")
		err = p.app.ExecOnHostOrService(s, p.injectedEnvironment()+"; "+p.DBImportCommand.Command)
	}
	return err
}

// importFilesBackup will import a downloaded files tarball or directory
// If a custom importer is provided, that will be used, otherwise
// the default is app.ImportFiles()
func (p *Provider) importFilesBackup(fileLocation string, importPath string) error {
	var err error
	if p.FilesImportCommand.Command == "" {
		err = p.app.ImportFiles(fileLocation, importPath)
	} else {
		s := p.FilesImportCommand.Service
		if s == "" {
			s = "web"
		}
		output.UserOut.Printf("Importing files via custom files_import_command")
		err = p.app.ExecOnHostOrService(s, p.injectedEnvironment()+"; "+p.FilesImportCommand.Command)
	}
	return err
}

// Write the generic provider configuration to a specified location on disk.
func (p *Provider) Write(configPath string) error {
	return nil
}

// Read generic provider configuration from a specified location on disk.
func (p *Provider) Read(configPath string) error {
	source, err := os.ReadFile(configPath)
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
func (p *Provider) Validate() error {
	return nil
}

// injectedEnvironment() returns a string with environment variables that should be injected
// before a command.
func (p *Provider) injectedEnvironment() string {
	s := "true"
	if len(p.EnvironmentVariables) > 0 {
		s = "export "
		for k, v := range p.EnvironmentVariables {
			s = s + fmt.Sprintf(" %s=%s ", k, v)
		}
	}
	return s
}
