package testcommon

import (
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	copy2 "github.com/otiai10/copy"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// URIWithExpect pairs a URI like "/readme.html" with some substring content "should be found in URI"
type URIWithExpect struct {
	URI    string
	Expect string
}

// TestSite describes a site for testing, with name, URL of tarball, and optional dir.
type TestSite struct {
	// Name is the generic name of the site, and is used as the default dir.
	Name string
	// Provide ability to disable
	Disable bool
	// SourceURL is the URL of the source code tarball to be used for building the site.
	SourceURL string
	// ArchiveExtractionPath is the relative path within the tarball which should be extracted, ending with /
	ArchiveInternalExtractionPath string
	// FullSiteTarballURL is the URL of the tarball of a full site archive used for testing import.
	FullSiteTarballURL string
	// FilesTarballURL is the URL of the tarball of file uploads used for testing file import.
	FilesTarballURL string
	// FilesZipballURL is the URL of the zipball of file uploads used for testing file import.
	FilesZipballURL string
	// DBTarURL is the URL of the database dump tarball used for testing database import.
	DBTarURL string
	// DBZipURL is the URL of an optional zip-style db dump.
	DBZipURL string
	// Dir is the rooted full path of the test site
	Dir string
	// HTTPProbeURI is the URI that can be probed to look for a working web container
	HTTPProbeURI string
	// WebEnvironment is strings that will be used in web_environment
	WebEnvironment []string
	// WebserverType if needed (apache-fpm, generic)
	WebserverType string
	// PretestCmd will be executed in web-entrypoint.d script before
	// daemons are started inside the web container
	PretestCmd string
	// Docroot is the subdirectory within the site that is the root/index.php
	Docroot string
	// Type is the type of application. This can be specified when a config file is not present
	// for a test site.
	Type string
	// Safe200URIWithExpectation provides a static URI with contents that it can be expected to contain.
	Safe200URIWithExpectation URIWithExpect
	// DynamicURI provides a dynamic (after db load) URI with contents we can expect.
	DynamicURI URIWithExpect
	// UploadDirs overrides the dirs used for upload_dirs
	UploadDirs []string
	// FilesImageURI is URI to a file loaded by import-files that is a jpg.
	FilesImageURI string
	// FullSiteArchiveExtPath is the path that should be extracted from inside an archive when
	// importing the files from a full site archive
	FullSiteArchiveExtPath string
}

// HTTPRequestOpts contains options for HTTP requests
type HTTPRequestOpts struct {
	// TimeoutSeconds is the number of seconds to wait for a response before timing out
	TimeoutSeconds int
	// MaxRetries is the number of times to retry the request
	MaxRetries int
}

// Prepare downloads and extracts a site codebase to a temporary directory.
func (site *TestSite) Prepare() error {
	testDir := CreateTmpDir(site.Name)
	site.Dir = testDir

	err := os.Setenv("DDEV_NONINTERACTIVE", "true")
	util.CheckErr(err)

	cachedSrcDir, _, err := GetCachedArchive(site.Name, site.Name+"_siteArchive", site.ArchiveInternalExtractionPath, site.SourceURL)

	if err != nil {
		site.Cleanup()
		return fmt.Errorf("failed to GetCachedArchive, err=%v", err)
	}
	// We must copy into a directory that does not yet exist :(
	err = os.Remove(site.Dir)
	util.CheckErr(err)

	output.UserOut.Printf("Copying directory %s to %s\n", cachedSrcDir, site.Dir)
	if !nodeps.IsWindows() {
		// Simple cp -r is far, far faster than our fileutil.CopyDir
		cmd := exec.Command("bash", "-c", fmt.Sprintf(`cp -rp %s %s`, cachedSrcDir, site.Dir))
		err = cmd.Run()
	} else {
		err = fileutil.CopyDir(cachedSrcDir, site.Dir)
	}
	if err != nil {
		site.Cleanup()
		return fmt.Errorf("failed to CopyDir from %s to %s, err=%v", cachedSrcDir, site.Dir, err)
	}
	output.UserOut.Println("Copying complete")

	// Remove existing in project registry
	_ = globalconfig.RemoveProjectInfo(site.Name)

	// Create an app. Err is ignored as we may not have
	// a config file to read in from a test site.
	app, err := ddevapp.NewApp(site.Dir, true)
	if err != nil {
		return err
	}
	// Set app name to the name we define for test sites. We'll
	// ignore app name defined in config file if present.
	app.Name = site.Name
	app.Docroot = site.Docroot
	app.UploadDirs = site.UploadDirs
	app.Type = site.Type
	app.WebserverType = site.WebserverType
	detectedType := app.DetectAppType()
	if app.Type != detectedType && app.Type != nodeps.AppTypeGeneric {
		return fmt.Errorf("detected apptype (%s) does not match provided site.Type (%s)", detectedType, site.Type)
	}

	app.WebEnvironment = site.WebEnvironment
	if site.PretestCmd != "" {
		err = os.MkdirAll(app.GetConfigPath("web-entrypoint.d"), 0755)
		if err != nil {
			return err
		}
		err = os.WriteFile(app.GetConfigPath("web-entrypoint.d/pretest.sh"), []byte(site.PretestCmd), 0755)
		if err != nil {
			return fmt.Errorf("failed to write pretest.sh, err=%v", err)
		}
	}
	err = app.ConfigFileOverrideAction(false)
	util.CheckErr(err)

	err = os.MkdirAll(filepath.Join(app.AppRoot, app.Docroot, app.GetUploadDir()), 0777)
	if err != nil {
		return fmt.Errorf("failed to create upload dir for test site: %v", err)
	}

	// Force creation of new global config if none exists.
	_ = globalconfig.ReadGlobalConfig()
	_ = globalconfig.ReadProjectList()

	err = app.WriteConfig()
	if err != nil {
		return fmt.Errorf("failed to write site config for site %s, dir %s, err: %v", app.Name, app.GetAppRoot(), err)
	}

	return nil
}

// Chdir will change to the directory for the site specified by TestSite.
func (site *TestSite) Chdir() func() {
	return Chdir(site.Dir)
}

// Cleanup removes the archive and codebase extraction for a site after a test run has completed.
func (site *TestSite) Cleanup() {
	// CleanupDir checks its own errors.
	CleanupDir(site.Dir)

	_ = globalconfig.RemoveProjectInfo(site.Name)
	siteData := filepath.Join(globalconfig.GetGlobalDdevDir(), site.Name)
	if fileutil.FileExists(siteData) {
		CleanupDir(siteData)
	}
}

// CleanupDir removes a directory specified by string.
func CleanupDir(dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		output.UserErr.Warn(fmt.Sprintf("Failed to remove directory %s, err: %v", dir, err))
	}
}

