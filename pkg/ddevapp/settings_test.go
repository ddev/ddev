package ddevapp_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

type settingsLocations struct {
	main  string
	local string
}

var drupalBackdropSettingsLocations = map[string]settingsLocations{
	nodeps.AppTypeDrupal6:  {main: "sites/default/settings.php", local: "sites/default/settings.ddev.php"},
	nodeps.AppTypeDrupal7:  {main: "sites/default/settings.php", local: "sites/default/settings.ddev.php"},
	nodeps.AppTypeDrupal11: {main: "sites/default/settings.php", local: "sites/default/settings.ddev.php"},
	nodeps.AppTypeDrupal12: {main: "sites/default/settings.php", local: "sites/default/settings.ddev.php"},
	nodeps.AppTypeBackdrop: {main: "settings.php", local: "settings.ddev.php"},
}

// TestWriteSettings tests writing app settings (like Drupal
// settings.php/settings.ddev.php
func TestWriteSettings(t *testing.T) {
	expectations := map[string]string{
		nodeps.AppTypeBackdrop:  "settings.ddev.php",
		nodeps.AppTypeDrupal6:   "sites/default/settings.ddev.php",
		nodeps.AppTypeDrupal7:   "sites/default/settings.ddev.php",
		nodeps.AppTypeDrupal11:  "sites/default/settings.ddev.php",
		nodeps.AppTypeWordPress: "wp-config-ddev.php",
		nodeps.AppTypeTYPO3:     "typo3conf/AdditionalConfiguration.php",
	}
	testDir := testcommon.CreateTmpDir(t.Name())

	app, err := ddevapp.NewApp(testDir, true)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		require.NoError(t, err)
		_ = os.RemoveAll(testDir)
	})

	err = os.MkdirAll(filepath.Join(testDir, app.Docroot, "sites", "default"), 0777)
	require.NoError(t, err)

	// Create expected folders for TYPO3.
	err = os.MkdirAll(filepath.Join(testDir, app.Docroot, "typo3"), 0777)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Join(testDir, app.Docroot, "typo3conf"), 0777)
	require.NoError(t, err)

	// TYPO3 wants LocalConfiguration.php to exist in the repo ahead of time.
	err = os.WriteFile(filepath.Join(testDir, app.Docroot, "typo3conf", "LocalConfiguration.php"), []byte("<?php\n"), 0644)
	require.NoError(t, err)

	for apptype, settingsRelativePath := range expectations {
		app.Type = apptype

		expectedSettingsFile := filepath.Join(testDir, settingsRelativePath)
		_, err = os.Stat(expectedSettingsFile)
		require.True(t, os.IsNotExist(err))
		createdFile, err := app.CreateSettingsFile()
		require.NoError(t, err)
		require.EqualValues(t, expectedSettingsFile, createdFile)
		_, err = os.Stat(expectedSettingsFile)
		require.NoError(t, err)
		signatureFound, err := fileutil.FgrepStringInFile(expectedSettingsFile, nodeps.DdevFileSignature)
		require.NoError(t, err)
		require.True(t, signatureFound, "Failed to find %s in %s", nodeps.DdevFileSignature, expectedSettingsFile)

		// For WordPress, verify WP_HOME uses runtime env vars instead of hardcoded URL
		if apptype == nodeps.AppTypeWordPress {
			envVarFound, err := fileutil.FgrepStringInFile(expectedSettingsFile, "DDEV_PRIMARY_URL")
			require.NoError(t, err)
			require.True(t, envVarFound, "Expected DDEV_PRIMARY_URL in %s for dynamic WP_HOME", expectedSettingsFile)
		}

		_ = os.Remove(expectedSettingsFile)
	}

	println("") // Just lets Goland find the PASS when done.
}

