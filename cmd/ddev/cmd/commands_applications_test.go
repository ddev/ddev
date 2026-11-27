package cmd

import (
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func newRootCommandForCustomCommands() *cobra.Command {
	rootCmd := &cobra.Command{Use: "ddev"}
	rootCmd.AddCommand(&cobra.Command{Use: "existing"})
	return rootCmd
}

func commandNames(rootCmd *cobra.Command) []string {
	names := make([]string, 0, len(rootCmd.Commands()))
	for _, command := range rootCmd.Commands() {
		names = append(names, command.Name())
	}
	return names
}

func writeBundledCommandForTest(t *testing.T, command, systemAppsDir string) string {
	t.Helper()

	bundled, err := os.ReadFile(filepath.Join("..", "..", "..", "pkg", "ddevapp", "global_dotddev_assets", "commands", "host", command))
	require.NoError(t, err)
	script := "## CanRunGlobally: true\n" + strings.Replace(string(bundled), "HostBinaryExists: /Applications/", "HostBinaryExists: "+systemAppsDir, 1)
	script = strings.ReplaceAll(script, ",/Applications/", ","+systemAppsDir)
	commandsDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(commandsDir, command), []byte(script), 0755))
	return commandsDir
}

func assertCommandRegistered(t *testing.T, commandsDir, command string) {
	t.Helper()

	rootCmd := newRootCommandForCustomCommands()
	require.NoError(t, addCustomCommandsFromDir(rootCmd, nil, commandsDir, []string{command}, true, map[string]int{}))
	require.Contains(t, commandNames(rootCmd), command)
}

func prepareStubOpen(t *testing.T) {
	t.Helper()

	binDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "open"), []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" >> \"$OPEN_LOG\"\n"), 0755))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("OPEN_LOG", filepath.Join(binDir, "open.log"))
	t.Setenv("DDEV_PROJECT_STATUS", "running")
	t.Setenv("DDEV_HOST_DB_PORT", "3306")
	t.Setenv("DDEV_SITENAME", "mysite")
	t.Setenv("DDEV_TLD", "ddev.site")
	t.Setenv("DDEV_PROJECT", "mysite")
	t.Setenv("DDEV_DBIMAGE", "mariadb:10.11")
}

func assertOpensExecutable(t *testing.T, script, executable string) {
	t.Helper()

	out, err := osexec.Command("bash", script).CombinedOutput()
	require.NoError(t, err, string(out))
	logged, err := os.ReadFile(os.Getenv("OPEN_LOG"))
	require.NoError(t, err)
	require.Contains(t, string(logged), executable)
}

func TestHostBinaryExistsHomeApplications(t *testing.T) {
	if !nodeps.IsMacOS() {
		t.Skip("macOS app lookup")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	appPath := filepath.Join(home, "Applications", "TestGUIApp.app")
	require.NoError(t, os.MkdirAll(appPath, 0755))

	commandsDir := t.TempDir()
	scriptPath := filepath.Join(commandsDir, "testguiapp")
	script := "#!/usr/bin/env bash\n## CanRunGlobally: true\n## Description: Test GUI app\n## HostBinaryExists: ~/Applications/TestGUIApp.app\n"
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0755))

	rootCmd := newRootCommandForCustomCommands()
	require.NoError(t, addCustomCommandsFromDir(rootCmd, nil, commandsDir, []string{"testguiapp"}, true, map[string]int{}))
	require.Contains(t, commandNames(rootCmd), "testguiapp")

	require.NoError(t, os.RemoveAll(appPath))
	rootCmd = newRootCommandForCustomCommands()
	require.NoError(t, addCustomCommandsFromDir(rootCmd, nil, commandsDir, []string{"testguiapp"}, true, map[string]int{}))
	require.NotContains(t, commandNames(rootCmd), "testguiapp")
}

