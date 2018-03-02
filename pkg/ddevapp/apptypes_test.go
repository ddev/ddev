package ddevapp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"io/ioutil"
)

// TestApptypeDetection does a simple test of various filesystem setups to make
// sure the expected apptype is returned.
func TestApptypeDetection(t *testing.T) {
	assert := asrt.New(t)

	fileLocations := map[string]string{
		"wordpress": "wp-login.php",
		"backdrop":  "core/scripts/backdrop.sh",
	}

	for expectedType, expectedPath := range fileLocations {
		testDir := testcommon.CreateTmpDir("TestApptype")

		// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
		defer testcommon.CleanupDir(testDir)
		defer testcommon.Chdir(testDir)()

		err := os.MkdirAll(filepath.Join(testDir, filepath.Dir(expectedPath)), 0777)
		assert.NoError(err)

		_, err = os.OpenFile(filepath.Join(testDir, expectedPath), os.O_RDONLY|os.O_CREATE, 0666)
		assert.NoError(err)

		app, err := ddevapp.NewApp(testDir, ddevapp.DefaultProviderName)
		assert.NoError(err)

		foundType := app.DetectAppType()
		assert.EqualValues(expectedType, foundType)
	}

}

// Tests Drupal 6 apptype detection
func TestDrupal6AppTypeDetection(t *testing.T) {
	assert := asrt.New(t)

	testDir := testcommon.CreateTmpDir("TestApptype")

	// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	// Drupal file comment.
	fileDoc := []byte("<?php\n\n/**\n* @file\n* The PHP page that serves all page requests on a Drupal installation.\n*\n* The routines here dispatch control to the appropriate handler, which then\n* prints the appropriate page.\n*\n* All Drupal code is released under the GNU General Public License.\n* See COPYRIGHT.txt and LICENSE.txt.\n*/")
	err := ioutil.WriteFile(filepath.Join(testDir, "index.php"), fileDoc, 0644)
	assert.NoError(err)

	// Drupal version identifier.
	err = os.MkdirAll(filepath.Join(testDir, filepath.Dir("misc/ahah.js")), 0777)
	assert.NoError(err)
	_, err = os.OpenFile(filepath.Join(testDir, "misc/ahah.js"), os.O_RDONLY|os.O_CREATE, 0666)
	assert.NoError(err)

	app, err := ddevapp.NewApp(testDir, ddevapp.DefaultProviderName)
	assert.NoError(err)

	foundType := app.DetectAppType()
	assert.EqualValues("drupal6", foundType)
}

// Tests Drupal 7 apptype detection
func TestDrupal7AppTypeDetection(t *testing.T) {
	assert := asrt.New(t)

	testDir := testcommon.CreateTmpDir("TestApptype")

	// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	// Drupal file comment.
	fileDoc := []byte("<?php\n\n/**\n* @file\n* The PHP page that serves all page requests on a Drupal installation.\n*\n* The routines here dispatch control to the appropriate handler, which then\n* prints the appropriate page.\n*\n* All Drupal code is released under the GNU General Public License.\n* See COPYRIGHT.txt and LICENSE.txt.\n*/")
	err := ioutil.WriteFile(filepath.Join(testDir, "index.php"), fileDoc, 0644)
	assert.NoError(err)

	app, err := ddevapp.NewApp(testDir, ddevapp.DefaultProviderName)
	assert.NoError(err)

	foundType := app.DetectAppType()
	assert.EqualValues("drupal7", foundType)
}

// Tests Drupal 8 apptype detection
func TestDrupal8AppTypeDetection(t *testing.T) {
	assert := asrt.New(t)

	testDir := testcommon.CreateTmpDir("TestApptype")

	// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	// Drupal file comment.
	fileDoc := []byte("/**\n* @file\n* The PHP page that serves all page requests on a Drupal installation.\n*\n* All Drupal code is released under the GNU General Public License.\n* See COPYRIGHT.txt and LICENSE.txt files in the \"core\" directory.\n*/")
	err := ioutil.WriteFile(filepath.Join(testDir, "index.php"), fileDoc, 0644)
	assert.NoError(err)

	// Drupal version identifier.
	err = os.MkdirAll(filepath.Join(testDir, filepath.Dir("core/composer.json")), 0777)
	assert.NoError(err)
	_, err = os.OpenFile(filepath.Join(testDir, "core/composer.json"), os.O_RDONLY|os.O_CREATE, 0666)
	assert.NoError(err)

	app, err := ddevapp.NewApp(testDir, ddevapp.DefaultProviderName)
	assert.NoError(err)

	foundType := app.DetectAppType()
	assert.EqualValues("drupal8", foundType)
}
