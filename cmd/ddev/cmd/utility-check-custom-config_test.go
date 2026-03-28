package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

// TestUtilityCheckCustomConfigCmd tests ddev debug check-custom-config
func TestUtilityCheckCustomConfigCmd(t *testing.T) {
	// Create isolated global DDEV directory for testing
	origDir, _ := os.Getwd()
	tmpXdgConfigHomeDir := testcommon.CopyGlobalDdevDir(t)
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		testcommon.ResetGlobalDdevDir(t, tmpXdgConfigHomeDir)
	})

	// Create a temporary directory and switch to it.
	tmpdir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()

	// Use tmpdir name as project name
	projectName := filepath.Base(tmpdir)

	// Create a basic config
	args := []string{
		"config",
		"--docroot", ".",
		"--project-name", projectName,
		"--project-type", "php",
	}

	_, err := exec.RunCommand(DdevBin, args)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = exec.RunCommand(DdevBin, []string{"delete", "-Oy", projectName})
	})

	// Test with no custom config
	t.Run("no custom config", func(t *testing.T) {
		// Default mode: no output when nothing to warn about
		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.NotContains(t, out, "Custom configuration detected")

		// --all mode: explicit success message
		out, err = exec.RunCommand(DdevBin, []string{"utility", "check-custom-config", "--all"})
		require.NoError(t, err)
		require.Contains(t, out, "No custom configuration detected in project '"+projectName+"'.")
	})

	// GLOBAL CHECKS (matching order in CheckCustomConfig)

	// Test global router-compose
	t.Run("global router-compose", func(t *testing.T) {
		globalDdevDir := globalconfig.GetGlobalDdevDir()
		customRouterCompose := filepath.Join(globalDdevDir, "router-compose.custom.yaml")
		err := os.WriteFile(customRouterCompose, []byte("# Custom router compose\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(customRouterCompose)
		})

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Router (global)")
		require.Contains(t, out, "router-compose.custom.yaml")
	})

	// Test global ssh-auth-compose
	t.Run("global ssh-auth-compose", func(t *testing.T) {
		globalDdevDir := globalconfig.GetGlobalDdevDir()
		customSSHCompose := filepath.Join(globalDdevDir, "ssh-auth-compose.custom.yaml")
		err := os.WriteFile(customSSHCompose, []byte("# Custom SSH auth compose\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(customSSHCompose)
		})

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "SSH agent (global)")
		require.Contains(t, out, "ssh-auth-compose.custom.yaml")
	})

	// Test global commands customization
	t.Run("global commands", func(t *testing.T) {
		globalCommandsDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands", "host")
		err := os.MkdirAll(globalCommandsDir, 0755)
		require.NoError(t, err)

		// Create a custom global command
		customCmd := filepath.Join(globalCommandsDir, "custom-global")
		err = os.WriteFile(customCmd, []byte("#!/bin/bash\necho 'custom global command'\n"), 0755)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(customCmd)
		})

		// Run check-custom-config
		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Commands (global)")
		require.Contains(t, out, "custom-global")
	})

	// Test global homeadditions customization
	t.Run("global homeadditions", func(t *testing.T) {
		globalHomeadditionsDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "homeadditions")
		err := os.MkdirAll(globalHomeadditionsDir, 0755)
		require.NoError(t, err)

		// Create a custom global homeaddition
		customFile := filepath.Join(globalHomeadditionsDir, "custom.sh")
		err = os.WriteFile(customFile, []byte("# Custom global bash config\nexport CUSTOM_VAR=1\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(customFile)
		})

		// Run check-custom-config
		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Home additions (global)")
		require.Contains(t, out, "custom.sh")
	})

	// Test global traefik static_config customization
	t.Run("global traefik static_config", func(t *testing.T) {
		traefikDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik")
		err := os.MkdirAll(traefikDir, 0755)
		require.NoError(t, err)

		// Create a custom traefik static config
		customConfig := filepath.Join(traefikDir, "static_config.custom.yaml")
		err = os.WriteFile(customConfig, []byte("# Custom traefik config\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(customConfig)
		})

		// Run check-custom-config
		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Router (global)")
		require.Contains(t, out, "static_config.custom.yaml")
	})

	// Test global traefik custom-global-config
	t.Run("global traefik custom-global-config", func(t *testing.T) {
		customGlobalConfigDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik", "custom-global-config")
		err := os.MkdirAll(customGlobalConfigDir, 0755)
		require.NoError(t, err)

		customFile := filepath.Join(customGlobalConfigDir, "custom.yaml")
		err = os.WriteFile(customFile, []byte("# Custom global traefik config\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(customFile)
		})

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Router (global)")
		require.Contains(t, out, "custom.yaml")
	})

	// PROJECT CHECKS (matching order in CheckCustomConfig)

	// Test apache config (conditional: only when webserver=apache)
	t.Run("apache config", func(t *testing.T) {
		// Configure to use apache webserver
		_, err := exec.RunCommand(DdevBin, []string{"config", "--webserver-type=apache-fpm"})
		require.NoError(t, err)
		t.Cleanup(func() {
			// Revert back to nginx
			_, _ = exec.RunCommand(DdevBin, []string{"config", "--webserver-type=" + nodeps.WebserverDefault})
		})

		ddevDir := filepath.Join(tmpdir, ".ddev")
		apacheDir := filepath.Join(ddevDir, "apache")
		err = os.MkdirAll(apacheDir, 0755)
		require.NoError(t, err)

		apacheFile := filepath.Join(apacheDir, "custom.conf")
		err = os.WriteFile(apacheFile, []byte("# Custom apache config\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(apacheFile)
		})

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Web server")
		require.Contains(t, out, "custom.conf")
	})

	// Test db-build Dockerfiles (all 6 patterns)
	t.Run("db-build Dockerfiles", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		dbBuildDir := filepath.Join(ddevDir, "db-build")
		err := os.MkdirAll(dbBuildDir, 0755)
		require.NoError(t, err)

		// Test all 6 valid Dockerfile patterns
		files := map[string]string{
			"Dockerfile":                "# Dockerfile\n",
			"Dockerfile.custom":         "# Dockerfile.custom\n",
			"pre.Dockerfile":            "# pre.Dockerfile\n",
			"pre.Dockerfile.custom":     "# pre.Dockerfile.custom\n",
			"prepend.Dockerfile":        "# prepend.Dockerfile\n",
			"prepend.Dockerfile.custom": "# prepend.Dockerfile.custom\n",
		}

		for name, content := range files {
			path := filepath.Join(dbBuildDir, name)
			err = os.WriteFile(path, []byte(content), 0644)
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = os.Remove(path)
			})
		}

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Database")
		// Check that all 6 files are detected
		for name := range files {
			require.Contains(t, out, name)
		}
	})

	// Test mutagen (conditional: only when mutagen is enabled)
	t.Run("mutagen config", func(t *testing.T) {
		// Configure to use mutagen
		_, err := exec.RunCommand(DdevBin, []string{"config", "--performance-mode=mutagen"})
		require.NoError(t, err)
		t.Cleanup(func() {
			// Revert back to default
			_, _ = exec.RunCommand(DdevBin, []string{"config", "--performance-mode=" + nodeps.PerformanceModeDefault})
		})

		ddevDir := filepath.Join(tmpdir, ".ddev")
		mutagenDir := filepath.Join(ddevDir, "mutagen")
		err = os.MkdirAll(mutagenDir, 0755)
		require.NoError(t, err)

		mutagenFile := filepath.Join(mutagenDir, "mutagen.yml")
		err = os.WriteFile(mutagenFile, []byte("# Custom mutagen config\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(mutagenFile)
		})

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Mutagen")
		require.Contains(t, out, "mutagen.yml")
	})

	// Test with custom PHP config
	t.Run("php config", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		phpDir := filepath.Join(ddevDir, "php")
		err := os.MkdirAll(phpDir, 0755)
		require.NoError(t, err)

		// Create a custom PHP config file
		customPHPFile := filepath.Join(phpDir, "custom.ini")
		err = os.WriteFile(customPHPFile, []byte("memory_limit = 512M\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(customPHPFile)
		})

		// Run check-custom-config
		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "PHP")
		require.Contains(t, out, "custom.ini")
	})

	// Test with silenced custom config file
	t.Run("silenced custom config", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		phpDir := filepath.Join(ddevDir, "php")

		// Create a silenced custom PHP config file
		silencedFile := filepath.Join(phpDir, "silenced.ini")
		err := os.WriteFile(silencedFile, []byte("#ddev-silent-no-warn\nmemory_limit = 256M\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(silencedFile)
		})

		// Default mode: silenced file should not appear
		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.NotContains(t, out, "silenced.ini")

		// --all mode: silenced file should appear with marker
		out, err = exec.RunCommand(DdevBin, []string{"utility", "check-custom-config", "--all"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "PHP")
		require.Contains(t, out, "silenced.ini")
		require.Contains(t, out, "(#ddev-silent-no-warn)")
	})

	// Test mysql config
	t.Run("mysql config", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		mysqlDir := filepath.Join(ddevDir, "mysql")
		err := os.MkdirAll(mysqlDir, 0755)
		require.NoError(t, err)

		mysqlFile := filepath.Join(mysqlDir, "custom.cnf")
		err = os.WriteFile(mysqlFile, []byte("[mysqld]\nmax_connections = 500\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(mysqlFile)
		})

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Database")
		require.Contains(t, out, "custom.cnf")
	})

	// Test nginx config
	t.Run("nginx config", func(t *testing.T) {
		// Configure to use nginx webserver
		_, err := exec.RunCommand(DdevBin, []string{"config", "--webserver-type=nginx-fpm"})
		require.NoError(t, err)
		t.Cleanup(func() {
			// Revert back to nginx
			_, _ = exec.RunCommand(DdevBin, []string{"config", "--webserver-type=" + nodeps.WebserverDefault})
		})
		ddevDir := filepath.Join(tmpdir, ".ddev")
		nginxDir := filepath.Join(ddevDir, "nginx")
		err = os.MkdirAll(nginxDir, 0755)
		require.NoError(t, err)

		nginxFile := filepath.Join(nginxDir, "custom.conf")
		err = os.WriteFile(nginxFile, []byte("# Custom nginx config\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(nginxFile)
		})

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Web server")
		require.Contains(t, out, "custom.conf")
	})

	// Test nginx_full config
	t.Run("nginx_full config", func(t *testing.T) {
		// Configure to use nginx webserver
		_, err := exec.RunCommand(DdevBin, []string{"config", "--webserver-type=nginx-fpm"})
		require.NoError(t, err)
		t.Cleanup(func() {
			// Revert back to nginx
			_, _ = exec.RunCommand(DdevBin, []string{"config", "--webserver-type=" + nodeps.WebserverDefault})
		})
		ddevDir := filepath.Join(tmpdir, ".ddev")
		nginxFullDir := filepath.Join(ddevDir, "nginx_full")
		err = os.MkdirAll(nginxFullDir, 0755)
		require.NoError(t, err)

		nginxFullFile := filepath.Join(nginxFullDir, "custom.conf")
		err = os.WriteFile(nginxFullFile, []byte("# Custom nginx_full config\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(nginxFullFile)
		})

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Web server")
		require.Contains(t, out, "custom.conf")
	})

	// Test postgres config (conditional: only when database type is postgres)
	t.Run("postgres config", func(t *testing.T) {
		// Configure to use postgres
		_, err := exec.RunCommand(DdevBin, []string{"config", "--database=postgres:18"})
		require.NoError(t, err)
		t.Cleanup(func() {
			// Revert back to mariadb
			_, _ = exec.RunCommand(DdevBin, []string{"config", "--database=" + nodeps.MariaDB + ":" + nodeps.MariaDBDefaultVersion})
		})

		ddevDir := filepath.Join(tmpdir, ".ddev")
		postgresDir := filepath.Join(ddevDir, "postgres")
		err = os.MkdirAll(postgresDir, 0755)
		require.NoError(t, err)

		postgresFile := filepath.Join(postgresDir, "custom.conf")
		err = os.WriteFile(postgresFile, []byte("# Custom postgres config\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(postgresFile)
		})

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Database")
		require.Contains(t, out, "custom.conf")
	})

	// Test providers
	t.Run("providers", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		providersDir := filepath.Join(ddevDir, "providers")
		err := os.MkdirAll(providersDir, 0755)
		require.NoError(t, err)

		customProvider := filepath.Join(providersDir, "custom.yaml")
		err = os.WriteFile(customProvider, []byte("# Custom provider\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(customProvider)
		})

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Hosting providers")
		require.Contains(t, out, "custom.yaml")
	})

	// Test share providers
	t.Run("share providers", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		shareProvidersDir := filepath.Join(ddevDir, "share-providers")
		err := os.MkdirAll(shareProvidersDir, 0755)
		require.NoError(t, err)

		customShare := filepath.Join(shareProvidersDir, "custom.sh")
		err = os.WriteFile(customShare, []byte("#!/bin/bash\n# Custom share provider\n"), 0755)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(customShare)
		})

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Share providers")
		require.Contains(t, out, "custom.sh")
	})

	// Test traefik certs
	t.Run("traefik certs", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		certsDir := filepath.Join(ddevDir, "traefik", "certs")
		err := os.MkdirAll(certsDir, 0755)
		require.NoError(t, err)

		customCert := filepath.Join(certsDir, "custom.crt")
		err = os.WriteFile(customCert, []byte("# Custom cert\n"), 0644)
		require.NoError(t, err)
		customKey := filepath.Join(certsDir, "custom.key")
		err = os.WriteFile(customKey, []byte("# Custom key\n"), 0600)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(customCert)
			_ = os.Remove(customKey)
		})

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Router")
		require.Contains(t, out, "custom.crt")
	})

	// Test custom_certs directory
	t.Run("custom certificates", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		customCertsDir := filepath.Join(ddevDir, "custom_certs")
		err := os.MkdirAll(customCertsDir, 0755)
		require.NoError(t, err)

		// Create custom certificate files
		certFile := filepath.Join(customCertsDir, "custom.crt")
		err = os.WriteFile(certFile, []byte("-----BEGIN CERTIFICATE-----\nfake cert\n-----END CERTIFICATE-----\n"), 0644)
		require.NoError(t, err)
		keyFile := filepath.Join(customCertsDir, "custom.key")
		err = os.WriteFile(keyFile, []byte("-----BEGIN PRIVATE KEY-----\nfake key\n-----END PRIVATE KEY-----\n"), 0600)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.RemoveAll(customCertsDir)
		})

		// Run check-custom-config
		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Router")
		require.Contains(t, out, "custom.crt")
		require.Contains(t, out, "custom.key")
	})

	// Test traefik config
	t.Run("traefik config", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		configDir := filepath.Join(ddevDir, "traefik", "config")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		customConfig := filepath.Join(configDir, "custom.yaml")
		err = os.WriteFile(customConfig, []byte("# Custom traefik config\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(customConfig)
		})

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Router")
		require.Contains(t, out, "custom.yaml")
	})

	// Test web-build Dockerfiles (all 6 patterns)
	t.Run("web-build Dockerfiles", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		webBuildDir := filepath.Join(ddevDir, "web-build")
		err := os.MkdirAll(webBuildDir, 0755)
		require.NoError(t, err)

		// Test all 6 valid Dockerfile patterns
		files := map[string]string{
			"Dockerfile":                "# Dockerfile\n",
			"Dockerfile.custom":         "# Dockerfile.custom\n",
			"pre.Dockerfile":            "# pre.Dockerfile\n",
			"pre.Dockerfile.custom":     "# pre.Dockerfile.custom\n",
			"prepend.Dockerfile":        "# prepend.Dockerfile\n",
			"prepend.Dockerfile.custom": "# prepend.Dockerfile.custom\n",
		}

		for name, content := range files {
			path := filepath.Join(webBuildDir, name)
			err = os.WriteFile(path, []byte(content), 0644)
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = os.Remove(path)
			})
		}

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Web server")
		// Check that all 6 files are detected
		for name := range files {
			require.Contains(t, out, name)
		}
	})

	// Test web-entrypoint.d
	t.Run("web-entrypoint.d", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		entrypointDir := filepath.Join(ddevDir, "web-entrypoint.d")
		err := os.MkdirAll(entrypointDir, 0755)
		require.NoError(t, err)

		customScript := filepath.Join(entrypointDir, "custom.sh")
		err = os.WriteFile(customScript, []byte("#!/bin/bash\n# Custom entrypoint script\n"), 0755)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(customScript)
		})

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Web server")
		require.Contains(t, out, "custom.sh")
	})

	// Test xhprof_prepend.php
	t.Run("xhprof config", func(t *testing.T) {
		// Configure to use xhprof prepend mode
		_, err := exec.RunCommand(DdevBin, []string{"config", "--xhprof-mode=prepend"})
		require.NoError(t, err)
		t.Cleanup(func() {
			// Revert back to default
			_, _ = exec.RunCommand(DdevBin, []string{"config", "--xhprof-mode="})
		})

		ddevDir := filepath.Join(tmpdir, ".ddev")
		xhprofDir := filepath.Join(ddevDir, "xhprof")
		err = os.MkdirAll(xhprofDir, 0755)
		require.NoError(t, err)

		// Create a custom xhprof_prepend.php file
		customXHProf := filepath.Join(xhprofDir, "xhprof_prepend.php")
		err = os.WriteFile(customXHProf, []byte("<?php\n// Custom xhprof config\nxhprof_enable();\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(customXHProf)
		})

		// Run check-custom-config
		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "XHProf")
		require.Contains(t, out, "xhprof_prepend.php")
	})

	// Test commands directory with 2-level depth
	t.Run("project commands at 2-level depth", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		commandsDir := filepath.Join(ddevDir, "commands", "web", "autocomplete")
		err := os.MkdirAll(commandsDir, 0755)
		require.NoError(t, err)

		// Create a custom command at 2-level depth
		customCmd := filepath.Join(commandsDir, "custom-complete")
		err = os.WriteFile(customCmd, []byte("#!/bin/bash\necho 'custom autocomplete'\n"), 0755)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.RemoveAll(filepath.Join(ddevDir, "commands"))
		})

		// Run check-custom-config
		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Commands")
		require.Contains(t, out, "custom-complete")
	})

	// Test homeadditions directory
	t.Run("project homeadditions", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		homeadditionsDir := filepath.Join(ddevDir, "homeadditions", ".bashrc.d")
		err := os.MkdirAll(homeadditionsDir, 0755)
		require.NoError(t, err)

		// Create a custom homeadditions file
		customFile := filepath.Join(homeadditionsDir, "custom.sh")
		err = os.WriteFile(customFile, []byte("# Custom bash config\nalias ll='ls -la'\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.RemoveAll(filepath.Join(ddevDir, "homeadditions"))
		})

		// Run check-custom-config
		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Home additions")
		require.Contains(t, out, "custom.sh")
	})

	// Test config files
	t.Run("config files", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		customConfig := filepath.Join(ddevDir, "config.custom.yaml")
		err := os.WriteFile(customConfig, []byte("# Custom config\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(customConfig)
		})

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Config")
		require.Contains(t, out, "config.custom.yaml")
	})

	// Test docker-compose files
	t.Run("docker-compose files", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		customCompose := filepath.Join(ddevDir, "docker-compose.custom.yaml")
		err := os.WriteFile(customCompose, []byte("# Custom docker-compose\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(customCompose)
		})

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Docker Compose")
		require.Contains(t, out, "docker-compose.custom.yaml")
	})

	// Test environment .env file
	t.Run("environment .env", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		customEnv := filepath.Join(ddevDir, ".env")
		err := os.WriteFile(customEnv, []byte("CUSTOM_VAR=value\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(customEnv)
		})

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Environment")
		require.Contains(t, out, ".env")
	})

	// Test environment .env.* files
	t.Run("environment .env.*", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		customEnvLocal := filepath.Join(ddevDir, ".env.local")
		err := os.WriteFile(customEnvLocal, []byte("LOCAL_VAR=value\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(customEnvLocal)
		})

		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "Environment")
		require.Contains(t, out, ".env.local")
	})

	// SPECIAL CASES

	// Test with unexpected DDEV-generated file
	t.Run("unexpected ddev-generated label", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		phpDir := filepath.Join(ddevDir, "php")

		// Create a file with #ddev-generated marker in unexpected location
		unexpectedFile := filepath.Join(phpDir, "unexpected.ini")
		err := os.WriteFile(unexpectedFile, []byte("#ddev-generated\nmemory_limit = 2G\n"), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(unexpectedFile)
		})

		// Run check-custom-config
		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "PHP")
		require.Contains(t, out, "unexpected.ini")
		require.Contains(t, out, "(unexpected #ddev-generated)")
		require.Contains(t, out, "Remove unexpected '#ddev-generated' comments")
	})

	// Test with addon-generated files
	t.Run("addon-generated files", func(t *testing.T) {
		if !github.HasGitHubToken() {
			t.Skip("Skipping because DDEV_GITHUB_TOKEN is not set")
		}

		// Install a simple addon to test addon file detection
		_, err := exec.RunCommand(DdevBin, []string{"add-on", "get", "ddev/ddev-phpmyadmin"})
		require.NoError(t, err)

		// Default mode: addon files should not appear (they are not user custom files)
		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.NotContains(t, out, "docker-compose.phpmyadmin.yaml")
		require.NotContains(t, out, "docker-compose.phpmyadmin_norouter.yaml")

		// --all mode: addon files should appear with (add-on name) marker
		out, err = exec.RunCommand(DdevBin, []string{"utility", "check-custom-config", "--all"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "docker-compose.phpmyadmin.yaml")
		require.Contains(t, out, "docker-compose.phpmyadmin_norouter.yaml")
		require.Contains(t, out, "(add-on phpmyadmin) (#ddev-generated)")
	})
}