// TestWriteDrushConfig test the Drush config we write
func TestWriteDrushConfig(t *testing.T) {
	app := &ddevapp.DdevApp{}
	origDir, _ := os.Getwd()

	for i, site := range TestSites {
		if site.Disable {
			t.Logf("Skipping TestSite %s=%d because disabled", site.Name, i)
			continue
		}

		runTime := util.TimeTrackC(fmt.Sprintf("%s WriteDrushrc", site.Name))

		testcommon.ClearDockerEnv()

		if !nodeps.ArrayContainsString([]string{"drupal7", "drupal10", "drupal11", "drupal12", "drupal", "backdrop"}, site.Type) {
			continue
		}
		err := app.Init(site.Dir)
		if err != nil {
			require.NoError(t, err, "failed init of %s: %v", site.Name, err)
			continue
		}
		t.Cleanup(func() {
			err = os.Chdir(origDir)
			require.NoError(t, err)
			err = app.Stop(true, false)
			require.NoError(t, err)
		})

		_, err = app.CreateSettingsFile()
		require.NoError(t, err)

		startErr := app.Start()
		//nolint: errcheck
		defer app.Stop(true, false)
		if startErr != nil {
			logs, health, _ := ddevapp.GetErrLogsFromApp(app, startErr)
			t.Fatalf("app.Start failed, startErr=%v, healthcheck:\n%s\n\nlogs=\n========\n%s\n===========\n", startErr, health, logs)
		}

		drushFilePath := filepath.Join(filepath.Dir(app.SiteSettingsPath), "drushrc.php")

		switch app.Type {
		case nodeps.AppTypeDrupal6, nodeps.AppTypeDrupal7, nodeps.AppTypeBackdrop:
			if !fileutil.FileExists(drushFilePath) {
				require.True(t, fileutil.FileExists(drushFilePath))
				continue
			}
			optionFound, err := fileutil.FgrepStringInFile(drushFilePath, "options")
			require.NoError(t, err)
			require.True(t, optionFound)

		default:
			if fileutil.FileExists(drushFilePath) {
				require.False(t, fileutil.FileExists(drushFilePath), "Drush settings file (%s) should not exist but it does (app.Type=%s)", drushFilePath, app.Type)
				continue
			}
		}

		runTime()
	}
}

// TestDrupalBackdropIncludeSettingsDdevInNewSettingsFile verifies that when no settings.php file exists,
// a settings.php file is created that includes settings.ddev.php.
func TestDrupalBackdropIncludeSettingsDdevInNewSettingsFile(t *testing.T) {
	dir := testcommon.CreateTmpDir(t.Name())

	app, err := ddevapp.NewApp(dir, true)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Join(dir, app.Docroot, "sites", "default"), 0777)
	require.NoError(t, err)

	for appType, relativeSettingsLocations := range drupalBackdropSettingsLocations {
		app.Type = appType

		relativeSettingsLocation := relativeSettingsLocations.main
		relativeSettingsDdevLocation := relativeSettingsLocations.local
		expectedSettingsLocation := filepath.Join(dir, relativeSettingsLocation)
		expectedSettingsDdevLocation := filepath.Join(dir, relativeSettingsDdevLocation)

		// Ensure that no settings.php exists
		_ = os.Remove(expectedSettingsLocation)

		// Ensure that no settings.ddev.php file exists
		_ = os.Remove(expectedSettingsDdevLocation)

		// Invoke the settings file creation process
		_, err := app.CreateSettingsFile()
		require.NoError(t, err)

		// Ensure that a settings.php was created
		require.True(t, fileutil.FileExists(expectedSettingsLocation))

		// Ensure that settings.php references settings.ddev.php
		settingsDdev := filepath.Base(relativeSettingsDdevLocation)
		newSettingsIncludesSettingsDdev, err := fileutil.FgrepStringInFile(expectedSettingsLocation, settingsDdev)
		require.NoError(t, err)
		require.True(t, newSettingsIncludesSettingsDdev, "Failed to find %s in %s", settingsDdev, expectedSettingsLocation)

		// Ensure that settings.ddev.php exists
		require.True(t, fileutil.FileExists(expectedSettingsDdevLocation))
	}
}