func TestBundledDatabaseGUICommandsHomeApplications(t *testing.T) {
	if !nodeps.IsMacOS() {
		t.Skip("macOS app lookup")
	}

	for _, testCase := range []struct {
		command string
		app     string
	}{
		{"tablepro", "TablePro.app"},
		{"tableplus", "TablePlus.app"},
		{"sequelace", "Sequel Ace.app"},
		{"querious", "Querious.app"},
		{"dbeaver", "DBeaver.app"},
	} {
		t.Run(testCase.command, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			systemAppsDir := filepath.Join(home, "SystemApplications") + string(os.PathSeparator)
			require.NoError(t, os.MkdirAll(filepath.Join(home, "Applications", testCase.app), 0755))
			commandsDir := writeBundledCommandForTest(t, testCase.command, systemAppsDir)
			assertCommandRegistered(t, commandsDir, testCase.command)
		})
	}
}

func TestBundledDatabaseGUICommandsSystemApplications(t *testing.T) {
	if !nodeps.IsMacOS() {
		t.Skip("macOS app lookup")
	}

	for _, testCase := range []struct {
		command string
		app     string
	}{
		{"tablepro", "TablePro.app"},
		{"tableplus", "TablePlus.app"},
		{"sequelace", "Sequel Ace.app"},
		{"querious", "Querious.app"},
		{"dbeaver", "DBeaver.app"},
	} {
		t.Run(testCase.command, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			systemAppsDir := filepath.Join(home, "SystemApplications") + string(os.PathSeparator)
			require.NoError(t, os.MkdirAll(filepath.Join(systemAppsDir, testCase.app), 0755))
			commandsDir := writeBundledCommandForTest(t, testCase.command, systemAppsDir)
			assertCommandRegistered(t, commandsDir, testCase.command)
		})
	}
}

func TestTablePlusSetappInstallation(t *testing.T) {
	if !nodeps.IsMacOS() {
		t.Skip("macOS app lookup")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	systemAppsDir := filepath.Join(home, "SystemApplications") + string(os.PathSeparator)
	require.NoError(t, os.MkdirAll(filepath.Join(systemAppsDir, "Setapp", "TablePlus.app"), 0755))
	commandsDir := writeBundledCommandForTest(t, "tableplus", systemAppsDir)
	assertCommandRegistered(t, commandsDir, "tableplus")
}

func TestDatabaseGUICommandsLaunchFromHomeApplications(t *testing.T) {
	if !nodeps.IsMacOS() {
		t.Skip("macOS app launch")
	}

	for _, testCase := range []struct {
		command    string
		app        string
		executable string
	}{
		{"tableplus", "TablePlus.app", "TablePlus"},
		{"sequelace", "Sequel Ace.app", "Sequel Ace"},
	} {
		t.Run(testCase.command, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			prepareStubOpen(t)
			executableDir := filepath.Join(home, "Applications", testCase.app, "Contents", "MacOS")
			require.NoError(t, os.MkdirAll(executableDir, 0755))

			script := filepath.Join("..", "..", "..", "pkg", "ddevapp", "global_dotddev_assets", "commands", "host", testCase.command)
			assertOpensExecutable(t, script, filepath.Join(executableDir, testCase.executable))
		})
	}
}

func TestDatabaseGUICommandsLaunchFromSystemApplications(t *testing.T) {
	if !nodeps.IsMacOS() {
		t.Skip("macOS app launch")
	}

	for _, testCase := range []struct {
		command    string
		app        string
		executable string
	}{
		{"tableplus", "TablePlus.app", "TablePlus"},
		{"sequelace", "Sequel Ace.app", "Sequel Ace"},
	} {
		t.Run(testCase.command, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			prepareStubOpen(t)
			systemAppsDir := filepath.Join(home, "SystemApplications")
			executableDir := filepath.Join(systemAppsDir, testCase.app, "Contents", "MacOS")
			require.NoError(t, os.MkdirAll(executableDir, 0755))

			bundled, err := os.ReadFile(filepath.Join("..", "..", "..", "pkg", "ddevapp", "global_dotddev_assets", "commands", "host", testCase.command))
			require.NoError(t, err)
			script := filepath.Join(t.TempDir(), testCase.command)
			rewritten := strings.ReplaceAll(string(bundled), "\"/Applications/", "\""+systemAppsDir+string(os.PathSeparator))
			require.NoError(t, os.WriteFile(script, []byte(rewritten), 0755))
			assertOpensExecutable(t, script, filepath.Join(executableDir, testCase.executable))
		})
	}
}
