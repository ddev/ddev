package config

import (
	"testing"

	"os"

	"io/ioutil"

	"github.com/drud/bootstrap/cli/cms/model"
	"github.com/stretchr/testify/assert"
)

func TestWriteDrupalConfig(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.NoError(t, err)

	file, err := ioutil.TempFile(dir, "file")
	assert.NoError(t, err)

	os.Chmod(dir, 0555)
	os.Chmod(file.Name(), 0444)

	drupalConfig := model.NewDrupalConfig()
	err = WriteDrupalConfig(drupalConfig, file.Name())
	assert.NoError(t, err)

	defer os.RemoveAll(dir)
}

func TestWriteDrushConfig(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.NoError(t, err)

	file, err := ioutil.TempFile(dir, "file")
	assert.NoError(t, err)

	os.Chmod(dir, 0555)
	os.Chmod(file.Name(), 0444)

	drushConfig := model.NewDrushConfig()
	err = WriteDrushConfig(drushConfig, file.Name())
	assert.NoError(t, err)

	defer os.RemoveAll(dir)
}

func TestWriteWordpressConfig(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.NoError(t, err)

	file, err := ioutil.TempFile(dir, "file")
	assert.NoError(t, err)

	os.Chmod(dir, 0555)
	os.Chmod(file.Name(), 0444)

	wpConfig := model.NewWordpressConfig()
	err = WriteWordpressConfig(wpConfig, file.Name())
	assert.NoError(t, err)

	defer os.RemoveAll(dir)
}
