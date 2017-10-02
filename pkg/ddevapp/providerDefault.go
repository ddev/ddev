package ddevapp

import "errors"
import "os"

// DefaultProvider provides a no-op for the provider plugin interface methods.
type DefaultProvider struct{}

// Init provides a no-op for the Init operation.
func (p *DefaultProvider) Init(config *Config) error {
	return nil
}

// ValidateField provides a no-op for the ValidateField operation.
func (p *DefaultProvider) ValidateField(field, value string) error {
	return nil
}

// Config provides a no-op for the Config operation.
func (p *DefaultProvider) PromptForConfig() error {
	return nil
}

// Write provides a no-op for the Write operation.
func (p *DefaultProvider) Write(configPath string) error {
	// Check if the file exists and can be read and just return nil if it doesn't.
	_, err := os.Stat(configPath)
	if err != nil {
		return nil
	}

	// Attempt to remove any import config for another provider which may be present.
	return os.Remove(configPath)
}

// Read provides a no-op for the Read operation.
func (p *DefaultProvider) Read(configPath string) error {
	return nil
}

// Validate always returns an error from the default provider, as we have no provider to import data from.
func (p *DefaultProvider) Validate() error {
	return errors.New("could not perform import because there is no configured provider for this application. please see `ddev config` documentation")
}

// GetBackup provides a no-op for the GetBackup operation.
func (p *DefaultProvider) GetBackup(backupType string) (fileLocation string, importPath string, err error) {
	return "", "", nil
}
