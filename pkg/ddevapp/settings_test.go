package ddevapp_test

import (
	"path/filepath"
	"testing"

	"os"

	"io/ioutil"

	. "github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	"github.com/stretchr/testify/assert"
)

// TestWriteSettings tests writing app settings (like Drupal
// settings.php/settings.local.php
func TestWriteSettings(t *testing.T) {
	expectations := map[string]string{
		"drupal7":   "sites/default/settings.php",
		"drupal8":   "sites/default/settings.php",
		"wordpress": "wp-config.php",
		"typo3":     "typo3conf/AdditionalConfiguration.php",
	}
	dir := testcommon.CreateTmpDir("example")
	err := os.MkdirAll(filepath.Join(dir, "sites/default"), 0777)
	assert.NoError(t, err)
	err = os.MkdirAll(filepath.Join(dir, "typo3conf"), 0777)
	assert.NoError(t, err)

	// typo3 wants LocalConfiguration.php to exist in the repo ahead of time.
	err = ioutil.WriteFile(filepath.Join(dir, "typo3conf", "LocalConfiguration.php"), []byte("<?php\n"), 0644)
	assert.NoError(t, err)

	app, err := NewApp(dir, DefaultProviderName)
	assert.NoError(t, err)

	for apptype, settingsRelativePath := range expectations {
		app.Type = apptype

		expectedSettingsFile := filepath.Join(dir, settingsRelativePath)
		_, err = os.Stat(expectedSettingsFile)
		assert.True(t, os.IsNotExist(err))
		createdFile, err := app.CreateSettingsFile()
		assert.NoError(t, err)
		assert.EqualValues(t, expectedSettingsFile, createdFile)
		_, err = os.Stat(expectedSettingsFile)
		assert.NoError(t, err)
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
