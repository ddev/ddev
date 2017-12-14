package ddevapp

import "os"

// DefaultProvider provides a no-op for the provider plugin interface methods.
type DefaultProvider struct{}

// Init provides a no-op for the Init operation.
func (p *DefaultProvider) Init(app *DdevApp) error {
	return nil
}

// ValidateField provides a no-op for the ValidateField operation.
func (p *DefaultProvider) ValidateField(field, value string) error {
	return nil
}

// PromptForConfig provides a no-op for the Config operation.
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

// Validate always succeeds, because the default provider is a fine provider.
func (p *DefaultProvider) Validate() error {
	return nil
}

// GetBackup provides a no-op for the GetBackup operation.
func (p *DefaultProvider) GetBackup(backupType string) (fileLocation string, importPath string, err error) {
	return "", "", nil
}
