package ddevapp_test

import (
	"path/filepath"
	"testing"

	"os"

	"io/ioutil"

	"fmt"

	. "github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	"github.com/stretchr/testify/assert"
)

var appTypeSettingsLocations = map[string][]string{
	"drupal6":  {"sites/default/settings.php", "sites/default/settings.ddev.php"},
	"drupal7":  {"sites/default/settings.php", "sites/default/settings.ddev.php"},
	"drupal8":  {"sites/default/settings.php", "sites/default/settings.ddev.php"},
	"backdrop": {"settings.php", "settings.ddev.php"},
}

// TestWriteSettings tests writing app settings (like Drupal
// settings.php/settings.local.php
func TestWriteSettings(t *testing.T) {
	expectations := map[string]string{
		"backdrop":  "settings.ddev.php",
		"drupal6":   "sites/default/settings.ddev.php",
		"drupal7":   "sites/default/settings.ddev.php",
		"drupal8":   "sites/default/settings.ddev.php",
		"wordpress": "wp-config.php",
		"typo3":     "typo3conf/AdditionalConfiguration.php",
	}
	dir := testcommon.CreateTmpDir("example")
	err := os.MkdirAll(filepath.Join(dir, "sites/default"), 0777)
	assert.NoError(t, err)
	err = os.MkdirAll(filepath.Join(dir, "typo3conf"), 0777)
	assert.NoError(t, err)

	// TYPO3 wants LocalConfiguration.php to exist in the repo ahead of time.
	err = ioutil.WriteFile(filepath.Join(dir, "typo3conf", "LocalConfiguration.php"), []byte("<?php\n"), 0644)
	assert.NoError(t, err)

	app, err := NewApp(dir, DefaultProviderName)
	assert.NoError(t, err)

	for apptype, settingsRelativePath := range expectations {
		app.Type = apptype

		expectedSettingsFile := filepath.Join(dir, settingsRelativePath)
		_, err = os.Stat(expectedSettingsFile)
		assert.True(t, os.IsNotExist(err))
		// nolint: vetshadow
		createdFile, err := app.CreateSettingsFile()
		assert.NoError(t, err)
		assert.EqualValues(t, expectedSettingsFile, createdFile)
		_, err = os.Stat(expectedSettingsFile)
		assert.NoError(t, err)
		// nolint: vetshadow
		signatureFound, err := fileutil.FgrepStringInFile(expectedSettingsFile, DdevFileSignature)
		assert.NoError(t, err)
		assert.True(t, signatureFound)
		err = os.Remove(expectedSettingsFile)
		assert.NoError(t, err)
	}

	err = os.RemoveAll(dir)
	assert.NoError(t, err)
}

// @todo: Take a look at drush config in general to make sure its config
// is noted properly. Do we need it? Are we using it?
func TestWriteDrushConfig(t *testing.T) {

	dir := testcommon.CreateTmpDir("example")

	file, err := ioutil.TempFile(dir, "file")
	assert.NoError(t, err)

	drushConfig := NewDrushConfig()
	err = WriteDrushConfig(drushConfig, file.Name())
	assert.NoError(t, err)

	util.CheckClose(file)

	err = os.Chmod(dir, 0755)
	assert.NoError(t, err)
	err = os.Chmod(file.Name(), 0666)
	assert.NoError(t, err)

	err = os.RemoveAll(dir)
	assert.NoError(t, err)
}

// TestIncludeSettingsDdevInNewSettingsFile verifies that when no settings.php file exists,
// a settings.php file is created that includes settings.ddev.php.
func TestIncludeSettingsDdevInNewSettingsFile(t *testing.T) {
	dir := testcommon.CreateTmpDir("")
	err := os.MkdirAll(filepath.Join(dir, "sites/default"), 0777)
	assert.NoError(t, err)

	app, err := NewApp(dir, DefaultProviderName)
	assert.NoError(t, err)

	for appType, relativeSettingsLocations := range appTypeSettingsLocations {
		app.Type = appType

		relativeSettingsLocation := relativeSettingsLocations[0]
		relativeSettingsDdevLocation := relativeSettingsLocations[1]
		expectedSettingsLocation := filepath.Join(dir, relativeSettingsLocation)
		expectedSettingsDdevLocation := filepath.Join(dir, relativeSettingsDdevLocation)

		// Ensure that no settings.php exists
		os.Remove(expectedSettingsLocation)

		// Ensure that no settings.ddev.php file exists
		os.Remove(expectedSettingsDdevLocation)

		// Invoke the settings file creation process
		_, err := app.CreateSettingsFile()
		assert.NoError(t, err)

		// Ensure that a settings.php was created
		assert.True(t, fileutil.FileExists(expectedSettingsLocation))

		// Ensure that settings.php references settings.ddev.php
		newSettingsIncludesSettingsDdev, err := fileutil.FgrepStringInFile(expectedSettingsLocation, relativeSettingsDdevLocation)
		assert.NoError(t, err)
		assert.True(t, newSettingsIncludesSettingsDdev)

		// Ensure that settings.ddev.php exists
		assert.True(t, fileutil.FileExists(expectedSettingsDdevLocation))
	}
}

