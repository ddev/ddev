package servicetest_test

import (
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
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
			Name:                          "TestServicesDrupal7", // Drupal D7
			SourceURL:                     "https://ftp.drupal.org/files/projects/drupal-7.61.tar.gz",
			ArchiveInternalExtractionPath: "drupal-7.61/",
			FullSiteTarballURL:            "",
			Docroot:                       "",
			Type:                          nodeps.AppTypeDrupal7,
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

			// nolint: errcheck
			defer app.Stop(true, false)

			for _, service := range ServiceFiles {
				confdir := filepath.Join(app.GetAppRoot(), ".ddev")
				err = fileutil.CopyFile(filepath.Join(ServiceDir, service), filepath.Join(confdir, service))
				assert.NoError(err)
			}

			err = app.Start()
			assert.NoError(err)

			// Normally a ddev start would happen, which would create the fully interpreted compose file
			// In a test environment, we recreate app as if ddev start had happened.
			// We need app.Describe to get us real running info.
			app, err = ddevapp.NewApp(app.AppRoot, false, "")
			require.NoError(t, err)

			checkSolrService(t, app)
			checkMemcachedService(t, app)

			desc, err := app.Describe()
			require.NoError(t, err)

			// Make sure desc had 3 services.
			require.Len(t, desc["extra_services"], 3)

			// A volume should have been created for solr (only)
			assert.True(dockerutil.VolumeExists(strings.ToLower("ddev-" + app.Name + "_" + "solr")))

			err = app.Stop(true, false)
			assert.NoError(err)

			// Solr volume should have been deleted
			assert.False(dockerutil.VolumeExists(strings.ToLower("ddev-" + app.Name + "_" + "solr")))

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
	require.NoError(t, err)
	require.NotEmpty(t, container)

	// Ensure container is running
	check, err := testcommon.ContainerCheck(dockerutil.ContainerName(*container), "running")
	assert.NoError(err)
	assert.True(check, "%s container is not running", service)

	// Ensure service is accessible from web container
	checkCommand := fmt.Sprintf("curl -sL -w '%%{http_code}' '%s' -o /dev/null", path)
	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     checkCommand,
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
	require.NoError(t, err)
	require.NotEmpty(t, container)

	// Ensure container is running
	check, err := testcommon.ContainerCheck(dockerutil.ContainerName(*container), "running")
	assert.NoError(err)
	assert.True(check, "%s container is not running", service)

	// Ensure service is accessible from web container
	checkCommand := fmt.Sprintf("echo stats | nc -q 1 %s %s", service, port)

	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     checkCommand,
	})
	assert.NoError(err)
	assert.Contains(out, "STAT pid")
}