// OsTempDir gets os.TempDir() (usually provided by $TMPDIR) but expands any symlinks found within it.
// This wrapper function can prevent problems with docker-for-mac trying to use /var/..., which is not typically
// shared/mounted. It will be expanded via the /var symlink to /private/var/...
func OsTempDir() (string, error) {
	dirName := os.TempDir()
	tmpDir, err := filepath.EvalSymlinks(dirName)
	if err != nil {
		return "", err
	}
	tmpDir = filepath.Clean(tmpDir)
	return tmpDir, nil
}

// CreateTmpDir creates a temporary directory in the homedir
// and returns its path as a string. It's important that it's in
// homedir since Colima doesn't mount things outside that.
func CreateTmpDir(prefix string) string {
	baseTmpDir := filepath.Join(util.GetHomeDir(), "tmp", "ddevtest")
	_ = os.MkdirAll(baseTmpDir, 0755)
	fullPath, err := os.MkdirTemp(baseTmpDir, prefix)
	if err != nil {
		output.UserErr.Fatalf("Failed to create temp directory %s, err=%v", fullPath, err)
	}
	// Make the tmpdir fully writeable/readable
	_ = util.Chmod(fullPath, 0777)
	return fullPath
}

// CopyGlobalDdevDir creates a temporary global config directory for DDEV
// using a temporary directory which is set to $XDG_CONFIG_HOME/ddev
// Don't forget to run ResetGlobalDdevDir(t, tmpXdgConfigHomeDir)
// in the test's cleanup function.
func CopyGlobalDdevDir(t *testing.T) string {
	// Create $XDG_CONFIG_HOME
	tmpXdgConfigHomeDir := CreateTmpDir("Home_" + util.RandString(5))
	// Global DDEV config directory should be named "ddev"
	tmpGlobalDdevDir := filepath.Join(tmpXdgConfigHomeDir, "ddev")
	// Make sure that the tmpDir/ddev doesn't exist.
	_, err := os.Stat(tmpGlobalDdevDir)
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
	// Original ~/.ddev dir location
	originalGlobalDdevDir := globalconfig.GetGlobalDdevDirLocation()
	// Make sure that the global config directory is set to ~/.ddev
	require.Equal(t, originalGlobalDdevDir, globalconfig.GetGlobalDdevDir())
	// Make sure that the original global config directory exists
	require.DirExists(t, originalGlobalDdevDir)
	originalGlobalConfig := globalconfig.DdevGlobalConfig
	// Stop the Mutagen daemon running in the ~/.ddev
	ddevapp.StopMutagenDaemon("")
	t.Logf("stopped mutagen daemon %s in MUTAGEN_DATA_DIRECTORY=%s", globalconfig.GetMutagenPath(), globalconfig.GetMutagenDataDirectory())
	// Set $XDG_CONFIG_HOME for tests
	t.Setenv("XDG_CONFIG_HOME", tmpXdgConfigHomeDir)
	// Make sure that the global config directory is set to $XDG_CONFIG_HOME/ddev
	require.Equal(t, tmpGlobalDdevDir, globalconfig.GetGlobalDdevDir())
	// And it should be created by now
	require.DirExists(t, tmpGlobalDdevDir)
	// Create the global config in $XDG_CONFIG_HOME/ddev
	globalconfig.EnsureGlobalConfig()
	// Copy some settings from ~/.ddev to $XDG_CONFIG_HOME/ddev
	globalconfig.DdevGlobalConfig.PerformanceMode = originalGlobalConfig.PerformanceMode
	globalconfig.DdevGlobalConfig.LastStartedVersion = originalGlobalConfig.LastStartedVersion
	err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	require.NoError(t, err)
	// Make sure we have the .ddev/bin dir we need for docker-compose and Mutagen
	sourceBinDir := filepath.Join(originalGlobalDdevDir, "bin")
	_, err = os.Stat(sourceBinDir)
	if !os.IsNotExist(err) {
		// Copy ~/.ddev/bin to $XDG_CONFIG_HOME/ddev/bin
		err = copy2.Copy(sourceBinDir, filepath.Join(tmpGlobalDdevDir, "bin"))
		require.NoError(t, err)
	}
	// globalconfig.GetMutagenDataDirectory sets MUTAGEN_DATA_DIRECTORY
	_ = globalconfig.GetMutagenDataDirectory()
	// Start mutagen daemon if it's enabled
	if globalconfig.DdevGlobalConfig.IsMutagenEnabled() {
		ddevapp.StartMutagenDaemon()
		t.Logf("started mutagen daemon '%s' with MUTAGEN_DATA_DIRECTORY='%s'", globalconfig.GetMutagenPath(), globalconfig.GetMutagenDataDirectory())
		// Make sure that $MUTAGEN_DATA_DIRECTORY is set to the correct directory
		require.Equal(t, os.Getenv("MUTAGEN_DATA_DIRECTORY"), globalconfig.GetMutagenDataDirectory())
	}

	return tmpXdgConfigHomeDir
}

