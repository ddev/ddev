package ddevapp_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebserverDenyRules checks that paths the webserver configs deny are
// refused for .php requests too, and that their PHP never runs. nginx uses the
// first regex location that matches, so a deny rule placed after the PHP
// location used to let .php files through. TestNginxDenyRulesPrecedePHPLocation
// checks the order in every template; this test checks the result.
func TestWebserverDenyRules(t *testing.T) {
	testcommon.SkipUnlessDefaultEnvironment(t)
	assert := asrt.New(t)
	packageDir, _ := os.Getwd()

	testDir := testcommon.CreateTmpDir(t.Name())
	appDir := filepath.Join(testDir, t.Name())
	err := os.MkdirAll(appDir, 0755)
	require.NoError(t, err)
	err = os.Chdir(appDir)
	require.NoError(t, err)

	app, err := ddevapp.NewApp(appDir, true)
	require.NoError(t, err)
	app.DisableSettingsManagement = true

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(packageDir)
		assert.NoError(err)
		_ = os.RemoveAll(testDir)
	})

	// The output differs from the source, so a served source file is not mistaken for a run.
	const phpSource = `<?php echo "deny-rule-" . "bypassed";`
	const phpRan = "deny-rule-bypassed"

	testCases := []struct {
		appType       string
		webserverType string
		docroot       string
		denyStatus    int
		denied        []string
		allowed       []string
	}{
		{nodeps.AppTypeTYPO3, nodeps.WebserverNginxFPM, "", http.StatusForbidden,
			[]string{"vendor/x.php", "vendor/x.txt", "typo3_src/x.php", "typo3temp/var/x.php", "fileadmin/_temp_/x.php", "fileadmin/templates/x.typoscript", "typo3conf/ext/x/Tests/x.php", ".hidden/x.php"},
			[]string{"typo3temp/assets/x.txt", "vendors/x.txt"}},
		{nodeps.AppTypeCodeIgniter, nodeps.WebserverNginxFPM, "", http.StatusNotFound,
			[]string{"app/x.php", "vendor/x.php", "writable/x.php"}, nil},
		{nodeps.AppTypeMagento2, nodeps.WebserverNginxFPM, "pub", http.StatusForbidden,
			[]string{"media/x.php", "static/x.php", ".hidden/x.php", "media/customer/x.txt"},
			[]string{"index.php", "health_check.php", "x.php"}},
		{nodeps.AppTypeMagento, nodeps.WebserverNginxFPM, "", http.StatusForbidden,
			[]string{"app/etc/local.xml", "var/x.txt"}, nil},
		{nodeps.AppTypeDrupal11, nodeps.WebserverNginxFPM, "web", http.StatusForbidden,
			[]string{"sites/default/files/sync/system.site.yml", "modules/custom/x/x.module", "themes/custom/x/x.html.twig", "x.tpl.php"},
			[]string{"core/misc/x.js"}},
		{nodeps.AppTypePHP, nodeps.WebserverNginxFPM, "", http.StatusForbidden,
			[]string{".env", ".hidden/x.php"}, []string{".well-known/x.txt"}},
		{nodeps.AppTypePHP, nodeps.WebserverApacheFPM, "", http.StatusForbidden,
			[]string{".env", ".git/config", ".hidden/x.php"}, []string{".well-known/x.txt"}},
	}

	for _, tc := range testCases {
		docroot := filepath.Join(appDir, tc.docroot)
		for _, p := range append(append([]string{}, tc.denied...), tc.allowed...) {
			// A path-info request such as x.php/index.php needs only the script before the slash.
			if i := strings.Index(p, ".php/"); i >= 0 {
				p = p[:i+len(".php")]
			}
			f := filepath.Join(docroot, filepath.FromSlash(p))
			err = os.MkdirAll(filepath.Dir(f), 0755)
			require.NoError(t, err)
			content := []byte("plain-file\n")
			if strings.HasSuffix(p, ".php") {
				content = []byte(phpSource)
			}
			err = os.WriteFile(f, content, 0644)
			require.NoError(t, err)
		}

		app.Type = tc.appType
		app.WebserverType = tc.webserverType
		app.Docroot = tc.docroot
		err = app.WriteConfig()
		require.NoError(t, err)
		testcommon.ClearDockerEnv()
		startErr := app.Start()
		if startErr != nil {
			appLogs, health, getLogsErr := ddevapp.GetErrLogsFromApp(app, startErr)
			assert.NoError(getLogsErr)
			t.Fatalf("app.Start() failure for %s/%s: err=%v, health:\n%s\n\nlogs:\n=====\n%s\n=====\n", tc.appType, tc.webserverType, startErr, health, appLogs)
		}

		for _, p := range tc.denied {
			out, _, err := testcommon.GetLocalHTTPResponse(t, app.GetWebContainerDirectHTTPURL()+"/"+p, testcommon.WithExpectStatus(tc.denyStatus))
			require.NoError(t, err, "%s/%s: /%s must be refused with %d", tc.appType, tc.webserverType, p, tc.denyStatus)
			require.NotContains(t, out, phpRan, "%s/%s: /%s must not run", tc.appType, tc.webserverType, p)
		}
		for _, p := range tc.allowed {
			want := "plain-file"
			if strings.HasSuffix(p, ".php") {
				want = phpRan
			}
			out, _, err := testcommon.GetLocalHTTPResponse(t, app.GetWebContainerDirectHTTPURL()+"/"+p)
			require.NoError(t, err, "%s/%s: /%s must stay reachable", tc.appType, tc.webserverType, p)
			require.Contains(t, out, want, "%s/%s: unexpected body for /%s", tc.appType, tc.webserverType, p)
		}

		err = app.Stop(true, false)
		require.NoError(t, err)
	}
}
