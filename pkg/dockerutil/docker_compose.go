package dockerutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/compose/v5/cmd/display"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/docker/compose/v5/pkg/compose"
	"github.com/joho/godotenv"
	"github.com/mattn/go-isatty"
	"github.com/sirupsen/logrus"
)

// ComposeUpOpts holds options for ComposeUp.
type ComposeUpOpts struct {
	ComposeFiles []string
	Project      *types.Project // Preloaded project; if set, ComposeFiles is ignored
	ProjectName  string
	Profiles     []string
	Build        bool // Build images before starting
	Progress     string
}

// ComposeDownOpts holds options for ComposeDown.
type ComposeDownOpts struct {
	ComposeFiles  []string
	Project       *types.Project
	ProjectName   string
	Profiles      []string
	RemoveOrphans bool
	Progress      string
}

// ComposeStopOpts holds options for ComposeStop.
type ComposeStopOpts struct {
	ComposeFiles []string
	Project      *types.Project
	ProjectName  string
	Profiles     []string
	Progress     string
}

// ComposeBuildOpts holds options for ComposeBuild.
type ComposeBuildOpts struct {
	ComposeFiles []string
	ProjectName  string
	Services     []string // Specific services to build; nil means all
	NoCache      bool
	Progress     string
	ShowDots     bool // Show dots on stderr while building
	Timeout      time.Duration
	Stdout       io.Writer // If set, stream output here; nil means capture
	Stderr       io.Writer
}

// ComposeExecOpts holds options for ComposeExec.
type ComposeExecOpts struct {
	ComposeFiles []string
	ProjectName  string
	Service      string
	Command      []string
	Tty          bool
	Interactive  bool
	Detach       bool
	User         string
	WorkDir      string
	Env          []string
	Stdin        io.Reader // If set, use for streaming input
	Stdout       io.Writer // If set, stream output here; nil means capture
	Stderr       io.Writer
}

// ComposeConfigOpts holds options for ComposeConfig.
type ComposeConfigOpts struct {
	ComposeFiles []string
	ProjectName  string
	Profiles     []string
	EnvFiles     []string
}

// ComposePullOpts holds options for ComposePull.
type ComposePullOpts struct {
	ComposeFiles []string
	Project      *types.Project // Preloaded project; if set, ComposeFiles is ignored
	ProjectName  string
	Progress     string
	Stdout       io.Writer // If nil, output is discarded
	Stderr       io.Writer
}

// ComposeUp starts compose services in detached mode.
func ComposeUp(opts ComposeUpOpts) error {
	ctx := context.Background()

	project, err := loadProject(ctx, opts.ComposeFiles, opts.Project, opts.ProjectName, opts.Profiles, nil)
	if err != nil {
		return err
	}

	svc, err := newComposeService(io.Discard, io.Discard, progressOpts(opts.Progress)...)
	if err != nil {
		return err
	}

	var createOpts api.CreateOptions
	if opts.Build {
		buildOpts := api.BuildOptions{}
		createOpts.Build = &buildOpts
	}
	return svc.Up(ctx, project, api.UpOptions{
		Create: createOpts,
		Start:  api.StartOptions{Project: project},
	})
}

// ComposeDown stops and removes containers and networks for a compose project.
func ComposeDown(opts ComposeDownOpts) error {
	ctx := context.Background()

	project, err := loadProject(ctx, opts.ComposeFiles, opts.Project, opts.ProjectName, opts.Profiles, nil)
	if err != nil {
		return err
	}

	// Temporarily suppress logrus warnings (e.g., "No resource found to remove")
	// since these are expected when resources don't exist
	originalLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.ErrorLevel)
	defer logrus.SetLevel(originalLevel)

	svc, err := newComposeService(io.Discard, io.Discard, progressOpts(opts.Progress)...)
	if err != nil {
		return err
	}
	return svc.Down(ctx, project.Name, api.DownOptions{
		RemoveOrphans: opts.RemoveOrphans,
		Project:       project,
	})
}

// ComposeStop stops services without removing them.
func ComposeStop(opts ComposeStopOpts) error {
	ctx := context.Background()

	project, err := loadProject(ctx, opts.ComposeFiles, opts.Project, opts.ProjectName, opts.Profiles, nil)
	if err != nil {
		return err
	}

	svc, err := newComposeService(io.Discard, io.Discard, progressOpts(opts.Progress)...)
	if err != nil {
		return err
	}
	return svc.Stop(ctx, project.Name, api.StopOptions{Project: project})
}

// ComposeRestartOpts holds options for ComposeRestart.
type ComposeRestartOpts struct {
	ComposeFiles []string
	Project      *types.Project
	ProjectName  string
	Profiles     []string
	Services     []string      // Specific services to restart; nil means all
	Timeout      time.Duration // Timeout for graceful shutdown
	Progress     string        // Progress mode
}

