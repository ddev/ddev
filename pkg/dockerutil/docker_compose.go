package dockerutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/compose/v5/cmd/display"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/docker/compose/v5/pkg/compose"
	"github.com/mattn/go-isatty"
	"github.com/sirupsen/logrus"
)

// LoadComposeProject loads a compose project from files using the upstream compose library.
// opts.ConfigPaths is set from files; opts.WorkingDir defaults to filepath.Dir(files[0]) if empty.
// Uses the cached singleton compose service (no stream/progress output) and its background context.
func LoadComposeProject(files []string, opts api.ProjectLoadOptions) (*types.Project, error) {
	if opts.ProjectName == "" {
		return nil, errors.New("LoadComposeProject: ProjectName must not be empty")
	}
	if opts.WorkingDir == "" && len(files) > 0 {
		opts.WorkingDir = filepath.Dir(files[0])
	}
	opts.ConfigPaths = files
	dm, err := getDockerManagerInstance()
	if err != nil {
		return nil, err
	}
	return dm.composeForLoad.LoadProject(dm.goContext, opts)
}

// NewComposeService creates a compose service backed by the singleton Docker CLI.
func NewComposeService() (context.Context, api.Compose, error) {
	return NewComposeServiceWithStreams(output.UserErr.Out, output.UserErr.Out)
}

// NewComposeServiceWithStreams creates a compose service backed by the singleton Docker CLI.
func NewComposeServiceWithStreams(stdout, stderr io.Writer) (context.Context, api.Compose, error) {
	dm, err := getDockerManagerInstance()
	if err != nil {
		return nil, nil, err
	}
	var ep api.EventProcessor
	if !output.JSONOutput && isatty.IsTerminal(os.Stdout.Fd()) {
		ep = display.Full(stdout, stderr, false)
	} else {
		progressOut := output.UserErr.Out
		if output.JSONOutput {
			progressOut = &output.JSONProgressWriter{}
		}
		ep = display.Plain(progressOut)
	}
	opts := []compose.Option{
		compose.WithOutputStream(stdout),
		compose.WithErrorStream(stderr),
		compose.WithEventProcessor(ep),
	}
	svc, err := compose.NewComposeService(dm.cli, opts...)
	return dm.goContext, svc, err
}

// ExitCodeToError converts the (exitCode, err) return of api.Compose.Exec /
// RunOneOffContainer into a single error usable with errors.As(&cli.StatusError{}).
//
// docker/cli's container.RunExec returns cli.StatusError with StatusCode set
// but an empty Status field
// (vendor/github.com/docker/cli/cli/command/container/exec.go:215),
// so callers that print err.Error() see an empty string. This helper attaches
// a default "exit status N" message in that case while preserving any non-empty
// Status the upstream layer already provided.
func ExitCodeToError(exitCode int, err error) error {
	if exitCode == 0 {
		return err
	}
	msg := fmt.Sprintf("exit status %d", exitCode)
	if err != nil && err.Error() != "" {
		msg = err.Error()
	}
	return cli.StatusError{StatusCode: exitCode, Status: msg}
}

// CaptureOutput runs fn against a compose service whose stdout/stderr are buffered,
// returning the captured strings. Use only when output text is genuinely needed (e.g. build retry detection).
func CaptureOutput(fn func(svc api.Compose) error) (string, string, error) {
	var stdoutBuf, stderrBuf bytes.Buffer
	_, svc, err := NewComposeServiceWithStreams(&stdoutBuf, &stderrBuf)
	if err != nil {
		return "", "", err
	}
	err = fn(svc)
	return cleanOutput(stdoutBuf.String()), cleanOutput(stderrBuf.String()), err
}

