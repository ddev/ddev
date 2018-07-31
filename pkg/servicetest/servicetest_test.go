package servicetest_test

import (
	"os"
	"strings"
	"testing"

	"path/filepath"

	"net"

	"fmt"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	"github.com/fsouza/go-dockerclient"
	log "github.com/sirupsen/logrus"
	asrt "github.com/stretchr/testify/assert"
)

var (
	TestSites = []testcommon.TestSite{
		{
			Name:                          "TestServicesDrupal8",
			SourceURL:                     "https://ftp.drupal.org/files/projects/drupal-8.5.3.tar.gz",
			ArchiveInternalExtractionPath: "drupal-8.5.3/",
			Docroot: "",
			Type:    "drupal8",
		},
	}
	ServiceFiles   []string
	ServiceDir     string
	serviceConfigs = map[string]serviceTestConfig{
		"solr": {
			proto: "http",
			port:  8983,
			path:  "/solr/",
		},
		"memcached": {
			proto: "tcp",
			port:  11211,
		},
	}
)

type serviceTestConfig struct {
	proto string
	port  int64
	path  string
}

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

				// Use the service name to find the expected way to communicate with the service.
				conf := serviceConfigs[serviceName]

				switch conf.proto {
				case "http":
					checkHTTP(t, container, conf.port, conf.path)
					continue

				case "tcp":
					checkTCP(t, container, conf.port, conf.path)
					continue

				default:
					t.Fail()
				}

			}

			err = app.Down(true)
			assert.NoError(err)
			site.Cleanup()
		}
	}
}

// checkHTTP will attempt to communicate with a service over HTTP
func checkHTTP(t *testing.T, container docker.APIContainers, targetPort int64, path string) {
	t.Log("Checking HTTP communication")
	assert := asrt.New(t)

	for _, port := range container.Ports {
		if port.PrivatePort == targetPort && port.PublicPort != 0 {
			address := fmt.Sprintf("http://127.0.0.1:%d%s", port.PublicPort, path)
			t.Logf("Checking %s for 200 status", address)

			o := util.NewHTTPOptions(address)
			o.ExpectedStatus = 200
			o.Timeout = 30
			runcheckErr := util.EnsureHTTPStatus(o)
			assert.NoError(runcheckErr)

			return
		}
	}

	t.Fatalf("Unable to find target port: %d", targetPort)
}

// checkTCP will attempt to communicate with a service over TCP
func checkTCP(t *testing.T, container docker.APIContainers, targetPort int64, path string) {
	t.Log("Checking TCP communication")
	assert := asrt.New(t)

	for _, port := range container.Ports {
		if port.PrivatePort == targetPort {
			address := fmt.Sprintf("127.0.0.1:%d%s", port.PublicPort, path)
			t.Logf("Checking tcp://%s", address)

			_, err := net.Dial("tcp", address)
			assert.NoError(err, "Unable to dial tcp://%s: %s", address, err)

			return
		}
	}

	t.Fatalf("Unable to find target port: %d", targetPort)
}
