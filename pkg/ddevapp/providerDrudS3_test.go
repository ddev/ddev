package ddevapp_test

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
)

/**
 * These tests rely on an external AWS account managed by DRUD. To run them, you'll
 * need to set the environment variables called "DDEV_DRUD_S3_AWS_ACCESS_KEY_ID" and
 * "DDEV_DRUD_S3_AWS_SECRET_ACCESS_KEY" with credentials for this account. If no such
 * environment variable is present, these tests will be skipped.
 *
 * A valid site (with backups) must be present which matches the test site and environment name
 * defined in the constants below.
 */

const DrudS3TestSiteName = "d7-kickstart"
const DrudS3TestEnvName = "production"
const DrudS3TestBucket = "ddev-local-tests"

// TODO: We need to actually test app.PromptForConfig(), but haven't succeeded in doing it
// (Problems with terminal emulation and survey.) We absolutely want to test the text prompts,
// but have not succeeded using Survey's go-expect technique nor capture std as pantheon tests do.

// TestDrudS3ValidDownloadObjects ensures we can find download objects from DrudS3 for a configured environment.
// Tests actual pull as well.
func TestDrudS3ValidDownloadObjects(t *testing.T) {
	accessKeyID := os.Getenv("DDEV_DRUD_S3_AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("DDEV_DRUD_S3_AWS_SECRET_ACCESS_KEY")
	if accessKeyID == "" || secretAccessKey == "" {
		t.Skip("No DDEV_DRUD_S3_AWS_ACCESS_KEY_ID and  DDEV_DRUD_S3_AWS_SECRET_ACCESS_KEY env vars have been set. Skipping DrudS3 specific test.")
	}

	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir("TestDrudS3ValidDownloadObjects")

	// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()
	err := os.MkdirAll("sites/default/files", 0777)
	assert.NoError(err)

	provider := ddevapp.DrudS3Provider{
		ProviderType:    "drud-s3",
		AWSSecretKey:    secretAccessKey,
		AWSAccessKey:    accessKeyID,
		S3Bucket:        DrudS3TestBucket,
		EnvironmentName: DrudS3TestEnvName,
	}

	app, err := ddevapp.NewApp(testDir, "drud-s3")
	assert.NoError(err)
	app.Name = DrudS3TestSiteName
	app.Type = "drupal7"

	err = provider.Init(app)
	assert.NoError(err)
	// Write the provider config
	err = provider.Write(app.GetConfigPath("import.yaml"))
	assert.NoError(err)

	err = app.WriteConfig()
	assert.NoError(err)
	err = app.Init(testDir)
	assert.NoError(err)

	// Ensure we can get a db backup on the happy path.
	backupLink, importPath, err := provider.GetBackup("database")
	assert.NoError(err)
	assert.Equal(importPath, "")
	assert.True(strings.HasSuffix(backupLink, "sql.gz"))

	// Ensure we can do a pull on the configured site.
	app, err = ddevapp.GetActiveApp("")
	assert.NoError(err)
	err = app.Import()
	assert.NoError(err)
	err = app.Down(true)
	assert.NoError(err)

	// Make sure invalid access key gets correct behavior
	provider.AWSAccessKey = "AKIAIBSTOTALLYINVALID"
	backupLink, importPath, err = provider.GetBackup("database")
	assert.Error(err)
	assert.Contains(err.Error(), "InvalidAccessKeyId")

	// Make sure invalid secret key gets correct behavior
	provider.AWSAccessKey = accessKeyID
	provider.AWSSecretKey = "rweeHGZ5totallyinvalidsecretkey"
	backupLink, importPath, err = provider.GetBackup("database")
	assert.Error(err)
	assert.Contains(err.Error(), "SignatureDoesNotMatch")

	// Make sure bad environment gets correct behavior.
	provider.AWSSecretKey = secretAccessKey
	provider.EnvironmentName = "someInvalidUnknownEnvironment"
	backupLink, importPath, err = provider.GetBackup("database")
	assert.Error(err)
	assert.Contains(err.Error(), "could not find an environment")

	// Make sure bad bucket gets correct behavior.
	provider.S3Bucket = DrudS3TestBucket
	provider.S3Bucket = "someInvalidUnknownBucket"
	backupLink, importPath, err = provider.GetBackup("database")
	assert.Error(err)
	assert.Contains(err.Error(), "NoSuchBucket")

}