// TestDrupalBackdropIncludeSettingsDdevInExistingSettingsFile verifies that when a settings.php file already exists,
// it is modified to include settings.ddev.php
func TestDrupalBackdropIncludeSettingsDdevInExistingSettingsFile(t *testing.T) {
	dir := testcommon.CreateTmpDir(t.Name())

	app, err := ddevapp.NewApp(dir, true)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Join(dir, app.Docroot, "sites", "default"), 0777)
	require.NoError(t, err)

	for appType, relativeSettingsLocations := range drupalBackdropSettingsLocations {
		app.Type = appType

		relativeSettingsLocation := relativeSettingsLocations.main
		relativeSettingsDdevLocation := relativeSettingsLocations.local
		expectedSettingsLocation := filepath.Join(dir, relativeSettingsLocation)
		expectedSettingsDdevLocation := filepath.Join(dir, relativeSettingsDdevLocation)

		// Ensure that no settings.php exists
		_ = os.Remove(expectedSettingsLocation)

		// Ensure that no settings.ddev.php file exists
		_ = os.Remove(expectedSettingsDdevLocation)

		// Create a settings.php that does not include settings.ddev.php
		originalContents := "// this file is not empty\n"
		err = os.WriteFile(expectedSettingsLocation, []byte(originalContents), 0644)
		require.NoError(t, err)

		// Invoke the settings file creation process
		_, err = app.CreateSettingsFile()
		require.NoError(t, err)

		// Ensure that settings.php exists
		require.True(t, fileutil.FileExists(expectedSettingsLocation))

		// Ensure that settings.ddev.php exists
		require.True(t, fileutil.FileExists(expectedSettingsDdevLocation))

		// Ensure that settings.php references settings.ddev.php
		settingsDdev := filepath.Base(relativeSettingsDdevLocation)
		existingSettingsIncludesSettingsDdev, err := fileutil.FgrepStringInFile(expectedSettingsLocation, settingsDdev)
		require.NoError(t, err)
		require.True(t, existingSettingsIncludesSettingsDdev, "Failed to find %s in %s, apptype=%s", settingsDdev, expectedSettingsLocation, appType)

		// Ensure that settings.php includes original contents
		modifiedSettingsIncludesOriginalContents, err := fileutil.FgrepStringInFile(expectedSettingsLocation, originalContents)
		require.NoError(t, err)
		require.True(t, modifiedSettingsIncludesOriginalContents, "Failed to find %s in %s", originalContents, expectedSettingsLocation)
	}
}

// TestDrupalBackdropCreateGitIgnoreIfNoneExists verifies that if no .gitignore file exists in the directory
// containing settings.php and settings.ddev.php, a .gitignore is created that includes settings.ddev.php.
func TestDrupalBackdropCreateGitIgnoreIfNoneExists(t *testing.T) {
	dir := testcommon.CreateTmpDir(t.Name())

	app, err := ddevapp.NewApp(dir, true)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Join(dir, app.Docroot, "sites", "default"), 0777)
	require.NoError(t, err)

	for appType, relativeSettingsLocations := range drupalBackdropSettingsLocations {
		app.Type = appType

		relativeSettingsDdevLocation := relativeSettingsLocations.local
		expectedSettingsDdevLocation := filepath.Join(dir, relativeSettingsDdevLocation)
		expectedGitIgnoreLocation := filepath.Join(filepath.Dir(expectedSettingsDdevLocation), ".gitignore")
		fmt.Println(expectedGitIgnoreLocation)

		// Ensure that no .gitignore exists
		_ = os.Remove(expectedGitIgnoreLocation)

		// Invoke the settings file creation process
		_, err = app.CreateSettingsFile()
		require.NoError(t, err)

		// Ensure that a .gitignore exists (except for Backdrop, which has settings in project root)
		if app.Type != nodeps.AppTypeBackdrop {
			require.True(t, fileutil.FileExists(expectedGitIgnoreLocation))

			// Ensure that the new .gitignore includes settings.ddev.php
			settingsDdev := filepath.Base(relativeSettingsDdevLocation)
			newGitIgnoreIncludesSettingsDdev, err := fileutil.FgrepStringInFile(expectedGitIgnoreLocation, settingsDdev)
			require.NoError(t, err)
			require.True(t, newGitIgnoreIncludesSettingsDdev, "Failed to find %s in %s", settingsDdev, expectedGitIgnoreLocation)
		}
	}
}