// ResetGlobalDdevDir removes temporary $XDG_CONFIG_HOME directory
func ResetGlobalDdevDir(t *testing.T, tmpXdgConfigHomeDir string) {
	// Stop the Mutagen daemon running in the $XDG_CONFIG_HOME/ddev
	ddevapp.StopMutagenDaemon("")
	t.Logf("stopped mutagen daemon '%s' with MUTAGEN_DATA_DIRECTORY=%s", globalconfig.GetMutagenPath(), globalconfig.GetMutagenDataDirectory())
	// After the $XDG_CONFIG_HOME directory is removed,
	// globalconfig.GetGlobalDdevDir() should point to ~/.ddev
	t.Setenv("XDG_CONFIG_HOME", "")
	_ = os.RemoveAll(tmpXdgConfigHomeDir)
	// Make sure that the global config directory is set to ~/.ddev
	originalGlobalDdevDir := globalconfig.GetGlobalDdevDirLocation()
	require.Equal(t, originalGlobalDdevDir, globalconfig.GetGlobalDdevDir())
	// Make sure that the original global config directory exists
	require.DirExists(t, originalGlobalDdevDir)
	// refresh the global config from ~/.ddev
	globalconfig.EnsureGlobalConfig()
	// Set $MUTAGEN_DATA_DIRECTORY
	_ = globalconfig.GetMutagenDataDirectory()

	// Start mutagen daemon if it's enabled
	if globalconfig.DdevGlobalConfig.IsMutagenEnabled() {
		ddevapp.StartMutagenDaemon()
		t.Logf("started mutagen daemon '%s' with MUTAGEN_DATA_DIRECTORY=%s", globalconfig.GetMutagenPath(), globalconfig.GetMutagenDataDirectory())
		// Make sure that $MUTAGEN_DATA_DIRECTORY is set to the correct directory
		require.Equal(t, os.Getenv("MUTAGEN_DATA_DIRECTORY"), globalconfig.GetMutagenDataDirectory())
	}
}

