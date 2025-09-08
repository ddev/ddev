package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
)

func TestGetConfigTool(t *testing.T) {
	// Create a temporary project directory
	tmpDir, err := os.MkdirTemp("", "test-ddev-config")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple .ddev/config.yaml
	ddevDir := filepath.Join(tmpDir, ".ddev")
	err = os.MkdirAll(ddevDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .ddev dir: %v", err)
	}

	configContent := `name: test-project
type: php
docroot: web
php_version: "8.1"
webserver_type: nginx-fpm
`

	configPath := filepath.Join(ddevDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config.yaml: %v", err)
	}

	// Test getting configuration
	input := GetConfigInput{
		AppRoot: tmpDir,
	}

	ctx := context.Background()
	result, output, err := handleGetConfig(ctx, nil, input)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !output.Success {
		t.Errorf("Expected success=true, got success=%v, message=%s", output.Success, output.Message)
	}

	if output.ProjectName == "" {
		t.Error("Expected non-empty project name")
	}

	if output.ConfigPath != configPath {
		t.Errorf("Expected config path %s, got %s", configPath, output.ConfigPath)
	}

	if output.Config == nil {
		t.Error("Expected non-nil config")
	}

	if result.IsError {
		t.Error("Expected result.IsError=false")
	}
}

func TestUpdateConfigTool(t *testing.T) {
	// Initialize global config to prevent panic during WriteConfig
	err := globalconfig.ReadGlobalConfig()
	if err != nil {
		t.Logf("Warning: Failed to read global config, using defaults: %v", err)
	}

	// Ensure critical global maps are initialized to prevent panics
	if globalconfig.DdevGlobalConfig.ProjectList == nil {
		globalconfig.DdevGlobalConfig.ProjectList = make(map[string]*globalconfig.ProjectInfo)
	}
	if globalconfig.DdevProjectList == nil {
		globalconfig.DdevProjectList = make(map[string]*globalconfig.ProjectInfo)
	}

	// Create a temporary project directory
	tmpDir, err := os.MkdirTemp("", "test-ddev-update-config")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple .ddev/config.yaml
	ddevDir := filepath.Join(tmpDir, ".ddev")
	err = os.MkdirAll(ddevDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .ddev dir: %v", err)
	}

	configContent := `name: test-project
type: php
docroot: web
php_version: "8.1"
webserver_type: nginx-fpm
`

	configPath := filepath.Join(ddevDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config.yaml: %v", err)
	}

	// Test validation only
	t.Run("ValidateOnly", func(t *testing.T) {
		input := UpdateConfigInput{
			AppRoot: tmpDir,
			Config: map[string]any{
				"php_version": "8.2",
			},
			ValidateOnly: true,
			CreateBackup: false,
		}

		ctx := context.Background()
		result, output, err := handleUpdateConfig(ctx, nil, input)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !output.Success {
			t.Errorf("Expected success=true, got success=%v, message=%s", output.Success, output.Message)
		}

		if !output.Validated {
			t.Error("Expected validated=true")
		}

		if output.Applied {
			t.Error("Expected applied=false for validate-only operation")
		}

		if result.IsError {
			t.Error("Expected result.IsError=false")
		}
	})

	// Test actual config update
	t.Run("ActualUpdate", func(t *testing.T) {
		input := UpdateConfigInput{
			AppRoot: tmpDir,
			Config: map[string]any{
				"php_version":    "8.2",
				"xdebug_enabled": true,
			},
			ValidateOnly: false,
			CreateBackup: true,
		}

		ctx := context.Background()
		result, output, err := handleUpdateConfig(ctx, nil, input)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !output.Success {
			t.Errorf("Expected success=true, got success=%v, message=%s", output.Success, output.Message)
		}

		if !output.Validated {
			t.Error("Expected validated=true")
		}

		if !output.Applied {
			t.Error("Expected applied=true")
		}

		if output.BackupPath == "" {
			t.Error("Expected non-empty backup path")
		}

		// Check that backup file exists
		if _, err := os.Stat(output.BackupPath); os.IsNotExist(err) {
			t.Errorf("Expected backup file to exist at %s", output.BackupPath)
		}

		if result.IsError {
			t.Error("Expected result.IsError=false")
		}

		// Verify the configuration was actually updated
		app, err := ddevapp.NewApp(tmpDir, true)
		if err != nil {
			t.Fatalf("Failed to load app after update: %v", err)
		}

		if app.PHPVersion != "8.2" {
			t.Errorf("Expected PHP version 8.2, got %s", app.PHPVersion)
		}

		if !app.XdebugEnabled {
			t.Error("Expected XdebugEnabled=true")
		}
	})

	// Test invalid configuration
	t.Run("InvalidConfig", func(t *testing.T) {
		input := UpdateConfigInput{
			AppRoot: tmpDir,
			Config: map[string]any{
				"php_version": "invalid-version",
			},
			ValidateOnly: true,
			CreateBackup: false,
		}

		ctx := context.Background()
		result, output, err := handleUpdateConfig(ctx, nil, input)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Should fail validation
		if output.Success {
			t.Error("Expected success=false for invalid config")
		}

		if output.Validated {
			t.Error("Expected validated=false for invalid config")
		}

		if len(output.Errors) == 0 {
			t.Error("Expected validation errors")
		}

		if result.IsError != true {
			t.Error("Expected result.IsError=true for invalid config")
		}
	})
}

