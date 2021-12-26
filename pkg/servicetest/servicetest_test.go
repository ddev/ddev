package servicetest_test

import (
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/stretchr/testify/require"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"path/filepath"

	"fmt"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
)

// TestServices tests each service compose file in the services folder.
// It tests that a site can fully start w/ the compose file present, and
// runs each service's check function to ensure it's accessible from
// the web container.
func TestServices(t *testing.T) {
	if runtime.GOOS == "windows" || dockerutil.IsColima() {
		t.Skip("skipping because unreliable, windows and colima seem to have port conflicts")
	}

	assert := asrt.New(t)
	_ = os.Setenv("DDEV_NONINTERACTIVE", "true")

	// We expect to find web + db + dba + what we add on here
	expectedServiceCount := 6

	err := globalconfig.ReadGlobalConfig()
	assert.NoError(err)

	origDir, _ := os.Getwd()
	testDir := testcommon.CreateTmpDir(t.Name())

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(testDir)
		assert.NoError(err)
	})

	workingServices := map[string]bool{
		"beanstalkd": true,
		"memcached":  true,
		"solr":       true,
	}
	err = os.Chdir(testDir)
	assert.NoError(err)

	app, err := ddevapp.NewApp(testDir, false)
	assert.NoError(err)
	err = fileutil.CopyDir(filepath.Join(origDir, "testdata", t.Name()), app.AppConfDir())
	assert.NoError(err)

	// the beanstalkd image is not pushed for arm64
	if runtime.GOARCH == "arm64" {
		workingServices["beanstalkd"] = false
	}

	// If bind-mounts are required, as currently with solr image, skip solr
	if globalconfig.DdevGlobalConfig.NoBindMounts || dockerutil.IsColima() {
		workingServices["solr"] = false
	}

	for s, enabled := range workingServices {
		if !enabled {
			err = os.RemoveAll(filepath.Join(app.GetConfigPath("docker-compose." + s + ".yaml")))
			expectedServiceCount = expectedServiceCount - 1
			assert.NoError(err)
		}
	}

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.RemoveAll(testDir)
		assert.NoError(err)
	})

	app.Name = t.Name()
	err = app.WriteConfig()
	assert.NoError(err)

	testcommon.ClearDockerEnv()

	err = app.Start()
	require.NoError(t, err)

	// Unfortunately to get service description to work, we have to create a new app
	// now that files are in place
	app, err = ddevapp.NewApp(app.AppRoot, false)

	if workingServices["solr"] {
		checkSolrService(t, app)
	}
	checkMemcachedService(t, app)

	desc, err := app.Describe(false)
	require.NoError(t, err)

	// Make sure desc has right number of services.
	require.Len(t, desc["services"].(map[string]map[string]string), expectedServiceCount)

	if workingServices["solr"] {
		// A volume should have been created for solr (only)
		assert.True(dockerutil.VolumeExists(strings.ToLower("ddev-" + app.Name + "_" + "solr")))
	}
	err = app.Stop(true, false)
	assert.NoError(err)

	if workingServices["solr"] {
		// Solr volume should have been deleted
		assert.False(dockerutil.VolumeExists(strings.ToLower("ddev-" + app.Name + "_" + "solr")))
	}
}

// checkSolrService ensures that the solr service's container is
// running and that the service is accessible from the web container
func checkSolrService(t *testing.T, app *ddevapp.DdevApp) {
	service := "solr"
	httpPort := 8983
	path := fmt.Sprintf("http://%s:%d/solr/", service, httpPort)

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

	// solr service seems to take a couple of seconds to come up after container running.
	time.Sleep(5 * time.Second)

	// Ensure service is accessible from web container
	checkCommand := fmt.Sprintf("curl -slL -w '%%{http_code}' %s -o /dev/null", path)
	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     checkCommand,
	})
	assert.NoError(err, "Unable to make in-webcontainer request to http://%s:%d/solr/", service, httpPort)
	assert.Equal("200", out)

	// Ensure solr service is available via HTTP at exposed port location
	resp, err := testcommon.EnsureLocalHTTPContent(t, fmt.Sprintf("http://%s.ddev.site:%d/solr/", app.GetName(), httpPort), "", 5)
	assert.NoError(err, "resp=%v", resp)
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
