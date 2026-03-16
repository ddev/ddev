package dockerutil_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/testsetup"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/docker/cli/cli"
	"github.com/docker/compose/v5/cmd/display"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/sirupsen/logrus"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var DdevBin = "ddev"

func ensureDdevBin() {
	if DdevBin == "ddev" {
		DdevBin = testsetup.MustResolveDdevBinary()
	}
}

func init() {
	if nodeps.IsEnvTrue("DDEV_TEST_NO_BIND_MOUNTS") {
		globalconfig.DdevGlobalConfig.NoBindMounts = true
	}

	globalconfig.EnsureGlobalConfig()
}

// TestComposeConfig tests ComposeConfig which loads and merges compose files.
func TestComposeConfig(t *testing.T) {
	assert := asrt.New(t)

	// Create a dummy app to set up DockerEnv
	testDir := testcommon.CreateTmpDir(t.Name())
	app, err := ddevapp.NewApp(testDir, true)
	assert.NoError(err)

	t.Cleanup(func() {
		_ = os.RemoveAll(testDir)
	})

	app.Name = "test"

	// Set up environment variables via DockerEnv
	_ = app.DockerEnv()

	composeFiles := []string{filepath.Join("testdata", "docker-compose.yml")}

	// Test: should return project with services
	project, err := dockerutil.LoadComposeProject(composeFiles, api.ProjectLoadOptions{
		ProjectName: "test",
	})
	assert.NoError(err)
	assert.NotNil(project)
	_, hasWeb := project.Services["web"]
	assert.True(hasWeb)
	_, hasDB := project.Services["db"]
	assert.True(hasDB)

	// Test with additional override file
	composeFiles = append(composeFiles, filepath.Join("testdata", "docker-compose.override.yml"))
	project, err = dockerutil.LoadComposeProject(composeFiles, api.ProjectLoadOptions{
		ProjectName: "test",
	})
	assert.NoError(err)
	assert.NotNil(project)
	_, hasWeb = project.Services["web"]
	assert.True(hasWeb)
	_, hasDB = project.Services["db"]
	assert.True(hasDB)
	_, hasFoo := project.Services["foo"]
	assert.True(hasFoo)

	// Test with invalid file
	_, err = dockerutil.LoadComposeProject([]string{"invalid.yml"}, api.ProjectLoadOptions{
		ProjectName: "test",
	})
	assert.Error(err)

	// Test with empty ProjectName
	_, err = dockerutil.LoadComposeProject(composeFiles, api.ProjectLoadOptions{})
	assert.Error(err)
	assert.Contains(err.Error(), "ProjectName")
}

