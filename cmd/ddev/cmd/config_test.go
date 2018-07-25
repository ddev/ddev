package cmd

import (
	"testing"

	"github.com/Netflix/go-expect"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/hinshun/vt10x"
	asrt "github.com/stretchr/testify/assert"
	"os"
	osexec "os/exec"
	"path/filepath"
)

// TestConfigDescribeLocation tries out the --show-config-location flag.
func TestConfigDescribeLocation(t *testing.T) {
	assert := asrt.New(t)

	// Create a temporary directory and switch to it.
	tmpdir := testcommon.CreateTmpDir("config-show-location")
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()

	// Create a config
	args := []string{"config", "--docroot=."}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Found a php codebase")

	// Now see if we can detect it
	args = []string{"config", "--show-config-location"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(string(out), tmpdir)

	// Now try it in a directory that doesn't have a config
	tmpdir = testcommon.CreateTmpDir("config_show_location")
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()

	args = []string{"config", "--show-config-location"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "No project configuration currently exists")

}

// TestConfigWithSitenameFlagDetectsDocroot tests docroot detected when flags passed.
func TestConfigWithSitenameFlagDetectsDocroot(t *testing.T) {
	assert := asrt.New(t)

	// Create a temporary directory and switch to it.
	testDocrootName := "web"
	tmpdir := testcommon.CreateTmpDir("config-with-sitename")
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()
	// Create a document root folder.
	err := os.MkdirAll(filepath.Join(tmpdir, testDocrootName), 0755)
	if err != nil {
		t.Errorf("Could not create %s directory under %s", testDocrootName, tmpdir)
	}
	err = os.MkdirAll(filepath.Join(tmpdir, testDocrootName, "sites", "default"), 0755)
	assert.NoError(err)
	_, err = os.OpenFile(filepath.Join(tmpdir, testDocrootName, "index.php"), os.O_RDONLY|os.O_CREATE, 0666)
	assert.NoError(err)

	expectedPath := "web/core/scripts/drupal.sh"
	err = os.MkdirAll(filepath.Join(tmpdir, filepath.Dir(expectedPath)), 0777)
	assert.NoError(err)

	_, err = os.OpenFile(filepath.Join(tmpdir, expectedPath), os.O_RDONLY|os.O_CREATE, 0666)
	assert.NoError(err)

	// Create a config
	args := []string{"config", "--sitename=config-with-sitename"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Found a drupal8 codebase")
}

const DrudS3TestSiteName = "d7-kickstart"
const DrudS3TestEnvName = "production"
const DrudS3TestBucket = "ddev-local-tests"

// TestConfigDrudS3 runs through the ddev config drud-s3 interactive command
func TestConfigDrudS3(t *testing.T) {
	assert := asrt.New(t)

	accessKeyID := os.Getenv("DDEV_DRUD_S3_AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("DDEV_DRUD_S3_AWS_SECRET_ACCESS_KEY")
	if accessKeyID == "" || secretAccessKey == "" {
		t.Skip("No DDEV_DRUD_S3_AWS_ACCESS_KEY_ID and  DDEV_DRUD_S3_AWS_SECRET_ACCESS_KEY env vars have been set. Skipping DrudS3 specific test.")
	}

	// Create a temporary directory and switch to it.
	tmpdir := testcommon.CreateTmpDir("TestConfigDrudS3")
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()

	// Multiplex stdin/stdout to a virtual terminal to respond to ANSI escape
	// sequences (i.e. cursor position report).
	c, state, err := vt10x.NewVT10XConsole()
	assert.NoError(err)
	defer c.Close()

	donec := make(chan struct{})
	go func(state2 *vt10x.State) {
		defer close(donec)
		testcommon.ExpectString(c, state2, "Project name")
		c.SendLine(DrudS3TestSiteName) // Project name

		testcommon.ExpectString(c, state2, "Docroot Location")
		c.SendLine("thereisnodocroot") //Docroot location
		testcommon.ExpectString(c, state2, "No directory could be found")
		testcommon.ExpectString(c, state2, "Docroot Location")
		c.SendLine(".")
		testcommon.ExpectString(c, state2, "Project Type")
		c.SendLine("junk")
		testcommon.ExpectString(c, state2, "is not a valid project type")
		testcommon.ExpectString(c, state2, "Project Type")
		c.SendLine("drupal7") // Project type
		testcommon.ExpectString(c, state2, "AWS access key id")
		c.SendLine(accessKeyID)
		testcommon.ExpectString(c, state2, "AWS secret access key")
		c.SendLine(secretAccessKey)
		testcommon.ExpectString(c, state2, "AWS S3 Bucket Name")
		c.SendLine(DrudS3TestBucket)
		testcommon.ExpectString(c, state2, "Choose an environment")
		c.SendLine(DrudS3TestEnvName)
		//c.SendLine("")  // AWS Access key
		//c.SendLine("")  // aws secret

		//c.SendLine("")
		//testcommon.ExpectString(c, "Docroot location")
		//c.SendLine("")
		//c.Send("\x03")
		//testcommon.ExpectString(c, "Nothing that should be there")

		//c.ExpectEOF()
		println("Got the eof")
	}(state)

	dcmd := osexec.Command(DdevBin, "config", "drud-s3")

	dcmd.Stdin = c.Tty()
	dcmd.Stdout = c.Tty()
	dcmd.Stderr = c.Tty()

	err = dcmd.Run()
	assert.NoError(err)
	println("Made it past dcmd.Start()")
	//c.SendLine("") // project name
	//c.SendLine("") // docroot
	//c.SendLine("") // project type
	//c.SendLine("") // a spare

	//dcmd.Wait()
	println("After dcmd.wait()")

	// Close the slave end of the pty, and read the remaining bytes from the master end.
	c.Tty().Close()
	<-donec

	// Dump the terminal's screen.
	t.Log(expect.StripTrailingEmptyLines(state.String()))
}