// cleanOutput strips ANSI escape codes and carriage-return overwrite sequences so
// captured compose output is safe to embed in log messages and errors.
func cleanOutput(s string) string {
	s = regexp.MustCompile(`\x1b[^a-zA-Z]*[a-zA-Z]`).ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// SetExecStdin installs `in` as the stdin of the singleton DockerCli for the
// duration of one compose Exec/Run call, returning a restore closure the
// caller MUST defer.
//
// Why mutate the singleton instead of using compose.WithInputStream:
// vendor/github.com/docker/compose/v5/pkg/compose/compose.go wraps any
// caller-supplied input through a readCloserAdapter inside wrapDockerCliWithStreams
// (https://github.com/docker/compose/blob/v5.1.3/pkg/compose/compose.go#L283-L301).
// That adapter drops the underlying *os.File, so streams.In.SetRawTerminal()
// — which docker/cli's hijack setupInput needs for TTY exec — fails. The only
// way to give compose a *os.File-backed streams.In today is to set it on the
// shared DockerCli before the call. ddev's CLI is single-threaded for exec, so
// the singleton mutation is safe in practice; this function is NOT safe for
// concurrent use.
//
// On TTY paths we also hand compose a dup of the file descriptor so the
// restoreTerminal->in.Close() call in
// vendor/github.com/docker/cli/cli/command/container/hijack.go (line 211)
// (https://github.com/docker/cli/blob/v29.4.0/cli/command/container/hijack.go#L211)
// lands on the dup, not on the ddev process's real fd 0. The dup is closed
// here on restore so it does not leak when compose declines to close it
// (darwin, windows) or on non-TTY exec where setupInput returns a no-op
// restore (line 96 in the same file).
func SetExecStdin(in io.ReadCloser, tty bool) (restore func(), err error) {
	dm, err := getDockerManagerInstance()
	if err != nil {
		return func() {}, err
	}
	var dupFile *os.File
	if tty {
		if f, ok := in.(*os.File); ok {
			if dup, dupErr := dupStdin(f); dupErr == nil && dup != f {
				// dup != f filters out the windows pass-through, where dupStdin
				// returns the original *os.File and we must not close it.
				in = dup
				dupFile = dup
			}
		}
	}
	prevIn := dm.cli.In()
	dm.cli.SetIn(streams.NewIn(in))
	return func() {
		dm.cli.SetIn(prevIn)
		if dupFile != nil {
			// Compose's hijack may have already closed the underlying fd on
			// linux TTY paths; *os.File.Close on an already-closed fd returns
			// an error that is safe to discard.
			_ = dupFile.Close()
		}
	}, nil
}

// CreateComposeProject creates a compose project from a YAML string.
func CreateComposeProject(yamlStr string) (*types.Project, error) {
	project, err := loader.LoadWithContext(
		context.Background(),
		types.ConfigDetails{
			ConfigFiles: []types.ConfigFile{
				{Content: []byte(yamlStr)},
			},
		},
		loader.WithProfiles([]string{`*`}),
	)
	if err != nil {
		return project, err
	}
	// Initialize Networks, Services, and Volumes to empty maps if nil
	if project.Networks == nil {
		project.Networks = types.Networks{}
	}
	if project.Services == nil {
		project.Services = types.Services{}
	}
	if project.Volumes == nil {
		project.Volumes = types.Volumes{}
	}
	// Ensure nested fields like Labels, Networks, and Environment are initialized
	for name, network := range project.Networks {
		if network.Labels == nil {
			network.Labels = types.Labels{}
		}
		project.Networks[name] = network
	}
	for name, service := range project.Services {
		if service.Networks == nil {
			service.Networks = map[string]*types.ServiceNetworkConfig{}
		}
		if service.Environment == nil {
			service.Environment = types.MappingWithEquals{}
		}
		project.Services[name] = service
	}
	return project, nil
}

// PullImages pulls images in parallel if they don't exist locally.
// If pullAlways is true, it will always pull.
func PullImages(images []string, pullAlways bool) error {
	if len(images) == 0 {
		return nil
	}

	// Build a minimal project directly without a YAML round-trip.
	project := &types.Project{
		Name:     "compose-yaml-pull",
		Services: types.Services{},
	}

	for _, image := range images {
		if image == "" {
			continue
		}
		if !pullAlways {
			if imageExists, _ := ImageExistsLocally(image); imageExists {
				continue
			}
		}
		service := sanitizeServiceName(image)
		if _, exists := project.Services[service]; exists {
			continue
		}
		project.Services[service] = types.ServiceConfig{
			Image: image,
		}
		util.Debug(`Pulling image for %s ("%s" service)`, image, service)
	}

	if len(project.Services) == 0 {
		util.Debug("All images already exist locally, no pull needed")
		return nil
	}

	pullCtx, pullSvc, pullErr := NewComposeService()
	if pullErr != nil {
		return pullErr
	}
	return pullSvc.Pull(pullCtx, project, api.PullOptions{})
}

// Pull pulls image if it doesn't exist locally.
func Pull(image string) error {
	return PullImages([]string{image}, false)
}

// sanitizeServiceName sanitizes a string to be a valid Docker Compose service name
// by replacing any characters that don't match [a-zA-Z0-9._-] with hyphens.
// See https://github.com/compose-spec/compose-go/blob/main/schema/compose-spec.json for allowed pattern.
func sanitizeServiceName(input string) string {
	if input == "" {
		return ""
	}

	invalidChars := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	sanitized := invalidChars.ReplaceAllString(input, "-")

	multipleHyphens := regexp.MustCompile(`-+`)
	sanitized = multipleHyphens.ReplaceAllString(sanitized, "-")

	sanitized = strings.Trim(sanitized, "-")

	return sanitized
}

// suppressedLogrusMessages are substrings that, when matched against a logrus
// entry's Message, cause the entry to be silently dropped at format time. Used
// to silence known-harmless compose noise (notably the project-level
// "Warning: No resource found to remove" warning emitted by compose Down at
// vendor/github.com/docker/compose/v5/pkg/compose/down.go:117).
var suppressedLogrusMessages = []string{
	"No resource found to remove",
}

// suppressLogrusFormatter wraps an underlying logrus.Formatter and returns
// (nil, nil) for entries whose Message matches a suppressedLogrusMessages
// substring. Returning empty bytes makes logrus's downstream Out.Write a
// no-op, suppressing the entry without touching the output writer or the log
// level. Non-matching entries are formatted by the underlying formatter.
//
// Installed once at package init via setupLogrusSuppression so we don't
// mutate global state per call. Compose hardwires logrus.StandardLogger,
// so a global filter is the only knob available; making it permanent
// removes the per-call SetOutput race the previous implementation had.
type suppressLogrusFormatter struct {
	underlying logrus.Formatter
	suppress   []string
}

func (f *suppressLogrusFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	for _, s := range f.suppress {
		if strings.Contains(entry.Message, s) {
			return nil, nil
		}
	}
	return f.underlying.Format(entry)
}

// setupLogrusSuppression installs a suppressLogrusFormatter on the given
// logger if one is not already installed. Idempotent; safe to call multiple
// times.
func setupLogrusSuppression(logger *logrus.Logger) {
	if _, alreadyWrapped := logger.Formatter.(*suppressLogrusFormatter); alreadyWrapped {
		return
	}
	logger.SetFormatter(&suppressLogrusFormatter{
		underlying: logger.Formatter,
		suppress:   suppressedLogrusMessages,
	})
}

func init() {
	setupLogrusSuppression(logrus.StandardLogger())
}