// TestDrupalBackdropConsistentHash makes sure that the hash_salt provided in
// settings.ddev.php is consistent across multiple `ddev start` but
// different between two project names
// Requires a drupal/backdrop project
func TestDrupalBackdropConsistentHash(t *testing.T) {
	projectTypes := []string{nodeps.AppTypeDrupal7, nodeps.AppTypeDrupal11, nodeps.AppTypeBackdrop}
	for _, projectType := range projectTypes {
		// Make a spare directory for the first project
		firstProjectDir := testcommon.CreateTmpDir(t.Name() + "-firstproject")
		app, err := ddevapp.NewApp(firstProjectDir, true)
		app.Type = projectType
		require.NoError(t, err)

		secondProjectDir := testcommon.CreateTmpDir(t.Name() + "-secondproject")
		secondApp, err := ddevapp.NewApp(secondProjectDir, true)
		secondApp.Type = projectType
		require.NoError(t, err)

		t.Cleanup(func() {
			err = app.Stop(true, false)
			require.NoError(t, err)
			_ = os.RemoveAll(firstProjectDir)
			err = secondApp.Stop(true, false)
			require.NoError(t, err)
			_ = os.RemoveAll(secondProjectDir)
		})
		// Start project and extract hash
		_, err = app.CreateSettingsFile()
		require.NoError(t, err)
		// Detect the hashSalt
		hash1, err := extractSettingsHashSalt(app)
		require.NoError(t, err)
		err = os.RemoveAll(app.SiteDdevSettingsFile)
		require.NoError(t, err)

		// Now restart project and make sure hash is same.
		_, err = app.CreateSettingsFile()
		require.NoError(t, err)
		// Detect the hashSalt
		hash2, err := extractSettingsHashSalt(app)
		require.NoError(t, err)

		require.Equal(t, hash1, hash2, "Hash should be the same across restarts")

		_, err = secondApp.CreateSettingsFile()
		require.NoError(t, err)
		secondProjectHash, err := extractSettingsHashSalt(secondApp)
		require.NoError(t, err)
		require.NotEqual(t, hash1, secondProjectHash, "Hash should be different for different projects")
	}
}

func extractSettingsHashSalt(app *ddevapp.DdevApp) (string, error) {
	// Read the file content
	content, err := os.ReadFile(app.SiteDdevSettingsFile)
	if err != nil {
		return "", err
	}

	// Use a regular expression to extract hash_salt value
	regex := regexp.MustCompile(`\$settings\['hash_salt'\] = '([a-fA-F0-9]{64})';`)
	if app.Type == nodeps.AppTypeDrupal7 {
		regex = regexp.MustCompile(`\$drupal_hash_salt = '([a-zA-Z0-9]{64})';`)
	}
	match := regex.FindStringSubmatch(string(content))
	if len(match) < 2 {
		return "", fmt.Errorf("hash_salt not found")
	}
	return match[1], nil
}