// Chdir will change to the directory for the site specified by TestSite.
// It returns an anonymous function which will return to the original working directory when called.
func Chdir(path string) func() {
	curDir, _ := os.Getwd()
	err := os.Chdir(path)
	if err != nil {
		output.UserErr.Errorf("Could not change to directory %s: %v\n", path, err)
	}

	return func() {
		err := os.Chdir(curDir)
		if err != nil {
			output.UserErr.Errorf("Failed to change directory to original dir=%s, err=%v", curDir, err)
		}
	}
}

// ClearDockerEnv unsets env vars set in platform DockerEnv() so that
// they can be set by another test run.
func ClearDockerEnv() {
	envVars := []string{
		"COMPOSE_PROJECT_NAME",
		"DDEV_SITENAME",
		"DDEV_DBIMAGE",
		"DDEV_WEBIMAGE",
		"DDEV_APPROOT",
		"DDEV_HOST_WEBSERVER_PORT",
		"DDEV_HOST_HTTPS_PORT",
		"DDEV_DOCROOT",
		"DDEV_HOSTNAME",
		"DDEV_DB_CONTAINER_COMMAND",
		"DDEV_PHP_VERSION",
		"DDEV_WEBSERVER_TYPE",
		"DDEV_PROJECT_TYPE",
		"DDEV_ROUTER_HTTP_PORT",
		"DDEV_ROUTER_HTTPS_PORT",
		"DDEV_HOST_DB_PORT",
		"DDEV_HOST_WEBSERVER_PORT",
		"DDEV_MAILPIT_PORT",
		"DDEV_MAILPIT_HTTPS_PORT",
		"COLUMNS",
		"LINES",
		"DDEV_XDEBUG_ENABLED",
		"IS_DDEV_PROJECT",
	}
	for _, env := range envVars {
		err := os.Unsetenv(env)
		if err != nil {
			output.UserErr.Warnf("failed to unset %s: %v\n", env, err)
		}
	}
}

// ContainerCheck determines if a given container name exists and matches a given state
func ContainerCheck(checkName string, checkState string) (bool, error) {
	// Ensure we have DDEV network
	dockerutil.EnsureDdevNetwork()

	c, err := dockerutil.FindContainerByName(checkName)
	if err != nil {
		output.UserErr.Fatal(err)
	}
	if c == nil {
		return false, fmt.Errorf("unable to find container %s", checkName)
	}

	if string(c.State) == checkState {
		return true, nil
	}
	return false, fmt.Errorf("container %s returned %s", checkName, c.State)
}

