package ddevapp_test

import (
	"testing"

	"os"

	"io/ioutil"

	. "github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	"github.com/stretchr/testify/assert"
)

// TestWriteDrupalSettings tests writing drupal settings.php/settings.local.php
func TestWriteDrupalSettings(t *testing.T) {
	dir := testcommon.CreateTmpDir("example")
	err := os.MkdirAll(dir+"/sites/default", 0777)
	assert.NoError(t, err)

	app, err := NewApp(dir, DefaultProviderName)
	assert.NoError(t, err)
	app.Type = "drupal8"

	err = CreateSettingsFile(app)
	assert.NoError(t, err)

	err = os.RemoveAll(dir)
	assert.NoError(t, err)
}

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

func TestWriteWordpressSettings(t *testing.T) {
	dir := testcommon.CreateTmpDir("example")

	file, err := ioutil.TempFile(dir, "file")
	assert.NoError(t, err)

	wpConfig := NewWordpressConfig()
	err = WriteWordpressConfig(wpConfig, file.Name())
	assert.NoError(t, err)

	util.CheckClose(file)
	err = os.Chmod(dir, 0755)
	assert.NoError(t, err)
	err = os.Chmod(file.Name(), 0666)
	assert.NoError(t, err)

	err = os.RemoveAll(dir)
	assert.NoError(t, err)
}
