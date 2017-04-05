package appimport

import (
	"fmt"
	"path"
	"testing"

	"strings"

	"os"

	"log"

	"github.com/drud/ddev/pkg/version"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/drud/drud-go/utils/system"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/assert"
)

var (
	testArchiveURL  = "https://github.com/drud/wordpress/releases/download/v0.1.0/db.tar.gz"
	testArchivePath = path.Join(os.TempDir(), "db.tar.gz")
	temp            = os.TempDir()
	composePath     = path.Join(".ddev", "docker-compose.yaml")
	cwd             string
)

func TestMain(m *testing.M) {
	err := system.DownloadFile(testArchivePath, testArchiveURL)
	if err != nil {
		log.Fatalf("archive download failed: %s", err)
	}

	cwd, err = os.Getwd()
	if err != nil {
		log.Fatalf("failed to get cwd: %s", err)
	}

	// prep db container for import testing
	dbimg := fmt.Sprintf("%s:%s", version.DBImg, version.DBTag)
	os.Setenv("DDEV_DBIMAGE", dbimg)
	err = os.MkdirAll(path.Join(".ddev", "data"), 0755)
	if err != nil {
		log.Fatalf("failed to make dir: %s", err)
	}
	CopyFile(path.Join("testing", "db-compose.yaml"), composePath)
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

	os.Remove(testArchivePath)
	os.RemoveAll(path.Join(temp, "extract"))
	os.RemoveAll(".ddev")

	os.Exit(testRun)
}

// TestExtractArchive tests extraction of an archive.
func TestExtractArchive(t *testing.T) {
	assert := assert.New(t)

	// test bad archive
	_, err := extractArchive("appimport.go")
	assert.Error(err)
	msg := fmt.Sprintf("%v", err)
	assert.Contains(msg, "Unable to extract archive")
	err = os.RemoveAll(path.Join(temp, "extract"))
	assert.NoError(err)

	// test good archive
	extract, err := extractArchive(testArchivePath)
	assert.NoError(err)
	assert.Contains(extract, temp)

	os.RemoveAll(extract)
}

// TestCopyFile tests copying a file.
func TestCopyFile(t *testing.T) {
	assert := assert.New(t)
	dest := path.Join(temp, "test.tar.gz")

	err := os.Chmod(testArchivePath, 0666)
	assert.NoError(err)

	err = CopyFile(testArchivePath, dest)
	assert.NoError(err)

	file, err := os.Stat(dest)
	assert.NoError(err)
	assert.Equal(int(file.Mode()), 0666)

	os.RemoveAll(dest)
}

// TestCopyDir tests copying a directory.
func TestCopyDir(t *testing.T) {
	assert := assert.New(t)
	dest := path.Join(temp, "copy")
	os.Mkdir(dest, 0755)

	// test source not a directory
	err := CopyDir("appimport.go", temp)
	assert.Error(err)
	msg := fmt.Sprintf("%v", err)
	assert.Contains(msg, "source is not a directory")

	// test destination exists
	err = CopyDir(temp, cwd)
	assert.Error(err)
	msg = fmt.Sprintf("%v", err)
	assert.Contains(msg, "destination already exists")
	os.RemoveAll(dest)

	// copy a directory.
	err = CopyDir(cwd, dest)
	assert.NoError(err)
	assert.True(system.FileExists(path.Join(dest, "appimport.go")))
	assert.True(system.FileExists(path.Join(dest, "appimport_test.go")))

	os.RemoveAll(dest)
}

// TestFindFileExt tests search function for files w/ matching extension.
func TestFindFileExt(t *testing.T) {
	assert := assert.New(t)

	// test against cwd
	match, err := findFileByExtension(cwd, ".go")
	assert.NoError(err)
	assert.True(len(match) > 1)

	// test no matching files
	_, err = findFileByExtension(cwd, ".sql")
	msg := fmt.Sprintf("%v", err)
	assert.Contains(msg, "no .sql files found in")
	assert.Error(err)
}

// TestValidateAsset tests validation of asset paths.
func TestValidateAsset(t *testing.T) {
	assert := assert.New(t)

	// test tilde expansion
	userDir, err := homedir.Dir()
	testDir := path.Join(userDir, "testpath")
	assert.NoError(err)
	os.Mkdir(testDir, 0755)

	testPath, err := ValidateAsset("~/testpath", "files")
	assert.NoError(err)
	assert.Contains(testPath, userDir)
	assert.False(strings.Contains(testPath, "~"))
	os.Remove(testDir)

	// test a relative path
	testPath, err = ValidateAsset("../../vendor", "files")
	assert.NoError(err)
	upTwo := strings.TrimSuffix(cwd, "/pkg/appimport")
	assert.Contains(testPath, upTwo)

	// trigger extraction
	testPath, err = ValidateAsset(testArchivePath, "db")
	fmt.Printf(testPath)
	assert.NoError(err)
	assert.Contains(testPath, temp)

	// fail to find sql
	_, err = ValidateAsset("../../vendor", "db")
	msg := fmt.Sprintf("%v", err)
	assert.Contains(msg, "no .sql files found")
	assert.Error(err)

	// files not a directory
	_, err = ValidateAsset("appimport.go", "files")
	assert.Error(err)
	msg = fmt.Sprintf("%v", err)
	assert.Contains(msg, "provided path is not a directory or archive")
}

// TestImportSQLDump tests import of db to container.
func TestImportSQLDump(t *testing.T) {
	assert := assert.New(t)
	importFile := path.Join(temp, "extract", "wp-db.sql")

	// test no sql dump provided
	err := ImportSQLDump("appimport.go", temp, "invalid")
	assert.Error(err)
	msg := fmt.Sprintf("%v", err)
	assert.Contains(msg, "a database dump in .sql format must be provided")

	// test container is not running
	err = ImportSQLDump(importFile, temp, "invalid")
	assert.Error(err)
	msg = fmt.Sprintf("%v", err)
	assert.Contains(msg, "container is not currently running")

	// test import
	// err = ImportSQLDump(importFile, cwd, "local-test-db")
	// assert.NoError(err)
}
