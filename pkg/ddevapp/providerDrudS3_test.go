package ddevapp_test

import (
	"github.com/Netflix/go-expect"
	"github.com/hinshun/vt10x"
	"os"
	"path/filepath"
	"testing"

	. "github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"os/exec"
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

func TestOutExpect(t *testing.T) {
	assert := asrt.New(t)

	// Multiplex stdin/stdout to a virtual terminal to respond to ANSI escape
	// sequences (i.e. cursor position report).
	c, state, err := vt10x.NewVT10XConsole()
	assert.NoError(err)
	defer c.Close()

	donec := make(chan struct{})
	go func() {
		defer close(donec)
		println("Made it into the gofunc to do expect stuff")
		c.ExpectString("Project name")

		c.SendLine("noname") // Project name

		c.ExpectString("Docroot Location")
		c.SendLine("none") //Docroot location
		c.ExpectString("No directory could be found")
		c.ExpectString("Docroot Location")
		c.SendLine("")
		c.ExpectString("Project Type")
		c.SendLine("junk")
		c.ExpectString("is not a valid project type")
		c.SendLine("drupal7") // Project type
		//c.SendLine("")  // AWS Access key
		//c.SendLine("")  // aws secret

		//c.SendLine("")
		//c.ExpectString("Docroot location")
		//c.SendLine("")
		//c.Send("\x03")
		//c.ExpectString("Nothing that should be there")

		println("Expecting the eof")
		c.ExpectEOF()
		println("Got the eof")
	}()

	cmd := exec.Command("/usr/local/bin/ddev", "config")
	cmd.Stdin = c.Tty()
	cmd.Stdout = c.Tty()
	cmd.Stderr = c.Tty()

	err = cmd.Start()
	assert.NoError(err)
	println("Made it past cmd.Start()")
	//c.SendLine("") // project name
	//c.SendLine("") // docroot
	//c.SendLine("") // project type
	//c.SendLine("") // a spare

	cmd.Wait()
	println("After cmd.wait()")

	// Close the slave end of the pty, and read the remaining bytes from the master end.
	c.Tty().Close()
	<-donec

	// Dump the terminal's screen.
	t.Log(expect.StripTrailingEmptyLines(state.String()))
}

// TestConfigCommand tests the interactive config options.
func TestDrudS3ConfigCommand(t *testing.T) {
	accessKeyID := os.Getenv("DDEV_DRUD_S3_AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("DDEV_DRUD_S3_AWS_SECRET_ACCESS_KEY")
	if accessKeyID == "" || secretAccessKey == "" {
		t.Skip("No DDEV_DRUD_S3_AWS_ACCESS_KEY_ID and  DDEV_DRUD_S3_AWS_SECRET_ACCESS_KEY env vars have been set. Skipping DrudS3 specific test.")
	}

	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir("TestDrudS3ConfigCommand")

	// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	// Create a docroot folder.
	err := os.Mkdir(filepath.Join(testDir, "docroot"), 0644)
	if err != nil {
		t.Errorf("Could not create docroot directory under %s", testDir)
	}

	// Create the ddevapp we'll use for testing.
	//app, err := NewApp(testDir, "drud-s3")
	app, err := NewApp(testDir, "")
	assert.NoError(err)

	// Randomize some values to use for Stdin during testing.
	//docroot := "docroot"
	//invalidName := strings.ToLower(util.RandString(16))
	//invalidEnvironment := strings.ToLower(util.RandString(8))

	c, state, err := vt10x.NewVT10XConsole()
	assert.NoError(err)
	defer c.Close()

	donec := make(chan struct{})
	go func() {
		defer close(donec)

		_, _ = c.ExpectString("Project name")
		_, _ = c.SendLine("somejunkproject")
		_, _ = c.SendLine("")

		c.ExpectString("Project Type")
		_, _ = c.SendLine("drupal7")
		//_, _ = c.ExpectString("AWS secret access key:")
		//_, _ = c.SendLine(secretAccessKey)
		//_, _ = c.ExpectString("AWS S3 Bucket Name:")
		//_, _ = c.SendLine(DrudS3TestBucket)
		//_, _ = c.ExpectString("Choose an environment to pull from:")
		//_, _ = c.SendLine(DrudS3TestEnvName)
		//_, _ = c.ExpectString("Configuration complete.")
		_, _ = c.ExpectEOF()
	}()

	err = app.PromptForConfig()
	assert.NoError(err)

	// Close the slave end of the pty, and read the remaining bytes from the master end.
	c.Tty().Close()
	<-donec

	// Dump the terminal's screen.
	t.Log(expect.StripTrailingEmptyLines(state.String()) + "\n")

	/**
	 * Do a full interactive configuration for a DrudS3 environment.
	 *
	 * 1. Provide an invalid site name, ensure there is an error.
	 * 2. Provide a valid site name. Ensure there is no error.
	 * 3. Provide a valid docroot (already tested elsewhere)
	 * 4. Provide a valid app type (drupal8)
	 * 5. Provide an invalid DrudS3 environment name, ensure an error is triggered.
	 * 6. Provide a valid environment name.
	 **/

	//input := fmt.Sprintf("%s\n%s\n%s\ndocroot\ndrupal8\n%s\n%s", invalidName, DrudS3TestSiteName, docroot, invalidEnvironment, DrudS3TestEnvName)
	//scanner := bufio.NewScanner(strings.NewReader(input))
	//util.SetInputScanner(scanner)
	//
	//restoreOutput := testcommon.CaptureUserOut()
	//err = app.PromptForConfig()
	//assert.NoError(err, t)
	//out := restoreOutput()
	//
	//// Get the provider interface and ensure it validates.
	//provider, err := app.GetProvider()
	//assert.NoError(err)
	//err = provider.Validate()
	//assert.NoError(err)
	//
	//// Ensure we have expected string values in output.
	//assert.Contains(out, testDir)
	//assert.Contains(out, fmt.Sprintf("could not find a DrudS3 site named %s", invalidName))
	//assert.Contains(out, fmt.Sprintf("could not find an environment named '%s'", invalidEnvironment))
	//
	//// Ensure values were properly set on the app struct.
	//assert.Equal(DrudS3TestSiteName, app.Name)
	//assert.Equal("drupal8", app.Type)
	//assert.Equal("docroot", app.Docroot)
	//err = PrepDdevDirectory(testDir)
	//assert.NoError(err)
}

// TestDrudS3ValidDownloadObjects ensures we can find download objects from DrudS3 for a configured environment.
func TestDrudS3ValidDownloadObjects(t *testing.T) {
	//if os.Getenv("DDEV_DRUD_S3_AWS_ACCESS_KEY_ID") == "" || os.Getenv("DDEV_DRUD_S3_AWS_ACCESS_KEY_ID") == "" {
	//	t.Skip("No DDEV_DRUD_S3_AWS_ACCESS_KEY_ID and  DDEV_DRUD_S3_AWS_ACCESS_KEY_ID env vars have been set. Skipping DrudS3 specific test.")
	//}
	//
	//// Set up tests and give ourselves a working directory.
	//assert := asrt.New(t)
	//testDir := testcommon.CreateTmpDir("TestDrudS3ValidDownloadObjects")
	//
	//// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
	//defer testcommon.CleanupDir(testDir)
	//defer testcommon.Chdir(testDir)()
	//
	//app, err := NewApp(testDir, "DrudS3")
	//assert.NoError(err)
	//app.Name = DrudS3TestSiteName
	//
	//provider := DrudS3Provider{}
	//err = provider.Init(app)
	//assert.NoError(err)
	//
	//provider.Sitename = DrudS3TestSiteName
	//provider.EnvironmentName = DrudS3TestEnvName
	//
	//// Ensure GetBackup triggers an error for unknown backup types.
	//_, _, err = provider.GetBackup(util.RandString(8))
	//assert.Error(err)
	//
	//// Ensure we can get a
	//backupLink, importPath, err := provider.GetBackup("database")
	//
	//assert.Equal(importPath, "")
	//assert.Contains(backupLink, "database.sql.gz")
	//assert.NoError(err)
}

// TestDrudS3Pull ensures we can pull backups from DrudS3 for a configured environment.
func TestDrudS3Pull(t *testing.T) {
	//	if os.Getenv("DDEV_DRUD_S3_AWS_ACCESS_KEY_ID") == "" || os.Getenv("DDEV_DRUD_S3_AWS_ACCESS_KEY_ID") == "" {
	//		t.Skip("No DDEV_DRUD_S3_AWS_ACCESS_KEY_ID and  DDEV_DRUD_S3_AWS_ACCESS_KEY_ID env vars have been set. Skipping DrudS3 specific test.")
	//	}
	//
	//	// Set up tests and give ourselves a working directory.
	//	assert := asrt.New(t)
	//	testDir := testcommon.CreateTmpDir("TestDrudS3Pull")
	//
	//	// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
	//	defer testcommon.CleanupDir(testDir)
	//	defer testcommon.Chdir(testDir)()
	//
	//	// Move into the properly named DrudS3 site (must match DrudS3 sitename)
	//	siteDir := testDir + "/" + DrudS3TestSiteName
	//	err := os.MkdirAll(siteDir+"/sites/default", 0777)
	//	assert.NoError(err)
	//	err = os.Chdir(siteDir)
	//	assert.NoError(err)
	//
	//	app, err := NewApp(siteDir, "DrudS3")
	//	assert.NoError(err)
	//	app.Name = DrudS3TestSiteName
	//	app.Type = "drupal8"
	//	err = app.WriteConfig()
	//	assert.NoError(err)
	//
	//	testcommon.ClearDockerEnv()
	//
	//	p := DrudS3Provider{}
	//	err = p.Init(app)
	//	assert.NoError(err)
	//
	//	p(name) = DrudS3TestSiteName
	//	p.EnvironmentName = DrudS3TestEnvName
	//	err = p.Write(app.GetConfigPath("import.yaml"))
	//	assert.NoError(err)
	//
	//	// Ensure we can do a pull on the configured site.
	//	app, err = GetActiveApp("")
	//	assert.NoError(err)
	//	err = app.Import()
	//	assert.NoError(err)
	//	err = app.Down(true)
	//	assert.NoError(err)
}