// ComposeRestart restarts services.
func ComposeRestart(opts ComposeRestartOpts) error {
	ctx := context.Background()

	project, err := loadProject(ctx, opts.ComposeFiles, opts.Project, opts.ProjectName, opts.Profiles, nil)
	if err != nil {
		return err
	}

	svc, err := newComposeService(io.Discard, io.Discard, progressOpts(opts.Progress)...)
	if err != nil {
		return err
	}

	var timeout *time.Duration
	if opts.Timeout > 0 {
		timeout = &opts.Timeout
	}

	return svc.Restart(ctx, project.Name, api.RestartOptions{
		Project:  project,
		Services: opts.Services,
		Timeout:  timeout,
	})
}

// ComposeBuild builds service images.
// When opts.Stdout is set, output streams there; otherwise stdout and stderr are captured and returned.
func ComposeBuild(opts ComposeBuildOpts) (string, string, error) {
	ctx := context.Background()
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	project, err := loadProject(ctx, opts.ComposeFiles, nil, opts.ProjectName, nil, nil)
	if err != nil {
		return "", "", err
	}

	var stdoutW, stderrW io.Writer
	var stdoutBuf, stderrBuf bytes.Buffer
	if opts.Stdout != nil {
		stdoutW = opts.Stdout
		stderrW = opts.Stderr
	} else {
		stdoutW = &stdoutBuf
		stderrW = &stderrBuf
	}

	svc, err := newComposeService(stdoutW, stderrW, progressOpts(opts.Progress)...)
	if err != nil {
		return "", "", err
	}

	var dotsDone chan bool
	if opts.ShowDots {
		dotsDone = util.ShowDots()
	}
	err = svc.Build(ctx, project, api.BuildOptions{
		Progress: opts.Progress,
		NoCache:  opts.NoCache,
		Services: opts.Services,
	})
	if opts.ShowDots {
		dotsDone <- true
	}
	if ctx.Err() != nil {
		return stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("ComposeBuild timed out after %v: %v", opts.Timeout, err)
	}
	return stdoutBuf.String(), stderrBuf.String(), err
}

// ComposeExec runs a command in a running service container.
// When opts.Stdout is set, output streams there and empty strings are returned.
// Otherwise stdout and stderr are captured and returned.
func ComposeExec(opts ComposeExecOpts) (string, string, error) {
	ctx := context.Background()

	project, err := loadProject(ctx, opts.ComposeFiles, nil, opts.ProjectName, nil, nil)
	if err != nil {
		return "", "", err
	}

	var stdoutW, stderrW io.Writer
	var stdoutBuf, stderrBuf bytes.Buffer
	if opts.Stdout != nil {
		stdoutW = opts.Stdout
		stderrW = opts.Stderr
	} else {
		stdoutW = &stdoutBuf
		stderrW = &stderrBuf
	}

	if opts.Tty {
		// For TTY exec, set the DockerCli stdin directly instead of using
		// compose.WithInputStream, which wraps stdin in a readCloserAdapter
		// that hides the file descriptor from TTY detection (term.GetFdInfo
		// only recognizes *os.File, not the adapter).
		dm, err2 := getDockerManagerInstance()
		if err2 != nil {
			return "", "", err2
		}
		if stdin, ok := opts.Stdin.(io.ReadCloser); ok {
			dm.cli.SetIn(streams.NewIn(stdin))
		}
	}
	var extraOpts []compose.Option
	if !opts.Tty && opts.Stdin != nil {
		extraOpts = append(extraOpts, compose.WithInputStream(opts.Stdin))
	}
	svc, err := newComposeService(stdoutW, stderrW, extraOpts...)
	if err != nil {
		return "", "", err
	}

	_, err = svc.Exec(ctx, project.Name, api.RunOptions{
		Service:     opts.Service,
		Command:     opts.Command,
		Tty:         opts.Tty,
		Interactive: opts.Interactive,
		Detach:      opts.Detach,
		WorkingDir:  opts.WorkDir,
		User:        opts.User,
		Environment: opts.Env,
	})
	return stdoutBuf.String(), stderrBuf.String(), err
}

// ComposeConfig loads and merges compose files and returns the project.
func ComposeConfig(opts ComposeConfigOpts) (*types.Project, error) {
	ctx := context.Background()

	project, err := loadProject(ctx, opts.ComposeFiles, nil, opts.ProjectName, opts.Profiles, opts.EnvFiles)
	if err != nil {
		return nil, err
	}

	// Check container name consistency
	err = project.CheckContainerNameUnicity()
	if err != nil {
		return nil, err
	}

	return project, nil
}