// GetCachedArchive returns a directory populated with the contents of the specified archive, either from cache or
// from downloading and creating cache.
// siteName is the site.Name used for storage
// prefixString is the prefix used to disambiguate downloads and extracts
// internalExtractionPath is the place in the archive to start extracting
// sourceURL is the actual URL to download.
// Returns the extracted path, the tarball path (both possibly cached), and an error value.
func GetCachedArchive(_, _, internalExtractionPath, sourceURL string) (string, string, error) {
	uniqueName := fmt.Sprintf("%.4x_%s", sha256.Sum256([]byte(sourceURL)), path.Base(sourceURL))
	testCache := filepath.Join(globalconfig.GetGlobalDdevDir(), "testcache")
	archiveFullPath := filepath.Join(testCache, "tarballs", uniqueName)
	_ = os.MkdirAll(filepath.Dir(archiveFullPath), 0777)
	extractPath := filepath.Join(testCache, uniqueName)

	// Check to see if we have it cached, if so return it.
	dStat, dErr := os.Stat(extractPath)
	aStat, aErr := os.Stat(archiveFullPath)
	if dErr == nil && dStat.IsDir() && aErr == nil && !aStat.IsDir() {
		return extractPath, archiveFullPath, nil
	}

	// Download if archive does not already exist.
	if aErr != nil {
		output.UserOut.Printf("Downloading %s", sourceURL)

		err := util.DownloadFile(archiveFullPath, sourceURL, false, "")
		if err != nil {
			_ = os.RemoveAll(archiveFullPath)
			return extractPath, archiveFullPath, fmt.Errorf("failed to download url=%s into %s, err=%v", sourceURL, archiveFullPath, err)
		}

		output.UserOut.Printf("Downloaded %s into %s", sourceURL, archiveFullPath)
	}

	err := os.RemoveAll(extractPath)
	if err != nil {
		return extractPath, "", fmt.Errorf("failed to remove %s: %v", extractPath, err)
	}

	if filepath.Ext(archiveFullPath) == ".zip" {
		err = archive.Unzip(archiveFullPath, extractPath, internalExtractionPath)
	} else {
		err = archive.Untar(archiveFullPath, extractPath, internalExtractionPath)
	}

	if err != nil {
		_ = fileutil.PurgeDirectory(extractPath)
		_ = os.RemoveAll(extractPath)
		_ = os.RemoveAll(archiveFullPath)
		return extractPath, archiveFullPath, fmt.Errorf("archive extraction of %s failed err=%v", archiveFullPath, err)
	}

	output.UserOut.Printf("Extracted %s into %s", archiveFullPath, extractPath)

	return extractPath, archiveFullPath, nil
}

// GetLocalHTTPResponse takes a URL and optional parameters,
// hits the local Docker for it, returns result
// Returns error (with the body) if not 200 status code.
// Parameters can be either:
// - HTTPRequestOpts struct with TimeoutSeconds and MaxRetries fields
// - int representing timeout seconds (for backward compatibility)
func GetLocalHTTPResponse(t *testing.T, rawurl string, params ...interface{}) (string, *http.Response, error) {
	options := parseHTTPRequestOpts(60, params...)

	timeoutTime := time.Duration(options.TimeoutSeconds) * time.Second
	assert := asrt.New(t)

	u, err := url.Parse(rawurl)
	if err != nil {
		t.Fatalf("Failed to parse url %s: %v", rawurl, err)
	}
	port := u.Port()

	dockerIP, err := dockerutil.GetDockerIP()
	assert.NoError(err)

	fakeHost := u.Hostname()
	// Add the port if there is one.
	u.Host = dockerIP
	if port != "" {
		u.Host = u.Host + ":" + port
	}
	localAddress := u.String()

	// Use ServerName: fakeHost to verify basic usage of certificate.
	// This technique is from https://stackoverflow.com/a/47169975/215713
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{ServerName: fakeHost},
	}

	// Do not follow redirects, https://stackoverflow.com/a/38150816/215713
	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: transport,
		Timeout:   timeoutTime,
	}

	var lastErr error
	var resp *http.Response
	var bodyString string

	for attempt := 1; attempt <= options.MaxRetries; attempt++ {
		req, err := http.NewRequest("GET", localAddress, nil)
		if err != nil {
			return "", nil, fmt.Errorf("failed to NewRequest GET %s: %v", localAddress, err)
		}
		req.Host = fakeHost

		resp, err = client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < options.MaxRetries {
				time.Sleep(time.Second)
				continue
			}
			return "", resp, err
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("unable to ReadAll resp.body: %v", err)
			if attempt < options.MaxRetries {
				time.Sleep(time.Second)
				continue
			}
			return "", resp, lastErr
		}

		bodyString = string(bodyBytes)
		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("http status code for '%s' was %d, not 200", localAddress, resp.StatusCode)
			if attempt < options.MaxRetries {
				time.Sleep(time.Second)
				continue
			}
			return bodyString, resp, lastErr
		}

		// Success
		return bodyString, resp, nil
	}

	// This should never be reached, but just in case
	return bodyString, resp, lastErr
}

