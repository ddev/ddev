package config

import (
	"testing"

	"os"

	"io/ioutil"

	"github.com/drud/ddev/pkg/cms/model"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestWriteDrupalConfig(t *testing.T) {
	dir := testcommon.CreateTmpDir("example")

	file, err := ioutil.TempFile(dir, "file")
	assert.NoError(t, err)

	util.CheckErr(err)
	err = os.Chmod(file.Name(), 0444)
	util.CheckErr(err)

	drupalConfig := model.NewDrupalConfig()
	err = WriteDrupalConfig(drupalConfig, file.Name())
	assert.NoError(t, err)

	defer func() {
		err := os.RemoveAll(dir)
		util.CheckErr(err)
	}()
}

func TestWriteDrushConfig(t *testing.T) {
	dir := testcommon.CreateTmpDir("example")

	file, err := ioutil.TempFile(dir, "file")
	assert.NoError(t, err)

	util.CheckErr(err)
	err = os.Chmod(file.Name(), 0444)
	util.CheckErr(err)

	drushConfig := model.NewDrushConfig()
	err = WriteDrushConfig(drushConfig, file.Name())
	assert.NoError(t, err)

	defer func() {
		err := os.RemoveAll(dir)
		util.CheckErr(err)
	}()
}

func TestWriteWordpressConfig(t *testing.T) {
	dir := testcommon.CreateTmpDir("example")

	file, err := ioutil.TempFile(dir, "file")
	assert.NoError(t, err)

	util.CheckErr(err)
	err = os.Chmod(file.Name(), 0444)
	util.CheckErr(err)

	wpConfig := model.NewWordpressConfig()
	err = WriteWordpressConfig(wpConfig, file.Name())
	assert.NoError(t, err)

	defer func() {
		err := os.RemoveAll(dir)
		util.CheckErr(err)
	}()
}
