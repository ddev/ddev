package ddevapp_test

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
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

const drudS3TestSiteName = "d7-kickstart"
const drudS3TestEnvName = "production"
const drudS3TestBucket = "ddev-local-tests"

var drudS3AccessKeyID = os.Getenv("DDEV_DRUD_S3_AWS_ACCESS_KEY_ID")
var drudS3SecretAccessKey = os.Getenv("DDEV_DRUD_S3_AWS_SECRET_ACCESS_KEY")

// TestDrudS3ConfigCommand tests the interactive config options.
func TestDrudS3ConfigCommand(t *testing.T) {
	if drudS3AccessKeyID == "" || drudS3SecretAccessKey == "" {
		t.Skip("No DDEV_DRUD_S3_AWS_ACCESS_KEY_ID and  DDEV_DRUD_S3_AWS_SECRET_ACCESS_KEY env vars have been set. Skipping DrudS3 specific test.")
	}

	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir("TestDrudS3ValidDownloadObjects")

	// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
	//defer testcommon.CleanupDir(testDir) REMEMBER TO PUT THIS BACK
	defer testcommon.Chdir(testDir)()

	// Create the app we'll use for testing.
	app, err := ddevapp.NewApp(testDir, "drud-s3")
	assert.NoError(err)

	// Attempt config with the whole config setup, including access keys.
	input := fmt.Sprintf("%s\n\n\n%s\n%s\n%s\n%s\n\n\n", drudS3TestSiteName, drudS3AccessKeyID, drudS3SecretAccessKey, drudS3TestBucket, drudS3TestEnvName)

	scanner := bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)

	restoreOutput := testcommon.CaptureUserOut()
	err = app.PromptForConfig()
	assert.NoError(err)
	_ = restoreOutput()

	// Get the provider interface and ensure it validates.
	provider, err := app.GetProvider()

	assert.NoError(err)
	err = provider.Validate()
	assert.NoError(err)

	assertEqualProviderValues(t, provider)

	// Now try the same thing again, but at this point we have established the AWS keys
	// so it shouldn't ask for those any more.
	input = fmt.Sprintf("%s\n\n\n%s\n%s\n", drudS3TestSiteName, drudS3TestBucket, drudS3TestEnvName)

	scanner = bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)

	restoreOutput = testcommon.CaptureUserOut()
	err = app.PromptForConfig()
	assert.NoError(err)
	_ = restoreOutput()

	// Get the provider interface and ensure it validates.
	provider, err = app.GetProvider()

	assert.NoError(err)
	err = provider.Validate()
	assert.NoError(err)

	assertEqualProviderValues(t, provider)

	// Now try with an invalid bucket name, should fail
	input = fmt.Sprintf("%s\n\n\n%s\n%s\n", drudS3TestSiteName, "InvalidTestBucket", drudS3TestEnvName)

	scanner = bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)

	restoreOutput = testcommon.CaptureUserOut()
	err = app.PromptForConfig()
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "NoSuchBucket")
	}
	_ = restoreOutput()

	// Now try with an invalid environment name, should fail
	input = fmt.Sprintf("%s\n\n\n%s\n%s\n", drudS3TestSiteName, drudS3TestBucket, "invalidEnvironmentName")

	scanner = bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)

	restoreOutput = testcommon.CaptureUserOut()
	err = app.PromptForConfig()
	assert.NoError(err)

	_ = restoreOutput()
	println() // Just lets goland find the PASS
	provider, err = app.GetProvider()
	assert.NoError(err)
	err = provider.Validate()
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "could not find an environment with backups")
	}
}

// assertEqualProviderValues is just a helper function to avoid repeating assertions.
func assertEqualProviderValues(t *testing.T, provider ddevapp.Provider) {
	assert := asrt.New(t)

	p := provider.(*ddevapp.DrudS3Provider)

	assert.EqualValues(p.EnvironmentName, drudS3TestEnvName)
	assert.EqualValues(p.AWSSecretKey, drudS3SecretAccessKey)
	assert.EqualValues(p.AWSAccessKey, drudS3AccessKeyID)
	assert.EqualValues(p.EnvironmentName, drudS3TestEnvName)
	assert.EqualValues(p.S3Bucket, drudS3TestBucket)
}

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
		S3Bucket:        drudS3TestBucket,
		EnvironmentName: drudS3TestEnvName,
	}

	app, err := ddevapp.NewApp(testDir, "drud-s3")
	assert.NoError(err)
	app.Name = drudS3TestSiteName
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
	err = app.Import(&ddevapp.ImportOptions{})
	assert.NoError(err)
	err = app.Down(true, false)
	assert.NoError(err)

	// Make sure invalid access key gets correct behavior
	provider.AWSAccessKey = "AKIAIBSTOTALLYINVALID"
	_, _, err = provider.GetBackup("database")
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "InvalidAccessKeyId")
	}

	// Make sure invalid secret key gets correct behavior
	provider.AWSAccessKey = accessKeyID
	provider.AWSSecretKey = "rweeHGZ5totallyinvalidsecretkey"
	_, _, err = provider.GetBackup("database")
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "SignatureDoesNotMatch")
	}

	// Make sure bad environment gets correct behavior.
	provider.AWSSecretKey = secretAccessKey
	provider.EnvironmentName = "someInvalidUnknownEnvironment"
	_, _, err = provider.GetBackup("database")
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "could not find an environment")
	}

	// Make sure bad bucket gets correct behavior.
	provider.S3Bucket = drudS3TestBucket
	provider.S3Bucket = "someInvalidUnknownBucket"
	_, _, err = provider.GetBackup("database")
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "NoSuchBucket")
	}
}
