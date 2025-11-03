package ddevapp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetXDdevExtension tests the GetXDdevExtension method of the ddevapp.App struct.
func TestGetXDdevExtension(t *testing.T) {
	assert := asrt.New(t)

	app := &ddevapp.DdevApp{}

	// Create a minimal compose YAML with x-ddev extensions
	composeYAML := `
name: test-project
services:
  web:
    image: ddev/ddev-utilities
    x-ddev:
      describe-url-port: |
        web
        description
      describe-info: "web info"
  db:
    image: ddev/ddev-utilities
    x-ddev:
      shell: "  /bin/zsh  "
      describe-info: "  db info with spaces  "
  custom:
    image: ddev/ddev-utilities
    x-ddev:
      shell: fish
  noshell:
    image: ddev/ddev-utilities
    x-ddev:
      describe-info: "has info but no shell"
  noextension:
    image: ddev/ddev-utilities
`

	project, err := dockerutil.CreateComposeProject(composeYAML)
	require.NoError(t, err)
	app.ComposeYaml = project

	// Test web service
	t.Run("web service with shell", func(t *testing.T) {
		xDdev := app.GetXDdevExtension("web")
		assert.Equal("web info", xDdev.DescribeInfo)
		assert.Equal("web\ndescription", xDdev.DescribeURLPort)
		assert.Equal("bash", xDdev.Shell)
	})

	// Test db service (overrides custom shell)
	t.Run("db service defaults to bash", func(t *testing.T) {
		xDdev := app.GetXDdevExtension("db")
		assert.Equal("db info with spaces\nShell: /bin/zsh", xDdev.DescribeInfo)
		assert.Equal("", xDdev.DescribeURLPort)
		assert.Equal("/bin/zsh", xDdev.Shell)
	})

	// Test custom service - should use custom shell
	t.Run("custom service with custom shell", func(t *testing.T) {
		xDdev := app.GetXDdevExtension("custom")
		assert.Equal("Shell: fish", xDdev.DescribeInfo)
		assert.Equal("", xDdev.DescribeURLPort)
		assert.Equal("fish", xDdev.Shell)
	})

	// Test service with no shell - should default to sh
	t.Run("service without shell defaults to sh", func(t *testing.T) {
		xDdev := app.GetXDdevExtension("noshell")
		assert.Equal("has info but no shell", xDdev.DescribeInfo)
		assert.Equal("", xDdev.DescribeURLPort)
		assert.Equal("sh", xDdev.Shell)
	})

	// Test service with no x-ddev extension - should default to sh
	t.Run("service without x-ddev extension", func(t *testing.T) {
		xDdev := app.GetXDdevExtension("noextension")
		assert.Equal("", xDdev.DescribeInfo)
		assert.Equal("", xDdev.DescribeURLPort)
		assert.Equal("sh", xDdev.Shell)
	})

	// Test non-existent service
	t.Run("non-existent service", func(t *testing.T) {
		xDdev := app.GetXDdevExtension("nonexistent")
		assert.Equal("", xDdev.DescribeInfo)
		assert.Equal("", xDdev.DescribeURLPort)
		assert.Equal("sh", xDdev.Shell)
	})

	// Test with nil ComposeYaml
	t.Run("nil ComposeYaml", func(t *testing.T) {
		app.ComposeYaml = nil
		// web service should still default to bash even with nil ComposeYaml
		xDdev := app.GetXDdevExtension("web")
		assert.Equal("", xDdev.DescribeInfo)
		assert.Equal("", xDdev.DescribeURLPort)
		assert.Equal("bash", xDdev.Shell)
		// non-web/db service should default to sh
		xDdev = app.GetXDdevExtension("custom")
		assert.Equal("", xDdev.DescribeInfo)
		assert.Equal("", xDdev.DescribeURLPort)
		assert.Equal("sh", xDdev.Shell)
	})
}

