package servicetest_test

import (
	"os"
	"strings"
	"testing"

	"path/filepath"

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
			SourceURL:                     "https://github.com/drud/drupal8/archive/v0.6.0.tar.gz",
			ArchiveInternalExtractionPath: "drupal8-0.6.0/",
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
	ServiceDir, err = filepath.Abs("../../services")
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
// checks that any exposed HTTP ports return 200.
func TestServices(t *testing.T) {
	assert := asrt.New(t)

	if len(ServiceFiles) > 0 {
		for _, site := range TestSites {
			err := site.Prepare()
			if err != nil {
				t.Fatalf("Prepare() failed on TestSite.Prepare() for site=%s, err=%v", site.Name, err)
			}

			app := &ddevapp.DdevApp{}

			err = app.Init(site.Dir)
			assert.NoError(err)

			for _, service := range ServiceFiles {
				confdir := filepath.Join(app.GetAppRoot(), ".ddev")
				err = fileutil.CopyFile(filepath.Join(ServiceDir, service), filepath.Join(confdir, service))
				assert.NoError(err)
			}

			err = app.Start()
			assert.NoError(err)

			for _, service := range ServiceFiles {
				t.Log("Checking containers for ", service)
				serviceName := strings.TrimPrefix(service, "docker-compose.")
				serviceName = strings.TrimSuffix(serviceName, ".yaml")

				labels := map[string]string{
					"com.ddev.site-name":         app.GetName(),
					"com.docker.compose.service": serviceName,
				}

				container, findErr := dockerutil.FindContainerByLabels(labels)
				assert.NoError(err)
				if findErr != nil {
					t.Fatalf("Could not find running container for service %s. Skipping remainder of test: %v", serviceName, findErr)
				}
				name := dockerutil.ContainerName(container)
				check, runcheckErr := testcommon.ContainerCheck(name, "running")
				assert.NoError(runcheckErr)
				assert.True(check, serviceName, "container is running")

				// check container env for HTTP_EXPOSE ports to check
				expose := dockerutil.GetContainerEnv("HTTP_EXPOSE", container)
				if expose != "" {
					if strings.Contains(expose, ":") {
						ports := strings.Split(expose, ":")
						expose = ports[1]
					}

					containerPorts := container.Ports
					for _, port := range containerPorts {
						if string(port.PrivatePort) == expose && port.PublicPort != 0 {
							log.Debugln("Checking for 200 status for port ", port.PrivatePort)
							o := util.NewHTTPOptions("http://127.0.0.1:" + string(port.PublicPort))
							o.ExpectedStatus = 200
							o.Timeout = 30
							runcheckErr = util.EnsureHTTPStatus(o)
							assert.NoError(runcheckErr)
						}
					}
				}

			}

			err = app.Down(true)
			assert.NoError(err)
			site.Cleanup()
		}
	}
}