// TestIncludeSettingsDdevInExistingSettingsFile verifies that when a settings.php file already exists,
// it is modified to include settings.ddev.php
func TestIncludeSettingsDdevInExistingSettingsFile(t *testing.T) {
	dir := testcommon.CreateTmpDir("")
	err := os.MkdirAll(filepath.Join(dir, "sites/default"), 0777)
	assert.NoError(t, err)

	app, err := NewApp(dir, DefaultProviderName)
	assert.NoError(t, err)

	for appType, relativeSettingsLocations := range appTypeSettingsLocations {
		app.Type = appType

		relativeSettingsLocation := relativeSettingsLocations[0]
		relativeSettingsDdevLocation := relativeSettingsLocations[1]
		expectedSettingsLocation := filepath.Join(dir, relativeSettingsLocation)
		expectedSettingsDdevLocation := filepath.Join(dir, relativeSettingsDdevLocation)

		// Ensure that no settings.php exists
		os.Remove(expectedSettingsLocation)

		// Ensure that no settings.ddev.php file exists
		os.Remove(expectedSettingsDdevLocation)

		// Create a settings.php that does not include settings.ddev.php
		originalContents := "not empty"
		settingsFile, err := os.Create(expectedSettingsLocation)
		assert.NoError(t, err)
		_, err = settingsFile.Write([]byte(originalContents))
		assert.NoError(t, err)

		// Invoke the settings file creation process
		_, err = app.CreateSettingsFile()
		assert.NoError(t, err)

		// Ensure that settings.php exists
		assert.True(t, fileutil.FileExists(expectedSettingsLocation))

		// Ensure that settings.ddev.php exists
		assert.True(t, fileutil.FileExists(expectedSettingsDdevLocation))

		// Ensure that settings.php references settings.ddev.php
		existingSettingsIncludesSettingsDdev, err := fileutil.FgrepStringInFile(expectedSettingsLocation, relativeSettingsDdevLocation)
		assert.NoError(t, err)
		assert.True(t, existingSettingsIncludesSettingsDdev)

		// Ensure that settings.php includes original contents
		modifiedSettingsIncludesOriginalContents, err := fileutil.FgrepStringInFile(expectedSettingsLocation, originalContents)
		assert.NoError(t, err)
		assert.True(t, modifiedSettingsIncludesOriginalContents)
	}
}

// TestCreateGitIgnoreIfNoneExists verifies that if no .gitignore file exists in the directory
// containing settings.php and settings.ddev.php, a .gitignore is created that includes settings.ddev.php.
func TestCreateGitIgnoreIfNoneExists(t *testing.T) {
	dir := testcommon.CreateTmpDir("")
	err := os.MkdirAll(filepath.Join(dir, "sites/default"), 0777)
	assert.NoError(t, err)

	app, err := NewApp(dir, DefaultProviderName)
	assert.NoError(t, err)

	for appType, relativeSettingsLocations := range appTypeSettingsLocations {
		app.Type = appType

		relativeSettingsDdevLocation := relativeSettingsLocations[1]
		expectedSettingsDdevLocation := filepath.Join(dir, relativeSettingsDdevLocation)
		expectedGitIgnoreLocation := filepath.Join(filepath.Dir(expectedSettingsDdevLocation), ".gitignore")
		fmt.Println(expectedGitIgnoreLocation)

		// Ensure that no .gitignore exists
		os.Remove(expectedGitIgnoreLocation)

		// Invoke the settings file creation process
		_, err = app.CreateSettingsFile()
		assert.NoError(t, err)

		// Ensure that a .gitignore exists
		assert.True(t, fileutil.FileExists(expectedGitIgnoreLocation))

		c, _ := ioutil.ReadFile(expectedGitIgnoreLocation)
		fmt.Printf("gitignore contents: %s", c)

		// Ensure that the new .gitignore includes settings.ddev.php
		newGitIgnoreIncludesSettingsDdev, err := fileutil.FgrepStringInFile(expectedGitIgnoreLocation, filepath.Base(relativeSettingsDdevLocation))
		assert.NoError(t, err)
		assert.True(t, newGitIgnoreIncludesSettingsDdev)
	}
}

// TestGitIgnoreAlreadyExists verifies that if a .gitignore already exists in the directory
// containing settings.php and settings.ddev.php, it is not modified.
func TestGitIgnoreAlreadyExists(t *testing.T) {
	dir := testcommon.CreateTmpDir("")
	err := os.MkdirAll(filepath.Join(dir, "sites/default"), 0777)
	assert.NoError(t, err)

	app, err := NewApp(dir, DefaultProviderName)
	assert.NoError(t, err)

	for appType, relativeSettingsLocations := range appTypeSettingsLocations {
		app.Type = appType

		relativeSettingsDdevLocation := relativeSettingsLocations[1]
		expectedSettingsDdevLocation := filepath.Join(dir, relativeSettingsDdevLocation)
		expectedGitIgnoreLocation := filepath.Join(filepath.Dir(expectedSettingsDdevLocation), ".gitignore")
		fmt.Println(expectedGitIgnoreLocation)

		// Ensure that a .gitignore already exists and has some contents
		originalContents := "not empty"
		settingsFile, err := os.Create(expectedGitIgnoreLocation)
		assert.NoError(t, err)
		_, err = settingsFile.Write([]byte(originalContents))
		assert.NoError(t, err)

		// Invoke the settings file creation process
		_, err = app.CreateSettingsFile()
		assert.NoError(t, err)

		// Ensure that .gitignore still exists
		assert.True(t, fileutil.FileExists(expectedGitIgnoreLocation))

		// Ensure that the new .gitignore has not been modified to include settings.ddev.php
		existingGitIgnoreIncludesSettingsDdev, err := fileutil.FgrepStringInFile(expectedGitIgnoreLocation, filepath.Base(relativeSettingsDdevLocation))
		assert.NoError(t, err)
		assert.False(t, existingGitIgnoreIncludesSettingsDdev)
	}
}