// TestMultipleComposeFiles checks to see if a set of docker-compose files gets
// properly loaded in the right order, with .ddev/.ddev-docker-compose*yaml first and
// with docker-compose.override.yaml last.
func TestMultipleComposeFiles(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	pwd, _ := os.Getwd()

	testDir := testcommon.CreateTmpDir(t.Name())
	//_ = os.Chdir(testDir)
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	err := fileutil.CopyDir(filepath.Join(pwd, "testdata", t.Name(), ".ddev"), filepath.Join(testDir, ".ddev"))
	assert.NoError(err)

	// Make sure that valid yaml files get properly loaded in the proper order
	app, err := ddevapp.NewApp(testDir, true)
	assert.NoError(err)
	_ = app.DockerEnv()

	//nolint: errcheck
	defer app.Stop(true, false)

	err = app.WriteConfig()
	assert.NoError(err)
	_, err = app.ReadConfig(true)
	require.NoError(t, err)
	err = app.WriteDockerComposeYAML()
	require.NoError(t, err)

	app, err = ddevapp.NewApp(testDir, true)
	assert.NoError(err)
	//nolint: errcheck
	defer app.Stop(true, false)

	desc, err := app.Describe(false)
	assert.NoError(err)
	_ = desc

	files, err := app.ComposeFiles()
	assert.NoError(err)
	require.NotEmpty(t, files)
	assert.Equal(4, len(files))
	require.Equal(t, app.GetConfigPath(".ddev-docker-compose-base.yaml"), files[0])
	require.Equal(t, app.GetConfigPath("docker-compose.override.yaml"), files[len(files)-1])

	require.NotEmpty(t, app.ComposeYaml)
	require.NotEmpty(t, app.ComposeYaml.Services)
	require.NotEmpty(t, app.ComposeYaml.Networks)
	require.NotEmpty(t, app.ComposeYaml.Volumes)

	webService, ok := app.ComposeYaml.Services["web"]
	require.True(t, ok, "web service not found in app.ComposeYaml.Services")
	// Verify that the env var DUMMY_BASE got set by docker-compose.override.yaml
	// The docker-compose.override should have won with the value of DUMMY_BASE
	require.NotNil(t, webService.Environment["DUMMY_BASE"])
	assert.Equal("override", *webService.Environment["DUMMY_BASE"])
	// But each of the DUMMY_COMPOSE_ONE/TWO/OVERRIDE which are unique
	// should come through fine.
	require.NotNil(t, webService.Environment["DUMMY_COMPOSE_ONE"])
	assert.Equal("1", *webService.Environment["DUMMY_COMPOSE_ONE"])
	require.NotNil(t, webService.Environment["DUMMY_COMPOSE_TWO"])
	assert.Equal("2", *webService.Environment["DUMMY_COMPOSE_TWO"])
	require.NotNil(t, webService.Environment["DUMMY_COMPOSE_OVERRIDE"])
	assert.Equal("override", *webService.Environment["DUMMY_COMPOSE_OVERRIDE"])

	_, err = app.ComposeFiles()
	assert.NoError(err)
}

