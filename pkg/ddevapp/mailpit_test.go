package ddevapp_test

import (
	"fmt"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
	assert2 "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestMailpit does a basic test of mailpit.
func TestMailpit(t *testing.T) {
	assert := assert2.New(t)

	testcommon.ClearDockerEnv()

	origDir, _ := os.Getwd()
	testDir := testcommon.CreateTmpDir(t.Name())

	app, err := ddevapp.NewApp(testDir, false)
	require.NoError(t, err)
	t.Cleanup(func() {
		globalconfig.DdevGlobalConfig.RouterMailpitHTTPPort = ""
		globalconfig.DdevGlobalConfig.RouterMailpitHTTPSPort = ""
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		_ = os.RemoveAll(testDir)
	})

	_ = os.RemoveAll(testDir)
	err = fileutil.CopyDir(filepath.Join(origDir, "testdata", t.Name()), testDir)
	require.NoError(t, err)

	err = app.WriteConfig()
	require.NoError(t, err)

	err = globalconfig.ReadGlobalConfig()
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	stdout, stderr, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "composer install",
	})
	require.NoError(t, err)
	assert.Contains(stderr, "No composer.lock file present. Updating dependencies", "stdout='%s' stderr='%s'", stdout, stderr)

	err = app.MutagenSyncFlush()
	require.NoError(t, err)

	expectation := "Testing DDEV Mailpit on default ports"
	stdout, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     `php send_email.php "` + expectation + `"`,
	})
	require.NoError(t, err)
	assert.Contains(stdout, "Message sent!")

	// See if we got the mail.
	desc, err := app.Describe(true)
	require.NoError(t, err)
	require.NotNil(t, desc["mailpit_url"])
	require.NotNil(t, desc["mailpit_https_url"])

	resp, err := testcommon.EnsureLocalHTTPContent(t, desc["mailpit_url"].(string)+"/api/v1/messages", expectation)
	require.NoError(t, err, "Error getting mailpit_url: %v resp=%v", err, resp)
	// Colima tests on github don't respect https
	if !dockerutil.IsColima() && !dockerutil.IsLima() {
		resp, err = testcommon.EnsureLocalHTTPContent(t, desc["mailpit_https_url"].(string)+"/api/v1/messages", expectation)
		require.NoError(t, err, "Error getting mailpit_url: %v resp=%v", err, resp)
	}
	// Change the global ports to make sure that works
	globalconfig.DdevGlobalConfig.RouterMailpitHTTPPort = "28023"
	globalconfig.DdevGlobalConfig.RouterMailpitHTTPSPort = "28024"
	require.NoError(t, err)

	err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	require.NoError(t, err)

	err = app.Restart()
	require.NoError(t, err)

	expectation = fmt.Sprintf("Testing DDEV Mailpit on global ports %v and %v", globalconfig.DdevGlobalConfig.RouterMailpitHTTPPort, globalconfig.DdevGlobalConfig.RouterMailpitHTTPSPort)
	stdout, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     `php send_email.php "` + expectation + `"`,
	})
	require.NoError(t, err)
	assert.Contains(stdout, "Message sent!")

	desc, err = app.Describe(true)
	require.NoError(t, err)
	require.NotNil(t, desc["mailpit_url"])
	require.NotNil(t, desc["mailpit_https_url"])

	// The API may not be ready the first time we hit it, especially on Rancher Desktop
	// So try a few times
	for i := 0; i < 5; i++ {
		_, _, err = testcommon.GetLocalHTTPResponse(t, desc["mailpit_url"].(string)+"/api/v1/messages")
		if err != nil {
			t.Logf("Error hitting mailpit_url (try %d): %v resp=%v", i, err, resp)
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
	resp, err = testcommon.EnsureLocalHTTPContent(t, desc["mailpit_url"].(string)+"/api/v1/messages", expectation)
	require.NoError(t, err, "Error getting mailpit_url: %v resp=%v", err, resp)
	// Colima tests on GitHub don't respect https
	if !dockerutil.IsColima() && !dockerutil.IsLima() {
		resp, err = testcommon.EnsureLocalHTTPContent(t, desc["mailpit_https_url"].(string)+"/api/v1/messages", expectation)
		require.NoError(t, err, "Error getting mailpit_url: %v resp=%v", err, resp)
	}

	// Change the ports on the project to make sure that works
	app.MailpitHTTPPort = "18025"
	app.MailpitHTTPSPort = "18026"
	err = app.Restart()
	require.NoError(t, err)

	expectation = fmt.Sprintf("Testing DDEV Mailpit on project-overridden ports %v and %v", app.MailpitHTTPPort, app.MailpitHTTPSPort)
	stdout, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     `php send_email.php "` + expectation + `"`,
	})
	require.NoError(t, err)
	assert.Contains(stdout, "Message sent!")

	desc, err = app.Describe(true)
	require.NoError(t, err)
	require.NotNil(t, desc["mailpit_url"])
	require.NotNil(t, desc["mailpit_https_url"])

	// The API may not be ready the first time we hit it, especially on Rancher Desktop
	// So try a few times
	for i := 0; i < 5; i++ {
		_, _, err = testcommon.GetLocalHTTPResponse(t, desc["mailpit_url"].(string)+"/api/v1/messages")
		if err != nil {
			t.Logf("Error hitting mailpit_url (try %d): %v resp=%v", i, err, resp)
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
	resp, err = testcommon.EnsureLocalHTTPContent(t, desc["mailpit_url"].(string)+"/api/v1/messages", expectation)
	require.NoError(t, err, "Error getting mailpit_url: %v resp=%v", err, resp)
	// Colima tests on github don't respect https
	if !dockerutil.IsColima() && !dockerutil.IsLima() {
		resp, err = testcommon.EnsureLocalHTTPContent(t, desc["mailpit_https_url"].(string)+"/api/v1/messages", expectation)
		require.NoError(t, err, "Error getting mailpit_url: %v resp=%v", err, resp)
	}
}