func TestConfigBackup(t *testing.T) {
	// Create a temporary config file
	tmpDir, err := os.MkdirTemp("", "test-config-backup")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := "name: test\ntype: php\n"

	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test backup creation
	backupPath, err := createConfigBackup(configPath)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if backupPath == "" {
		t.Error("Expected non-empty backup path")
	}

	// Check that backup file exists
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Errorf("Failed to read backup file: %v", err)
	}

	if string(backupContent) != configContent {
		t.Errorf("Expected backup content '%s', got '%s'", configContent, string(backupContent))
	}
}

func TestSetConfigField(t *testing.T) {
	app := &ddevapp.DdevApp{}

	tests := []struct {
		name        string
		fieldName   string
		value       any
		expectError bool
		checkFunc   func(*ddevapp.DdevApp) bool
	}{
		{
			name:      "Set string field",
			fieldName: "name",
			value:     "test-project",
			checkFunc: func(a *ddevapp.DdevApp) bool { return a.Name == "test-project" },
		},
		{
			name:      "Set PHP version",
			fieldName: "php_version",
			value:     "8.2",
			checkFunc: func(a *ddevapp.DdevApp) bool { return a.PHPVersion == "8.2" },
		},
		{
			name:      "Set boolean field",
			fieldName: "xdebug_enabled",
			value:     true,
			checkFunc: func(a *ddevapp.DdevApp) bool { return a.XdebugEnabled == true },
		},
		{
			name:      "Set string array field",
			fieldName: "additional_hostnames",
			value:     []string{"example.com", "test.com"},
			checkFunc: func(a *ddevapp.DdevApp) bool {
				return len(a.AdditionalHostnames) == 2 &&
					a.AdditionalHostnames[0] == "example.com" &&
					a.AdditionalHostnames[1] == "test.com"
			},
		},
		{
			name:      "Set interface array field",
			fieldName: "additional_fqdns",
			value:     []interface{}{"example.ddev.site", "test.ddev.site"},
			checkFunc: func(a *ddevapp.DdevApp) bool {
				return len(a.AdditionalFQDNs) == 2 &&
					a.AdditionalFQDNs[0] == "example.ddev.site" &&
					a.AdditionalFQDNs[1] == "test.ddev.site"
			},
		},
		{
			name:        "Invalid field name",
			fieldName:   "nonexistent_field",
			value:       "value",
			expectError: true,
		},
		{
			name:        "Invalid type for string field",
			fieldName:   "php_version",
			value:       123,
			expectError: true,
		},
		{
			name:        "Invalid type for boolean field",
			fieldName:   "xdebug_enabled",
			value:       "not-a-boolean",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset app for each test
			app = &ddevapp.DdevApp{}

			err := setConfigField(app, tt.fieldName, tt.value)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, got none", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s, got: %v", tt.name, err)
				}

				if tt.checkFunc != nil && !tt.checkFunc(app) {
					t.Errorf("Field value check failed for %s", tt.name)
				}
			}
		})
	}
}

func TestApplyConfigChanges(t *testing.T) {
	app := &ddevapp.DdevApp{}

	changes := map[string]any{
		"name":                 "test-app",
		"php_version":          "8.2",
		"type":                 "php",
		"xdebug_enabled":       true,
		"additional_hostnames": []string{"example.com"},
	}

	err := applyConfigChanges(app, changes)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify all changes were applied
	if app.Name != "test-app" {
		t.Errorf("Expected name='test-app', got '%s'", app.Name)
	}

	if app.PHPVersion != "8.2" {
		t.Errorf("Expected PHPVersion='8.2', got '%s'", app.PHPVersion)
	}

	if app.Type != "php" {
		t.Errorf("Expected Type='php', got '%s'", app.Type)
	}

	if !app.XdebugEnabled {
		t.Error("Expected XdebugEnabled=true")
	}

	if len(app.AdditionalHostnames) != 1 || app.AdditionalHostnames[0] != "example.com" {
		t.Errorf("Expected AdditionalHostnames=['example.com'], got %v", app.AdditionalHostnames)
	}
}