// TestFixupComposeYaml verifies that the fixupComposeYaml function properly applies
// required DDEV configurations to all services in the compose project.
func TestFixupComposeYaml(t *testing.T) {
	assert := asrt.New(t)
	pwd, _ := os.Getwd()

	testDir := testcommon.CreateTmpDir(t.Name())

	t.Cleanup(func() {
		testcommon.CleanupDir(testDir)
	})

	defer testcommon.Chdir(testDir)()

	err := fileutil.CopyDir(filepath.Join(pwd, "testdata", t.Name(), ".ddev"), filepath.Join(testDir, ".ddev"))
	require.NoError(t, err)

	app, err := ddevapp.NewApp(testDir, true)
	require.NoError(t, err)
	_ = app.DockerEnv()

	t.Cleanup(func() {
		err := app.Stop(true, false)
		assert.NoError(err)
	})

	err = app.WriteConfig()
	require.NoError(t, err)

	_, err = app.ReadConfig(true)
	require.NoError(t, err)

	err = app.WriteDockerComposeYAML()
	require.NoError(t, err)

	app, err = ddevapp.NewApp(testDir, true)
	require.NoError(t, err)

	require.NotEmpty(t, app.ComposeYaml)
	require.NotEmpty(t, app.ComposeYaml.Services)

	expectedServices := []string{"web", "db", "dummy1"}
	for _, serviceName := range expectedServices {
		_, ok := app.ComposeYaml.Services[serviceName]
		require.True(t, ok, "%s service not found in app.ComposeYaml.Services", serviceName)
	}

	webService := app.ComposeYaml.Services["web"]
	require.Nil(t, webService.Networks["ddev_default"])
	require.NotNil(t, webService.Networks["default"])
	assert.Equal(1, webService.Networks["default"].Priority)
	require.NotNil(t, webService.Networks["dummy"])
	assert.Equal(2, webService.Networks["dummy"].Priority)

	hostDockerInternal := dockerutil.GetHostDockerInternal()

	for serviceName, service := range app.ComposeYaml.Services {
		t.Run("service_"+serviceName, func(t *testing.T) {
			require.Contains(t, service.Networks, "ddev_default", "service %s missing ddev_default network", serviceName)
			require.Contains(t, service.Networks, "default", "service %s missing default network", serviceName)

			require.NotNil(t, service.Environment["HOST_DOCKER_INTERNAL_IP"], "service %s missing HOST_DOCKER_INTERNAL_IP", serviceName)
			require.Equal(t, hostDockerInternal.IPAddress, *service.Environment["HOST_DOCKER_INTERNAL_IP"], "service %s HOST_DOCKER_INTERNAL_IP value incorrect", serviceName)

			if hostDockerInternal.ExtraHost != "" {
				require.NotNil(t, service.ExtraHosts, "service %s missing ExtraHosts", serviceName)
				require.Contains(t, service.ExtraHosts, "host.docker.internal", "service %s missing host.docker.internal in ExtraHosts", serviceName)
				require.Contains(t, service.ExtraHosts["host.docker.internal"], hostDockerInternal.ExtraHost, "service %s host.docker.internal should contain %s", serviceName, hostDockerInternal.ExtraHost)
			}

			for portIndex, port := range service.Ports {
				require.Equal(t, "127.0.0.1", port.HostIP, "service %s port %d should have HostIP set to 127.0.0.1", serviceName, portIndex)
			}
		})
	}

	hasPort12345 := false
	for _, port := range webService.Ports {
		if port.Target == 12345 {
			hasPort12345 = true
			break
		}
	}
	require.True(t, hasPort12345, "no port with Target 12345 found in web service")

	// Test network configurations
	networkTests := []struct {
		key      string
		name     string
		external bool
		label    string
	}{
		{"ddev_default", "ddev_default", true, ""},
		{"default", app.GetDefaultNetworkName(), false, "com.ddev.platform"},
		{"dummy", "dummy_name", false, "com.ddev.platform"},
	}

	require.Len(t, app.ComposeYaml.Networks, 3)

	for _, networkTest := range networkTests {
		t.Run("network_"+networkTest.key, func(t *testing.T) {
			network, exists := app.ComposeYaml.Networks[networkTest.key]
			require.True(t, exists, "network %s not found", networkTest.key)
			assert.Equal(networkTest.name, network.Name, "unexpected name for %s", networkTest.key)
			assert.Equal(networkTest.external, bool(network.External), "unexpected external for %s", networkTest.key)

			if networkTest.label != "" {
				require.NotNil(t, network.Labels, "labels missing for %s", networkTest.key)
				_, hasLabel := network.Labels[networkTest.label]
				assert.True(hasLabel, "%s label missing for %s", networkTest.label, networkTest.key)
			}
		})
	}
}