// ComposePull pulls service images.
func ComposePull(opts ComposePullOpts) error {
	ctx := context.Background()

	project, err := loadProject(ctx, opts.ComposeFiles, opts.Project, opts.ProjectName, nil, nil)
	if err != nil {
		return err
	}

	svc, err := newComposeService(os.Stdout, os.Stderr, progressOpts(opts.Progress)...)
	if err != nil {
		return err
	}
	return svc.Pull(ctx, project, api.PullOptions{})
}

// loadProject returns a compose project, either the preloaded one or loaded from files.
func loadProject(ctx context.Context, composeFiles []string, project *types.Project, projectName string, profiles []string, envFiles []string) (*types.Project, error) {
	if project != nil {
		if projectName != "" {
			project.Name = projectName
		}
		setCustomLabels(project)
		return project, nil
	}

	environment := make(types.Mapping)
	for _, e := range os.Environ() {
		if k, v, ok := strings.Cut(e, "="); ok {
			environment[k] = v
		}
	}
	for _, envFile := range envFiles {
		envMap, err := godotenv.Read(envFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load env file %s: %v", envFile, err)
		}
		for k, v := range envMap {
			environment[k] = v
		}
	}

	configFiles := make([]types.ConfigFile, 0, len(composeFiles))
	for _, f := range composeFiles {
		configFiles = append(configFiles, types.ConfigFile{Filename: f})
	}

	workingDir := ""
	if len(composeFiles) > 0 {
		workingDir = filepath.Dir(composeFiles[0])
	}

	var loaderOpts []func(*loader.Options)
	if len(profiles) > 0 {
		loaderOpts = append(loaderOpts, loader.WithProfiles(profiles))
	}
	if projectName != "" {
		pn := projectName
		loaderOpts = append(loaderOpts, func(opts *loader.Options) {
			opts.SetProjectName(pn, true)
		})
	}

	p, err := loader.LoadWithContext(ctx, types.ConfigDetails{
		ConfigFiles: configFiles,
		Environment: environment,
		WorkingDir:  workingDir,
	}, loaderOpts...)
	if err != nil {
		return nil, err
	}
	setCustomLabels(p)
	return p, nil
}

// setCustomLabels sets the docker compose custom labels on each service in the project.
// These labels are required for containers to be discoverable by the SDK after creation.
func setCustomLabels(project *types.Project) {
	for name, s := range project.Services {
		s.CustomLabels = types.Labels{
			api.ProjectLabel:     project.Name,
			api.ServiceLabel:     name,
			api.VersionLabel:     api.ComposeVersion,
			api.WorkingDirLabel:  project.WorkingDir,
			api.ConfigFilesLabel: strings.Join(project.ComposeFiles, ","),
			api.OneoffLabel:      "False",
		}
		project.Services[name] = s
	}
}

// newComposeService creates a compose service backed by the singleton Docker CLI.
func newComposeService(stdout, stderr io.Writer, opts ...compose.Option) (api.Compose, error) {
	dm, err := getDockerManagerInstance()
	if err != nil {
		return nil, err
	}
	baseOpts := []compose.Option{
		compose.WithOutputStream(stdout),
		compose.WithErrorStream(stderr),
	}
	return compose.NewComposeService(dm.cli, append(baseOpts, opts...)...)
}

// progressOpts returns compose.Option slice for progress display.
// mode uses display mode constants: ModeAuto, ModePlain, ModeQuiet, ModeTTY
func progressOpts(mode string) []compose.Option {
	if mode == "" {
		mode = display.ModeAuto
	}

	var ep api.EventProcessor
	switch mode {
	case display.ModeQuiet:
		// No event processor
		return nil
	case display.ModePlain:
		ep = display.Plain(output.UserErr.Writer())
	case display.ModeTTY:
		ep = display.Full(os.Stderr, os.Stderr, false)
	case display.ModeAuto:
		if !output.JSONOutput && isatty.IsTerminal(os.Stdout.Fd()) {
			ep = display.Full(os.Stderr, os.Stderr, false)
		} else {
			ep = display.Plain(output.UserErr.Writer())
		}
	}
	return []compose.Option{compose.WithEventProcessor(ep)}
}