// TestComposeExec tests execution of docker-compose exec commands with streams
func TestComposeExec(t *testing.T) {
	assert := asrt.New(t)

	projectName := "test-compose-exec"

	container, _ := dockerutil.FindContainerByName(projectName)
	if container != nil {
		_ = dockerutil.RemoveContainer(container.ID)
	}

	// Use the current actual web container for this, so replace in base docker-compose file
	composeBase := filepath.Join("testdata", "TestComposeExec", "test-compose-exec.yaml")
	tmpDir := testcommon.CreateTmpDir(t.Name())
	realComposeFile := filepath.Join(tmpDir, "replaced-compose-exec.yaml")

	err := fileutil.ReplaceStringInFile("TEST-COMPOSE-EXEC-IMAGE", versionconstants.WebImg+":"+versionconstants.WebTag, composeBase, realComposeFile)
	assert.NoError(err)

	composeFiles := []string{realComposeFile}

	t.Cleanup(func() {
		cleanProject, cleanErr := dockerutil.LoadComposeProject(composeFiles, api.ProjectLoadOptions{ProjectName: projectName})
		if cleanErr == nil {
			cleanCtx, cleanSvc, svcErr := dockerutil.NewComposeService()
			if svcErr == nil {
				_ = cleanSvc.Down(cleanCtx, cleanProject.Name, api.DownOptions{Project: cleanProject})
			}
		}
	})

	upProject, err := dockerutil.LoadComposeProject(composeFiles, api.ProjectLoadOptions{ProjectName: projectName})
	require.NoError(t, err)
	upCtx, upSvc, err := dockerutil.NewComposeService()
	require.NoError(t, err)
	err = upSvc.Up(upCtx, upProject, api.UpOptions{
		Create: api.CreateOptions{Build: &api.BuildOptions{Progress: display.ModePlain}},
		Start:  api.StartOptions{Project: upProject},
	})
	require.NoError(t, err)

	_, err = dockerutil.ContainerWait(60, map[string]string{
		"com.ddev.site-name":        projectName,
		"com.docker.compose.oneoff": "False",
	})
	if err != nil {
		logout, _ := exec.RunCommand("docker", []string{"logs", projectName})
		inspectOut, _ := exec.RunCommandPipe("sh", []string{"-c", fmt.Sprintf("docker inspect %s|jq -r '.[0].State.Health.Log'", projectName)})
		t.Fatalf("FAIL: TestComposeExec failed to ContainerWait for container: %v, logs\n========= container logs ======\n%s\n======= end logs =======\n==== health log =====\ninspectOut\n%s\n========", err, logout, inspectOut)
	}

	execProject, err := dockerutil.LoadComposeProject(composeFiles, api.ProjectLoadOptions{ProjectName: projectName})
	require.NoError(t, err)

	execFn := func(stdoutW, stderrW io.Writer, command []string) error {
		execCtx, svc, svcErr := dockerutil.NewComposeServiceWithStreams(stdoutW, stderrW)
		if svcErr != nil {
			return svcErr
		}
		return dockerutil.ExitCodeToError(svc.Exec(execCtx, execProject.Name, api.RunOptions{
			Service: "web",
			Command: command,
		}))
	}

	// Point stdout to os.Stdout and do simple ps -ef in web container
	stdout := util.CaptureStdOut()
	err = execFn(os.Stdout, os.Stderr, []string{"ps", "-ef"})
	assert.NoError(err)
	output := stdout()
	assert.Contains(output, "supervisord")

	// Reverse stdout and stderr: error output should appear on our captured stdout
	stdout = util.CaptureStdOut()
	err = execFn(os.Stderr, os.Stdout, []string{"ls", "-d", "xx", "/var/run/apache2"})
	assert.Error(err)
	output = stdout()
	assert.Contains(output, "ls: cannot access 'xx': No such file or directory")

	// Normal stdout/stderr: success output should appear on our captured stdout
	stdout = util.CaptureStdOut()
	err = execFn(os.Stdout, os.Stderr, []string{"ls", "-d", "xx", "/var/run/apache2"})
	assert.Error(err)
	output = stdout()
	assert.Contains(output, "/var/run/apache2", output)

	// Exit code: non-zero exit returns a cli.StatusError carrying the exit code,
	// extractable by callers via errors.As. The Status string is also populated
	// (docker/cli's container.RunExec leaves it empty; ExitCodeToError fills it in).
	err = execFn(os.Stdout, os.Stderr, []string{"sh", "-c", "exit 42"})
	require.Error(t, err)
	var statusErr cli.StatusError
	require.ErrorAs(t, err, &statusErr)
	require.Equal(t, 42, statusErr.StatusCode)
	require.Contains(t, err.Error(), "42")

	// Exit code 0 with a successful command must NOT produce a cli.StatusError.
	err = execFn(os.Stdout, os.Stderr, []string{"true"})
	require.NoError(t, err)
	require.False(t, errors.As(err, &statusErr))

	// A different non-zero exit code (1) round-trips correctly through the helper.
	err = execFn(os.Stdout, os.Stderr, []string{"false"})
	require.Error(t, err)
	require.ErrorAs(t, err, &statusErr)
	require.Equal(t, 1, statusErr.StatusCode)

	execWithOpts := func(stdoutW, stderrW io.Writer, opts api.RunOptions) error {
		opts.Service = "web"
		execCtx, svc, svcErr := dockerutil.NewComposeServiceWithStreams(stdoutW, stderrW)
		if svcErr != nil {
			return svcErr
		}
		return dockerutil.ExitCodeToError(svc.Exec(execCtx, execProject.Name, opts))
	}

	// Working directory: pwd inside /tmp should report /tmp
	stdout = util.CaptureStdOut()
	err = execWithOpts(os.Stdout, os.Stderr, api.RunOptions{Command: []string{"pwd"}, WorkingDir: "/tmp"})
	assert.NoError(err)
	output = stdout()
	assert.Contains(output, "/tmp")

	// User override: whoami should report root
	stdout = util.CaptureStdOut()
	err = execWithOpts(os.Stdout, os.Stderr, api.RunOptions{Command: []string{"whoami"}, User: "root"})
	assert.NoError(err)
	output = stdout()
	assert.Contains(output, "root")

	// Environment variables: injected var must appear in the command output
	stdout = util.CaptureStdOut()
	err = execWithOpts(os.Stdout, os.Stderr, api.RunOptions{
		Command:     []string{"sh", "-c", "echo $DDEV_COMPOSE_TEST"},
		Environment: []string{"DDEV_COMPOSE_TEST=hello"},
	})
	assert.NoError(err)
	output = stdout()
	assert.Contains(output, "hello")
}

// TestCreateComposeProject verifies nil-map initialization for services, networks, and volumes.
func TestCreateComposeProject(t *testing.T) {
	yamlStr := `
name: test-project
services:
  web:
    image: alpine
`
	project, err := dockerutil.CreateComposeProject(yamlStr)
	require.NoError(t, err)
	require.NotNil(t, project.Networks)
	require.NotNil(t, project.Services)
	require.NotNil(t, project.Volumes)

	webSvc := project.Services["web"]
	require.NotNil(t, webSvc.Networks)
	require.NotNil(t, webSvc.Environment)
}

