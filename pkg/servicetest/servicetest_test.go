package servicetest_test

import (
	"os"
	"strings"
	"testing"

	"path/filepath"

	"fmt"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	log "github.com/sirupsen/logrus"
	asrt "github.com/stretchr/testify/assert"
)

var (
	TestSites = []testcommon.TestSite{
		{
			Name:                          "TestServicesDrupal8",
			SourceURL:                     "https://ftp.drupal.org/files/projects/drupal-8.5.3.tar.gz",
			ArchiveInternalExtractionPath: "drupal-8.5.3/",
			Docroot:                       "",
			Type:                          ddevapp.AppTypeDrupal8,
		},
	}
	ServiceFiles []string
	ServiceDir   string
)

// TestMain runs the tests in servicetest
func TestMain(m *testing.M) {
	output.LogSetUp()

	if os.Getenv("GOTEST_SHORT") != "" {
		log.Info("servicetest skipped in short mode because GOTEST_SHORT is set")
		os.Exit(0)
	}

	var err error
	ServiceDir, err = filepath.Abs("testdata/services")
	util.CheckErr(err)

	err = filepath.Walk(ServiceDir, func(path string, f os.FileInfo, _ error) error {
		if !f.IsDir() && strings.HasPrefix(f.Name(), "docker-compose") {
			ServiceFiles = append(ServiceFiles, f.Name())
		}
		return nil
	})
	util.CheckErr(err)

	err = dockerutil.EnsureNetwork(dockerutil.GetDockerClient(), dockerutil.NetName)
	util.CheckErr(err)
	log.Debugln("Running tests in servicetest...")
	testRun := m.Run()

	os.Exit(testRun)
}

// TestServices tests each service compose file in the services folder.
// It tests that a site can fully start w/ the compose file present, and
// runs each service's check function to ensure it's accessible from
// the web container.
func TestServices(t *testing.T) {
	assert := asrt.New(t)

	if len(ServiceFiles) > 0 {
		for _, site := range TestSites {
			// If running this with GOTEST_SHORT we have to create the directory, tarball etc.
			if site.Dir == "" || !fileutil.FileExists(site.Dir) {
				err := site.Prepare()
				if err != nil {
					t.Fatalf("Prepare() failed on TestSite.Prepare() site=%s, err=%v", site.Name, err)
				}
			}

			app := &ddevapp.DdevApp{}

			err := app.Init(site.Dir)
			assert.NoError(err)

			for _, service := range ServiceFiles {
				confdir := filepath.Join(app.GetAppRoot(), ".ddev")
				err = fileutil.CopyFile(filepath.Join(ServiceDir, service), filepath.Join(confdir, service))
				assert.NoError(err)
			}

			err = app.Start()
			assert.NoError(err)

			checkSolrService(t, app)
			checkMemcachedService(t, app)

			err = app.Down(true, false)
			assert.NoError(err)
			site.Cleanup()
		}
	}
}

// checkSolrService ensures that the solr service's container is
// running and that the service is accessible from the web container
func checkSolrService(t *testing.T, app *ddevapp.DdevApp) {
	service := "solr"
	port := "8983"
	path := fmt.Sprintf("http://%s:%s/solr/", service, port)

	var err error
	assert := asrt.New(t)
	labels := map[string]string{
		"com.ddev.site-name":         app.GetName(),
		"com.docker.compose.service": service,
	}

	container, err := dockerutil.FindContainerByLabels(labels)
	if err != nil {
		t.Fatalf("Could not find running container for %s service. Skipping remainder of test: %v", service, err)
	}

	// Ensure container is running
	check, err := testcommon.ContainerCheck(dockerutil.ContainerName(container), "running")
	assert.NoError(err)
	assert.True(check, "%s container is not running", service)

	// Ensure service is accessible from web container
	checkCommand := fmt.Sprintf("curl -sL -w '%%{http_code}' '%s' -o /dev/null", path)
	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     []string{"sh", "-c", checkCommand},
	})
	assert.NoError(err, "Unable to make request to http://%s:%s/solr/", service, port)
	assert.Equal("200", out)
}

// checkMemcachedService ensures that the memcached service's
// container is running and that the service is accessible from
// the web container
func checkMemcachedService(t *testing.T, app *ddevapp.DdevApp) {
	service := "memcached"
	port := "11211"

	var err error
	assert := asrt.New(t)
	labels := map[string]string{
		"com.ddev.site-name":         app.GetName(),
		"com.docker.compose.service": service,
	}

	container, err := dockerutil.FindContainerByLabels(labels)
	if err != nil {
		t.Fatalf("Could not find running container for %s service. Skipping remainder of test: %v", service, err)
	}

	// Ensure container is running
	check, err := testcommon.ContainerCheck(dockerutil.ContainerName(container), "running")
	assert.NoError(err)
	assert.True(check, "%s container is not running", service)

	// Ensure service is accessible from web container
	checkCommand := fmt.Sprintf("echo stats | nc -q 1 %s %s", service, port)

	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     []string{"sh", "-c", checkCommand},
	})
	assert.NoError(err)
	assert.Contains(out, "STAT pid")
}