// GetLocalHTTPResponseWithBackoff calls GetLocalHTTPResponse with an external
// retry/backoff strategy. It takes a number of attempts and an initialDelay
// duration. params follow the same conventions as GetLocalHTTPResponse
// (HTTPRequestOpts or int timeout). The inner call has MaxRetries forced to 1
// to avoid nested retries.
func GetLocalHTTPResponseWithBackoff(t *testing.T, rawurl string, attempts int, initialDelay time.Duration, params ...interface{}) (string, *http.Response, error) {
	if attempts < 1 {
		attempts = 1
	}
	// Build options from params but force inner MaxRetries to 1 to avoid nested retries.
	innerOpts := parseHTTPRequestOpts(60, params...)
	innerOpts.MaxRetries = 1

	var lastBody string
	var lastResp *http.Response
	var lastErr error
	delay := initialDelay

	for attempt := 1; attempt <= attempts; attempt++ {
		body, resp, err := GetLocalHTTPResponse(t, rawurl, innerOpts)
		if err == nil {
			return body, resp, nil
		}
		lastBody = body
		lastResp = resp
		lastErr = err

		if attempt < attempts {
			// Log and sleep with exponential backoff
			t.Logf("GetLocalHTTPResponseWithBackoff attempt %d/%d failed for %s: %v; retrying after %s", attempt, attempts, rawurl, err, delay)
			time.Sleep(delay)
			// Double the delay for next attempt
			delay *= 2
		}
	}

	return lastBody, lastResp, lastErr
}

// EnsureLocalHTTPContent will verify a URL responds with a 200, expected content string, and optional parameters.
// Parameters can be either:
// - HTTPRequestOpts struct with TimeoutSeconds and MaxRetries fields
// - int representing timeout seconds (for backward compatibility)
func EnsureLocalHTTPContent(t *testing.T, rawurl string, expectedContent string, params ...interface{}) (*http.Response, error) {
	options := parseHTTPRequestOpts(40, params...)
	assert := asrt.New(t)

	body, resp, err := GetLocalHTTPResponse(t, rawurl, options)
	// We see intermittent php-fpm SIGBUS failures, only on macOS.
	// That results in a 502/503. If we get a 502/503 on macOS, try again.
	// It seems to be a 502 with nginx-fpm and a 503 with apache-fpm
	if nodeps.IsMacOS() && resp != nil && (resp.StatusCode >= 500) {
		t.Logf("Received %d error on macOS, retrying GetLocalHTTPResponse", resp.StatusCode)
		time.Sleep(time.Second)
		body, resp, err = GetLocalHTTPResponse(t, rawurl, options)
	}
	assert.NoError(err, "GetLocalHTTPResponse returned err on rawurl %s, resp=%v, body=%v: %v", rawurl, resp, body, err)
	assert.Contains(body, expectedContent, "request %s got resp=%v, body:\n========\n%s\n==========\n", rawurl, resp, body)
	return resp, err
}

// CheckGoroutineOutput makes sure that goroutines
// aren't beyond specified level
func CheckGoroutineOutput(t *testing.T, out string) {
	goroutineLimit := nodeps.GoroutineLimit
	// regex to find "goroutines=4 at exit of main()"
	re := regexp.MustCompile(`goroutines=(\d+) at exit of main\(\)`)
	matches := re.FindAllStringSubmatch(out, -1)
	require.Equal(t, 1, len(matches), "must be exactly one match for goroutines=<value>, DDEV_GOROUTINES=%s actual output='%s'", os.Getenv(`DDEV_GOROUTINES`), out)
	num, err := strconv.Atoi(matches[0][1])
	require.NoError(t, err, "can't convert %s to number: %v", matches[0][1])
	require.LessOrEqual(t, num, goroutineLimit, "number of goroutines=%v, higher than limit=%d", num, goroutineLimit)
}

// PortPair is for tests to use naming portsets for tests
type PortPair struct {
	HTTPPort  string
	HTTPSPort string
}

// parseHTTPRequestOpts extracts HTTPRequestOpts from interface{} parameters with defaults
func parseHTTPRequestOpts(defaultTimeout int, params ...interface{}) HTTPRequestOpts {
	var options HTTPRequestOpts

	// Handle different parameter types for backward compatibility
	if len(params) > 0 {
		switch param := params[0].(type) {
		case HTTPRequestOpts:
			options = param
		case int:
			// Backward compatibility: int parameter is timeout in seconds
			options.TimeoutSeconds = param
		default:
			options.TimeoutSeconds = defaultTimeout // Default if invalid type
		}
	}
	// Set defaults if not provided
	if options.TimeoutSeconds == 0 {
		options.TimeoutSeconds = defaultTimeout
	}
	// Negative timeout means no timeout
	if options.TimeoutSeconds < 0 {
		options.TimeoutSeconds = 0
	}
	if options.MaxRetries < 1 {
		options.MaxRetries = 1
	}

	return options
}