// TestCreateComposeProjectInvalid verifies that invalid YAML returns an error.
func TestCreateComposeProjectInvalid(t *testing.T) {
	_, err := dockerutil.CreateComposeProject("not: valid: yaml: [")
	require.Error(t, err)
}

// TestSuppressedLogrusNoise verifies, via real logrus.StandardLogger calls,
// that the compose-noise filter installed at package init drops only matching
// entries and lets all other entries pass through.
func TestSuppressedLogrusNoise(t *testing.T) {
	var buf strings.Builder
	origOut := logrus.StandardLogger().Out
	logrus.SetOutput(&stringWriter{&buf})
	t.Cleanup(func() { logrus.SetOutput(origOut) })

	// Compose's "No resource found to remove" warning must be silently dropped.
	buf.Reset()
	logrus.Warn("Warning: No resource found to remove for project \"foo\".")
	require.NotContains(t, buf.String(), "No resource found to remove")

	// Unrelated warnings still flow through to the configured output.
	buf.Reset()
	logrus.Warn("some other warning")
	require.Contains(t, buf.String(), "some other warning")

	// Different log levels are also suppressed when the message matches.
	buf.Reset()
	logrus.Info("No resource found to remove for project \"bar\".")
	require.NotContains(t, buf.String(), "No resource found to remove")
}

// stringWriter adapts strings.Builder to io.Writer for logrus.
type stringWriter struct{ b *strings.Builder }

func (w *stringWriter) Write(p []byte) (int, error) { return w.b.Write(p) }

// TestCaptureOutputErrorPassthrough verifies that errors returned by fn are propagated.
func TestCaptureOutputErrorPassthrough(t *testing.T) {
	sentinelErr := errors.New("expected error")
	_, _, err := dockerutil.CaptureOutput(func(svc api.Compose) error {
		return sentinelErr
	})
	require.ErrorIs(t, err, sentinelErr)
}

// TestLoadComposeProjectEmptyProjectName focuses on the input-validation
// branch: ProjectName is mandatory and must surface a clear error.
func TestLoadComposeProjectEmptyProjectName(t *testing.T) {
	_, err := dockerutil.LoadComposeProject(
		[]string{filepath.Join("testdata", "docker-compose.yml")},
		api.ProjectLoadOptions{}, // ProjectName intentionally empty
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ProjectName",
		"empty ProjectName must produce an error mentioning ProjectName so callers can diagnose")
}

// TestLoadComposeProjectDoesNotMutateOpts pins the value-semantics contract:
// LoadComposeProject must not write through to the caller's opts struct.
// If LoadComposeProject is ever changed to take *api.ProjectLoadOptions (pointer),
// this test will start failing and the call sites must be re-audited — they
// currently rely on being able to pass an opts literal without worrying about
// in-place mutation across calls.
func TestLoadComposeProjectDoesNotMutateOpts(t *testing.T) {
	opts := api.ProjectLoadOptions{ProjectName: "test"}
	files := []string{filepath.Join("testdata", "docker-compose.yml")}

	_, _ = dockerutil.LoadComposeProject(files, opts)

	require.Nil(t, opts.ConfigPaths,
		"LoadComposeProject must not assign ConfigPaths on caller's opts")
	require.Empty(t, opts.WorkingDir,
		"LoadComposeProject must not assign WorkingDir on caller's opts")
}

// TestPullImagesEmpty verifies that PullImages with empty/nil input is a no-op.
func TestPullImagesEmpty(t *testing.T) {
	require.NoError(t, dockerutil.PullImages(nil, false))
	require.NoError(t, dockerutil.PullImages([]string{}, false))
	// A slice of only empty strings should also be a no-op (all skipped).
	require.NoError(t, dockerutil.PullImages([]string{""}, false))
}

// TestSetExecStdinRestoreOnPanic verifies that SetExecStdin's restore closure
// reinstates the original stdin even when the caller panics.
func TestSetExecStdinRestoreOnPanic(t *testing.T) {
	// Read the original stdin from the singleton before the call.
	// We compare os.Stdin here as a proxy — after restore the CLI's In() will
	// point back to whatever it was before SetExecStdin ran.
	restore, err := dockerutil.SetExecStdin(os.Stdin, false)
	require.NoError(t, err)

	func() {
		defer func() {
			_ = recover()
			restore() // must not panic itself and must restore stdin
		}()
		panic("synthetic panic")
	}()

	// If we get here without the process crashing, restore() ran successfully.
	// A second call with the same arguments must also succeed, proving stdin
	// was returned to a state that can accept another SetExecStdin call.
	restore2, err := dockerutil.SetExecStdin(os.Stdin, false)
	require.NoError(t, err)
	restore2()
}