// DownloadDockerBuildxIfNeeded downloads the proper version of docker-buildx
// if it's either not yet installed or doesn't meet the minimum version requirement.
// Returns downloaded bool (true if it did the download) and err.
// Pass force=true (tests only) to download when DDEV's binary is absent even if
// a system binary already satisfies the version requirement.
func DownloadDockerBuildxIfNeeded(force ...bool) (bool, error) {
	forceDownload := len(force) > 0 && force[0]
	dockerBuildxResetMsg := `To reset to DDEV's default built-in docker-buildx:
  ddev config global --required-docker-buildx-version="" --use-docker-buildx-from-system=false`

	requiredVersion := globalconfig.GetRequiredDockerBuildxVersion()
	err := CheckDockerBuildxVersion(DockerRequirements)
	if err != nil {
		err = fmt.Errorf("%w\n%s", err, dockerBuildxResetMsg)
	}
	// If no required version is set, then we don't need to download
	// but if there was an error checking the version, report it
	if requiredVersion == "" {
		return false, err
	}
	currentVersion, _ := GetDockerBuildxVersion()
	// If the current version meets the requirement, skip the download to avoid
	// unnecessary network traffic. forceDownload (used in tests only) overrides
	// this only when DDEV's own binary is absent at the destination — it does not
	// force a download when the binary already exists there.
	destinationPath, _ := globalconfig.GetDockerBuildxDestination()
	_, destStatErr := os.Stat(destinationPath)
	if currentVersion == requiredVersion && (!forceDownload || destStatErr == nil) {
		return false, err
	}
	// If we get here, we need to download the required version.
	// If that fails, report the error but also include any error
	// from the version check (e.g., if docker-buildx isn't found at all)
	downloadErr := DownloadDockerBuildx()
	if downloadErr == nil {
		// Reset the plugins to pick up the new binary
		_ = ResetCLIPlugins()
		if err = CheckDockerBuildxVersion(DockerRequirements); err != nil {
			return false, fmt.Errorf("%w\n%s", err, dockerBuildxResetMsg)
		}
		return true, nil
	}
	if err != nil {
		err = fmt.Errorf("%v\nUnable to download required docker-buildx version %q: %w", err, requiredVersion, downloadErr)
	} else {
		err = fmt.Errorf("unable to download required docker-buildx version %q: %w\n%s", requiredVersion, downloadErr, dockerBuildxResetMsg)
	}
	return false, err
}

// DownloadDockerBuildx gets the docker-buildx binary and puts it into
// ~/.ddev/bin
func DownloadDockerBuildx() error {
	globalBinDir := globalconfig.GetDDEVBinDir()
	destFile, _ := globalconfig.GetDockerBuildxDestination()

	buildxURL, shasumURL, err := dockerBuildxDownloadLink()
	if err != nil {
		return err
	}
	util.Debug("Downloading '%s' to '%s' ...", buildxURL, destFile)

	_ = os.Remove(destFile)

	_ = os.MkdirAll(globalBinDir, 0777)
	err = util.DownloadFile(destFile, buildxURL, globalconfig.IsInteractive(), shasumURL)
	if err != nil {
		_ = os.Remove(destFile)
		return err
	}
	output.UserErr.Printf("Download complete.")

	err = util.Chmod(destFile, 0755)
	if err != nil {
		return err
	}

	return nil
}

// dockerBuildxDownloadLink returns the URL and SHASUM-file link for docker-buildx
func dockerBuildxDownloadLink() (buildxURL string, shasumURL string, err error) {
	arch := runtime.GOARCH

	if arch != "arm64" && arch != "amd64" {
		return "", "", fmt.Errorf("only ARM64 and AMD64 architectures are supported for docker-buildx, not %s", arch)
	}
	flavor := runtime.GOOS + "-" + arch
	version := globalconfig.GetRequiredDockerBuildxVersion()
	if version == "" {
		version = versionconstants.RequiredDockerBuildxVersionDefault
	}
	binaryURL := fmt.Sprintf("https://github.com/docker/buildx/releases/download/v%[1]s/buildx-v%[1]s.%[2]s", version, flavor)
	if nodeps.IsWindows() {
		binaryURL = binaryURL + ".exe"
	}
	shasumURL = fmt.Sprintf("https://github.com/docker/buildx/releases/download/v%s/checksums.txt", version)

	return binaryURL, shasumURL, nil
}

// CreateComposeProject creates a compose project from a string
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

	composeYamlPull, err := CreateComposeProject("name: compose-yaml-pull")
	if err != nil {
		return err
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
		if _, exists := composeYamlPull.Services[service]; exists {
			continue
		}
		composeYamlPull.Services[service] = types.ServiceConfig{
			Image: image,
		}
		util.Debug(`Pulling image for %s ("%s" service)`, image, service)
	}

	if len(composeYamlPull.Services) == 0 {
		util.Debug("All images already exist locally, no pull needed")
		return nil
	}

	return ComposePull(ComposePullOpts{
		Project: composeYamlPull,
	})
}

// Pull pulls image if it doesn't exist locally
func Pull(image string) error {
	return PullImages([]string{image}, false)
}

// sanitizeServiceName sanitizes a string to be a valid Docker Compose service name
// by replacing any characters that don't match [a-zA-Z0-9._-] with hyphens
// See https://github.com/compose-spec/compose-go/blob/main/schema/compose-spec.json for allowed pattern
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