// TestDrupalBackdropGitIgnoreAlreadyExists verifies that if a .gitignore already exists in the directory
// containing settings.php and settings.ddev.php, it is not modified.
func TestDrupalBackdropGitIgnoreAlreadyExists(t *testing.T) {
	dir := testcommon.CreateTmpDir(t.Name())

	app, err := ddevapp.NewApp(dir, true)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Join(dir, app.Docroot, "sites", "default"), 0777)
	require.NoError(t, err)

	for appType, relativeSettingsLocations := range drupalBackdropSettingsLocations {
		app.Type = appType

		relativeSettingsDdevLocation := relativeSettingsLocations.local
		expectedSettingsDdevLocation := filepath.Join(dir, relativeSettingsDdevLocation)
		expectedGitIgnoreLocation := filepath.Join(filepath.Dir(expectedSettingsDdevLocation), ".gitignore")
		fmt.Println(expectedGitIgnoreLocation)

		// Ensure that a .gitignore already exists and has some contents
		originalContents := "not empty"
		settingsFile, err := os.Create(expectedGitIgnoreLocation)
		require.NoError(t, err)
		_, err = settingsFile.Write([]byte(originalContents))
		require.NoError(t, err)

		// Invoke the settings file creation process
		_, err = app.CreateSettingsFile()
		require.NoError(t, err)

		// Ensure that .gitignore still exists
		require.True(t, fileutil.FileExists(expectedGitIgnoreLocation))

		// Ensure that the new .gitignore has not been modified to include settings.ddev.php
		settingsDdev := relativeSettingsDdevLocation
		existingGitIgnoreIncludesSettingsDdev, err := fileutil.FgrepStringInFile(expectedGitIgnoreLocation, settingsDdev)
		require.NoError(t, err)
		require.False(t, existingGitIgnoreIncludesSettingsDdev, "Found unexpected %s in %s", settingsDdev, expectedGitIgnoreLocation)
	}
}

// TestDrupalBackdropOverwriteDdevSettings ensures that if a settings.ddev.php file already exists, it is overwritten by the
// settings creation process.
func TestDrupalBackdropOverwriteDdevSettings(t *testing.T) {
	dir := testcommon.CreateTmpDir(t.Name())

	app, err := ddevapp.NewApp(dir, true)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Join(dir, app.Docroot, "sites", "default"), 0777)
	require.NoError(t, err)

	for appType, relativeSettingsLocations := range drupalBackdropSettingsLocations {
		app.Type = appType

		relativeSettingsDdevLocation := relativeSettingsLocations.local
		expectedSettingsDdevLocation := filepath.Join(dir, relativeSettingsDdevLocation)

		// Ensure that a settings.ddev.php file exists, WITH the #ddev-generated signature
		originalContents := "not empty " + nodeps.DdevFileSignature
		settingsFile, err := os.Create(expectedSettingsDdevLocation)
		require.NoError(t, err)
		_, err = settingsFile.Write([]byte(originalContents))
		require.NoError(t, err)

		// Invoke the settings file creation process
		_, err = app.CreateSettingsFile()
		require.NoError(t, err)

		// Ensure settings.ddev.php was overwritten; It had the signature in it
		// so it was valid to overwrite. The original string should no longer be there.
		containsOriginalString, err := fileutil.FgrepStringInFile(expectedSettingsDdevLocation, originalContents)
		require.NoError(t, err)
		require.False(t, containsOriginalString, "The file should not have contained the original string %s and it did not.", originalContents)

		// Now do the whole thing again, but this time the settings.ddev.php does *not* have
		// the #ddev-generated signature, so the file will be respected and not replaced
		originalContents = "nearly empty "
		settingsFile, err = os.Create(expectedSettingsDdevLocation)
		require.NoError(t, err)
		_, err = settingsFile.Write([]byte(originalContents))
		require.NoError(t, err)

		// Invoke the settings file creation process
		_, err = app.CreateSettingsFile()
		require.NoError(t, err)

		// Ensure settings.ddev.php was overwritten with new contents
		containsOriginalString, err = fileutil.FgrepStringInFile(expectedSettingsDdevLocation, originalContents)
		require.NoError(t, err)
		require.True(t, containsOriginalString, "Did not find %s in the settings file; it should have still been there", originalContents)
	}
}
