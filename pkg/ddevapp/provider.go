package ddevapp

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"gopkg.in/yaml.v3"
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

	if p.EnvironmentVariables == nil {
		p.EnvironmentVariables = map[string]string{}
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

	status, _ := app.SiteStatus()
	if status != SiteRunning {
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
		output.UserOut.Println("Obtaining databases...")
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
			output.UserOut.Printf("Importing databases %v\n", fileLocation)
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
		files, _, err := provider.GetBackup("files")
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
			f := ""
			if files != nil && len(files) > 0 {
				f = files[0]
			}
			err = provider.doFilesImport(f, "")
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

	status, _ := app.SiteStatus()
	if status != SiteRunning {
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

// GetBackup will create and download a set of backups
// Valid values for backupType are "database" or "files".
// returns []fileURL, []importPath, error
func (p *Provider) GetBackup(backupType string) ([]string, []string, error) {
	var err error
	var fileNames []string
	if backupType != "database" && backupType != "files" {
		return nil, nil, fmt.Errorf("could not get backup: %s is not a valid backup type", backupType)
	}

	p.prepDownloadDir()

	switch backupType {
	case "database":
		fileNames, err = p.getDatabaseBackups()
	case "files":
		fileNames, err = p.doFilesPullCommand()
	default:
		return nil, nil, fmt.Errorf("could not get backup: %s is not a valid backup type", backupType)
	}
	if err != nil {
		return nil, nil, err
	}

	importPaths := make([]string, len(fileNames))
	// We don't use importPaths for the providers
	for i := range fileNames {
		importPaths[i] = ""
	}

	return fileNames, importPaths, nil
}

// UploadDB is used by Push to push the database to hosting provider
func (p *Provider) UploadDB() error {
	_ = os.RemoveAll(p.getDownloadDir())
	_ = os.Mkdir(p.getDownloadDir(), 0755)

	if p.DBPushCommand.Command == "" {
		util.Warning("No DBPushCommand is defined for provider '%s'", p.ProviderType)
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
		util.Warning("No FilesPushCommand is defined for provider '%s'", p.ProviderType)
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

func (p *Provider) doFilesPullCommand() (filename []string, error error) {
	destDir := filepath.Join(p.getDownloadDir(), "files")
	_ = os.RemoveAll(destDir)
	_ = os.MkdirAll(destDir, 0755)

	if p.FilesPullCommand.Command == "" {
		util.Warning("No FilesPullCommand is defined for provider '%s'", p.ProviderType)
		return nil, nil
	}
	s := p.FilesPullCommand.Service
	if s == "" {
		s = "web"
	}

	err := p.app.ExecOnHostOrService(s, p.injectedEnvironment()+"; "+p.FilesPullCommand.Command)
	if err != nil {
		return nil, fmt.Errorf("Failed to exec %s on %s: %v", p.FilesPullCommand.Command, s, err)
	}

	return []string{filepath.Join(p.getDownloadDir(), "files")}, nil
}

// getDatabaseBackups retrieves database using `generic backup database`, then
// describe until it appears, then download it.
func (p *Provider) getDatabaseBackups() (filename []string, error error) {
	_ = os.RemoveAll(p.getDownloadDir())
	_ = os.Mkdir(p.getDownloadDir(), 0755)

	if p.DBPullCommand.Command == "" {
		util.Warning("No DBPullCommand is defined for provider '%s'", p.ProviderType)
		return nil, nil
	}

	s := p.DBPullCommand.Service
	if s == "" {
		s = "web"
	}
	err := p.app.MutagenSyncFlush()
	if err != nil {
		return nil, err
	}
	err = p.app.ExecOnHostOrService(s, p.injectedEnvironment()+"; "+p.DBPullCommand.Command)
	if err != nil {
		return nil, fmt.Errorf("Failed to exec %s on %s: %v", p.DBPullCommand.Command, s, err)
	}
	err = p.app.MutagenSyncFlush()
	if err != nil {
		return nil, err
	}

	sqlTarballs, err := fileutil.ListFilesInDirFullPath(p.getDownloadDir())
	if err != nil || sqlTarballs == nil {
		return nil, fmt.Errorf("failed to find downloaded files in %s: %v", p.getDownloadDir(), err)
	}
	return sqlTarballs, nil
}

// importDatabaseBackup will import a slice of downloaded databases
// If a custom importer is provided, that will be used, otherwise
// the default is app.ImportDB()
func (p *Provider) importDatabaseBackup(fileLocation []string, importPath []string) error {
	var err error
	if p.DBImportCommand.Command == "" {
		for i, loc := range fileLocation {
			// The database name used will be basename of the file.
			// For example. `db.sql.gz` will go into the database named 'db'
			// xxx.sql will go into database named 'xxx';
			b := path.Base(loc)
			n := strings.Split(b, ".")
			dbName := n[0]
			err = p.app.ImportDB(loc, importPath[i], true, false, dbName)
		}
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

// doFilesImport will import previously downloaded files tarball or directory
// If a custom importer (FileImportCommand) is provided, that will be used, otherwise
// the default is app.ImportFiles()
// FilesImportCommand may also optionally take on the job of downloading the files.
func (p *Provider) doFilesImport(fileLocation string, importPath string) error {
	var err error
	if p.FilesImportCommand.Command == "" {
		err = p.app.ImportFiles("", fileLocation, importPath)
	} else {
		s := p.FilesImportCommand.Service
		if s == "" {
			s = "web"
		}
		output.UserOut.Printf("Importing files via custom files_import_command...")
		err = p.app.ExecOnHostOrService(s, p.injectedEnvironment()+"; "+p.FilesImportCommand.Command)
	}
	return err
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
		s = "export"
		for k, v := range p.EnvironmentVariables {
			v = strings.Replace(v, " ", `\ `, -1)
			s = s + fmt.Sprintf(" %s=%s ", k, v)
		}
	}
	return s
}
