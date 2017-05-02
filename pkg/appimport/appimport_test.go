package appimport

import (
	"fmt"
	"path"
	"testing"

	"strings"

	"os"

	"log"

	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/drud/drud-go/utils/system"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/assert"
)

const netName = "ddev_default"

var (
	testArchiveURL  = "https://github.com/drud/wordpress/releases/download/v0.4.0/db.tar.gz"
	testArchivePath = path.Join(testcommon.CreateTmpDir("appimport"), "db.tar.gz")
	composePath     = path.Join("testing", "db-compose.yaml")
	cwd             string
)

func TestMain(m *testing.M) {
	err := system.DownloadFile(testArchivePath, testArchiveURL)
	if err != nil {
		log.Fatalf("archive download failed: %s", err)
	}

	err = os.Mkdir(path.Join("testing", "data"), 0755)
	util.CheckErr(err)

	err = util.Untar(testArchivePath, path.Join("testing", "data"))
	if err != nil {
		log.Fatalf("archive extraction failed: %s", err)
	}

	cwd, err = os.Getwd()
	if err != nil {
		log.Fatalf("failed to get cwd: %s", err)
	}

	// ensure we have docker network
	client, err := dockerutil.GetDockerClient()
	util.CheckErr(err)
	err = util.EnsureNetwork(client, "ddev_default")
	util.CheckErr(err)

	// prep db container for import testing
	dbimg := fmt.Sprintf("%s:%s", version.DBImg, version.DBTag)
	err = os.Setenv("DDEV_DBIMAGE", dbimg)
	util.CheckErr(err)

	err = dockerutil.DockerCompose("-f", composePath, "up", "-d")
	if err != nil {
		log.Fatalf("failed to start db container: %s", err)
	}

	fmt.Println("Running tests.")
	testRun := m.Run()

	err = dockerutil.DockerCompose("-f", composePath, "down")
	if err != nil {
		log.Fatalf("failed to remove db container: %s", err)
	}

	err = os.RemoveAll(path.Dir(testArchivePath))
	util.CheckErr(err)
	err = os.RemoveAll(path.Join("testing", "data"))
	util.CheckErr(err)

	os.Exit(testRun)
}

// TestValidateAsset tests validation of asset paths.
func TestValidateAsset(t *testing.T) {
	assert := assert.New(t)

	// test tilde expansion
	userDir, err := homedir.Dir()
	testDir := path.Join(userDir, "testpath")
	assert.NoError(err)
	err = os.Mkdir(testDir, 0755)
	assert.NoError(err)

	testPath, err := ValidateAsset("~/testpath", "files")
	assert.NoError(err)
	assert.Contains(testPath, userDir)
	assert.False(strings.Contains(testPath, "~"))
	err = os.Remove(testDir)
	assert.NoError(err)

	// test a relative path
	testPath, err = ValidateAsset("../../vendor", "files")
	assert.NoError(err)
	upTwo := strings.TrimSuffix(cwd, "/pkg/appimport")
	assert.Contains(testPath, upTwo)

	// archive
	_, err = ValidateAsset(testArchivePath, "db")
	assert.Error(err)
	assert.Equal(err.Error(), "is archive")

	// db no sql
	_, err = ValidateAsset("appimport.go", "db")
	assert.Contains(err.Error(), "provided path is not a .sql file or archive")
	assert.Error(err)

	// files not a directory
	_, err = ValidateAsset("appimport.go", "files")
	assert.Error(err)
	assert.Contains(err.Error(), "provided path is not a directory or archive")
}

// TestImportSQLDump tests import of db to container.
func TestImportSQLDump(t *testing.T) {
	assert := assert.New(t)

	// test container is not running
	err := ImportSQLDump("invalid")
	assert.Error(err)
	assert.Contains(err.Error(), "container is not currently running")

	// test import
	labels := map[string]string{
		"com.ddev.site-name":      "test",
		"com.ddev.container-type": "db",
	}

	err = util.ContainerWait(90, labels)
	assert.NoError(err)

	err = ImportSQLDump("local-test-db")
	assert.NoError(err)
}
