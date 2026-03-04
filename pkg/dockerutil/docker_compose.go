package dockerutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/docker/compose/v5/cmd/display"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/docker/compose/v5/pkg/compose"
	"github.com/joho/godotenv"
	"github.com/mattn/go-isatty"
)

type ComposeCmdOpts struct {
	ComposeFiles []string
	ComposeYaml  *types.Project
	Profiles     []string
	Action       []string
	Progress     bool // Add dots every second while the compose command is running
	Timeout      time.Duration
	ProjectName  string // Optional project name to set via -p flag
	Env          []string
}

// parsedAction holds the result of parsing a ComposeCmdOpts.Action array.
type parsedAction struct {
	projectName string
	progress    string
	envFiles    []string
	subcommand  string
	subArgs     []string
}

// parseComposeAction extracts global flags and the subcommand from an action array.
// Global flags handled: -p <name>, --progress=<val>, --env-file <file>.
func parseComposeAction(action []string) parsedAction {
	var result parsedAction
	i := 0
	for i < len(action) {
		switch {
		case action[i] == "-p" && i+1 < len(action):
			result.projectName = action[i+1]
			i += 2
		case strings.HasPrefix(action[i], "--progress="):
			result.progress = strings.TrimPrefix(action[i], "--progress=")
			i++
		case action[i] == "--env-file" && i+1 < len(action):
			result.envFiles = append(result.envFiles, action[i+1])
			i += 2
		case strings.HasPrefix(action[i], "--env-file="):
			result.envFiles = append(result.envFiles, strings.TrimPrefix(action[i], "--env-file="))
			i++
		case !strings.HasPrefix(action[i], "-"):
			result.subcommand = action[i]
			result.subArgs = action[i+1:]
			return result
		default:
			i++ // skip unknown global flags
		}
	}
	return result
}

// setCustomLabels sets the docker compose custom labels on each service in the project.
// These labels are required for containers to be discoverable by the SDK after creation,
// since the SDK's ContainerList queries filter by the project label.
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

// loadProjectFromCmd loads (or returns) the compose project described by cmd.
// projectName is the effective project name (from -p flag or cmd.ProjectName).
// envFiles are additional .env files used for variable interpolation.
func loadProjectFromCmd(ctx context.Context, cmd *ComposeCmdOpts, projectName string, envFiles []string) (*types.Project, error) {
	if cmd.ComposeYaml != nil {
		project := cmd.ComposeYaml
		if projectName != "" {
			project.Name = projectName
		}
		setCustomLabels(project)
		return project, nil
	}

	// Build environment for interpolation: OS env + any --env-file contents
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

	configFiles := make([]types.ConfigFile, 0, len(cmd.ComposeFiles))
	for _, f := range cmd.ComposeFiles {
		configFiles = append(configFiles, types.ConfigFile{Filename: f})
	}

	// Set WorkingDir from the first compose file so relative paths resolve correctly
	workingDir := ""
	if len(cmd.ComposeFiles) > 0 {
		workingDir = filepath.Dir(cmd.ComposeFiles[0])
	}

	var loaderOpts []func(*loader.Options)
	if len(cmd.Profiles) > 0 {
		loaderOpts = append(loaderOpts, loader.WithProfiles(cmd.Profiles))
	}
	if projectName != "" {
		pn := projectName
		loaderOpts = append(loaderOpts, func(opts *loader.Options) {
			opts.SetProjectName(pn, true)
		})
	}

	project, err := loader.LoadWithContext(ctx, types.ConfigDetails{
		ConfigFiles: configFiles,
		Environment: environment,
		WorkingDir:  workingDir,
	}, loaderOpts...)
	if err != nil {
		return nil, err
	}
	setCustomLabels(project)
	return project, nil
}

// execSubArgs holds parsed options from a compose exec subcommand argument list.
type execSubArgs struct {
	service string
	command []string
	tty     bool
	detach  bool
	workDir string
	user    string
	env     []string
}

// parseExecSubArgs parses exec subcommand args into an execSubArgs struct.
// Handled flags: -T, -it/-ti, -d/--detach, -w/--workdir, -u/--user, -e/--env.
func parseExecSubArgs(args []string) execSubArgs {
	var result execSubArgs
	i := 0
	for i < len(args) {
		arg := args[i]
		switch {
		case arg == "-T":
			i++
		case arg == "-it" || arg == "-ti":
			result.tty = true
			i++
		case arg == "-d" || arg == "--detach":
			result.detach = true
			i++
		case (arg == "-w" || arg == "--workdir") && i+1 < len(args):
			result.workDir = args[i+1]
			i += 2
		case (arg == "-u" || arg == "--user") && i+1 < len(args):
			result.user = args[i+1]
			i += 2
		case (arg == "-e" || arg == "--env") && i+1 < len(args):
			result.env = append(result.env, args[i+1])
			i += 2
		case strings.HasPrefix(arg, "-"):
			i++ // skip unknown flags
		default:
			result.service = arg
			result.command = args[i+1:]
			return result
		}
	}
	return result
}

// getComposeService returns a compose.api.Compose instance backed by the existing dockerCli.
// When cmd is non-nil and cmd.Progress is true, attaches a display.EventProcessor using
// the same TTY detection logic as the compose CLI: Full for terminal stdout, Plain otherwise.
func getComposeService(cmd *ComposeCmdOpts, stdout, stderr io.Writer, extraOpts ...compose.Option) (api.Compose, error) {
	dm, err := getDockerManagerInstance()
	if err != nil {
		return nil, err
	}
	opts := []compose.Option{
		compose.WithOutputStream(stdout),
		compose.WithErrorStream(stderr),
	}
	if cmd != nil && cmd.Progress {
		var ep api.EventProcessor
		// Check actual stdout terminal, not dm.cli.Out() which is io.Discard
		if isatty.IsTerminal(os.Stdout.Fd()) {
			ep = display.Full(os.Stderr, os.Stderr, false)
		} else {
			ep = display.Plain(os.Stderr)
		}
		opts = append(opts, compose.WithEventProcessor(ep))
	}
	opts = append(opts, extraOpts...)
	return compose.NewComposeService(dm.cli, opts...)
}

// ComposeWithStreams executes a compose operation using the Docker Compose SDK,
// routing output to the provided stdin/stdout/stderr streams.
func ComposeWithStreams(cmd *ComposeCmdOpts, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	defer util.TimeTrack()()

	parsed := parseComposeAction(cmd.Action)
	projectName := cmd.ProjectName
	if parsed.projectName != "" {
		projectName = parsed.projectName
	}

	ctx := context.Background()

	var stdinOpt []compose.Option
	if stdin != nil {
		stdinOpt = []compose.Option{compose.WithInputStream(stdin)}
	}
	svc, err := getComposeService(cmd, stdout, stderr, stdinOpt...)
	if err != nil {
		return err
	}

	switch parsed.subcommand {
	case "up":
		project, err := loadProjectFromCmd(ctx, cmd, projectName, parsed.envFiles)
		if err != nil {
			return err
		}
		var createOpts api.CreateOptions
		if containsFlag(parsed.subArgs, "--build") {
			buildOpts := api.BuildOptions{Progress: parsed.progress}
			createOpts.Build = &buildOpts
		}
		return svc.Up(ctx, project, api.UpOptions{Create: createOpts, Start: api.StartOptions{Project: project}})

	case "pull":
		project, err := loadProjectFromCmd(ctx, cmd, projectName, parsed.envFiles)
		if err != nil {
			return err
		}
		return svc.Pull(ctx, project, api.PullOptions{})

	case "exec":
		ea := parseExecSubArgs(parsed.subArgs)
		project, err := loadProjectFromCmd(ctx, cmd, projectName, parsed.envFiles)
		if err != nil {
			return err
		}
		_, err = svc.Exec(ctx, project.Name, api.RunOptions{
			Service:     ea.service,
			Command:     ea.command,
			Tty:         ea.tty,
			Interactive: ea.tty,
			Detach:      ea.detach,
			WorkingDir:  ea.workDir,
			User:        ea.user,
			Environment: ea.env,
		})
		return err

	default:
		return fmt.Errorf("ComposeWithStreams: unsupported compose action %q in %v", parsed.subcommand, cmd.Action)
	}
}

// ComposeCmd executes a compose operation using the Docker Compose SDK.
// Returns stdout output, stderr output, and any error.
func ComposeCmd(cmd *ComposeCmdOpts) (string, string, error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	parsed := parseComposeAction(cmd.Action)
	projectName := cmd.ProjectName
	if parsed.projectName != "" {
		projectName = parsed.projectName
	}

	ctx := context.Background()
	if cmd.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cmd.Timeout)
		defer cancel()
	}

	switch parsed.subcommand {
	case "config":
		project, err := loadProjectFromCmd(ctx, cmd, projectName, parsed.envFiles)
		if err != nil {
			return "", "", err
		}
		if containsFlag(parsed.subArgs, "--services") {
			services := make([]string, 0, len(project.Services))
			for name := range project.Services {
				services = append(services, name)
			}
			sort.Strings(services)
			return strings.Join(services, "\n") + "\n", "", nil
		}
		yamlBytes, err := project.MarshalYAML()
		if err != nil {
			return "", "", err
		}
		return string(yamlBytes), "", nil

	case "up":
		project, err := loadProjectFromCmd(ctx, cmd, projectName, parsed.envFiles)
		if err != nil {
			return "", "", err
		}
		svc, err := getComposeService(cmd, &stdoutBuf, &stderrBuf)
		if err != nil {
			return "", "", err
		}
		var createOpts api.CreateOptions
		if containsFlag(parsed.subArgs, "--build") {
			buildOpts := api.BuildOptions{Progress: parsed.progress}
			createOpts.Build = &buildOpts
		}
		err = svc.Up(ctx, project, api.UpOptions{Create: createOpts, Start: api.StartOptions{Project: project}})
		if ctx.Err() != nil {
			return stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("ComposeCmd timed out after %v: %v", cmd.Timeout, err)
		}
		return stdoutBuf.String(), stderrBuf.String(), err

	case "down":
		project, err := loadProjectFromCmd(ctx, cmd, projectName, parsed.envFiles)
		if err != nil {
			return "", "", err
		}
		svc, err := getComposeService(cmd, &stdoutBuf, &stderrBuf)
		if err != nil {
			return "", "", err
		}
		err = svc.Down(ctx, project.Name, api.DownOptions{RemoveOrphans: true, Project: project})
		if ctx.Err() != nil {
			return stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("ComposeCmd timed out after %v: %v", cmd.Timeout, err)
		}
		return stdoutBuf.String(), stderrBuf.String(), err

	case "stop":
		project, err := loadProjectFromCmd(ctx, cmd, projectName, parsed.envFiles)
		if err != nil {
			return "", "", err
		}
		svc, err := getComposeService(cmd, &stdoutBuf, &stderrBuf)
		if err != nil {
			return "", "", err
		}
		err = svc.Stop(ctx, project.Name, api.StopOptions{Project: project})
		if ctx.Err() != nil {
			return stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("ComposeCmd timed out after %v: %v", cmd.Timeout, err)
		}
		return stdoutBuf.String(), stderrBuf.String(), err

	case "pull":
		project, err := loadProjectFromCmd(ctx, cmd, projectName, parsed.envFiles)
		if err != nil {
			return "", "", err
		}
		svc, err := getComposeService(cmd, &stdoutBuf, &stderrBuf)
		if err != nil {
			return "", "", err
		}
		err = svc.Pull(ctx, project, api.PullOptions{})
		if ctx.Err() != nil {
			return stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("ComposeCmd timed out after %v: %v", cmd.Timeout, err)
		}
		return stdoutBuf.String(), stderrBuf.String(), err

	case "build":
		project, err := loadProjectFromCmd(ctx, cmd, projectName, parsed.envFiles)
		if err != nil {
			return "", "", err
		}
		svc, err := getComposeService(cmd, &stdoutBuf, &stderrBuf)
		if err != nil {
			return "", "", err
		}
		err = svc.Build(ctx, project, api.BuildOptions{
			Progress: parsed.progress,
			NoCache:  containsFlag(parsed.subArgs, "--no-cache"),
		})
		if ctx.Err() != nil {
			return stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("ComposeCmd timed out after %v: %v", cmd.Timeout, err)
		}
		return stdoutBuf.String(), stderrBuf.String(), err

	case "exec":
		ea := parseExecSubArgs(parsed.subArgs)
		project, err := loadProjectFromCmd(ctx, cmd, projectName, parsed.envFiles)
		if err != nil {
			return "", "", err
		}
		svc, err := getComposeService(cmd, &stdoutBuf, &stderrBuf)
		if err != nil {
			return "", "", err
		}
		_, err = svc.Exec(ctx, project.Name, api.RunOptions{
			Service:     ea.service,
			Command:     ea.command,
			Tty:         ea.tty,
			Interactive: ea.tty,
			Detach:      ea.detach,
			WorkingDir:  ea.workDir,
			User:        ea.user,
			Environment: ea.env,
		})
		if ctx.Err() != nil {
			return stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("ComposeCmd timed out after %v: %v", cmd.Timeout, err)
		}
		return stdoutBuf.String(), stderrBuf.String(), err

	default:
		return "", "", fmt.Errorf("ComposeCmd: unsupported compose action %q in %v", parsed.subcommand, cmd.Action)
	}
}

// containsFlag reports whether flag appears in args.
func containsFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

// GetLiveDockerBuildxVersion runs `docker-compose --version` and caches result
func GetLiveDockerBuildxVersion() (string, error) {
	if globalconfig.DockerBuildxVersion != "" {
		return globalconfig.DockerBuildxVersion, nil
	}

	composePath, err := globalconfig.GetDockerBuildxPath()
	if err != nil {
		return "", err
	}

	if !fileutil.FileExists(composePath) {
		globalconfig.DockerBuildxVersion = ""
		return globalconfig.DockerBuildxVersion, fmt.Errorf("docker-buildx does not exist at %s", composePath)
	}
	out, err := exec.Command(composePath, "version").Output()
	if err != nil {
		return "", err
	}
	v := strings.Trim(string(out), "\r\n")

	// docker-compose v1 and v2.3.3 return a version without the prefix "v", so add it.
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}

	globalconfig.DockerBuildxVersion = v
	return globalconfig.DockerBuildxVersion, nil
}

// DownloadDockerBuildxIfNeeded downloads the proper version of docker-buildx
// if it's either not yet installed or doesn't meet the minimum version requirement.
// Returns downloaded bool (true if it did the download) and err
func DownloadDockerBuildxIfNeeded() (bool, error) {
	needDownload := false
	requiredVersion := globalconfig.GetRequiredDockerBuildxVersion()
	curVersion, err := GetBuildxVersion()
	if err != nil {
		needDownload = true
	} else if requiredVersion != "" && curVersion != strings.TrimPrefix(requiredVersion, "v") {
		needDownload = true
	}
	if err = CheckDockerBuildxVersion(DockerRequirements); err != nil {
		needDownload = true
	}
	if needDownload {
		err = DownloadDockerBuildx()
		if err == nil {
			_ = ResetCLIPlugins()
			return true, err
		}
	}
	return false, err
}

// DownloadDockerBuildx gets the docker-compose binary and puts it into
// ~/.ddev/bin
func DownloadDockerBuildx() error {
	globalBinDir := globalconfig.GetDDEVBinDir()
	destFile, _ := globalconfig.GetDockerBuildxPath()

	composeURL, shasumURL, err := dockerBuildxDownloadLink()
	if err != nil {
		return err
	}
	util.Debug("Downloading '%s' to '%s' ...", composeURL, destFile)

	_ = os.Remove(destFile)

	_ = os.MkdirAll(globalBinDir, 0777)
	err = util.DownloadFile(destFile, composeURL, globalconfig.IsInteractive(), shasumURL)
	if err != nil {
		_ = os.Remove(destFile)
		return err
	}
	output.UserErr.Printf("Download complete.")

	// Remove the cached DockerBuildxVersion
	globalconfig.DockerBuildxVersion = ""

	err = util.Chmod(destFile, 0755)
	if err != nil {
		return err
	}

	return nil
}

// dockerBuildxDownloadLink returns the URL and SHASUM-file link for docker-compose
func dockerBuildxDownloadLink() (composeURL string, shasumURL string, err error) {
	arch := runtime.GOARCH

	switch arch {
	case "arm64":
		arch = "arm64"
	case "amd64":
		arch = "amd64"
	default:
		return "", "", fmt.Errorf("only ARM64 and AMD64 architectures are supported for docker-buildx, not %s", arch)
	}
	flavor := runtime.GOOS + "-" + arch
	version := globalconfig.GetRequiredDockerBuildxVersion()
	if version == "" {
		version = versionconstants.RequiredDockerBuildxVersionDefault
	}
	composerURL := fmt.Sprintf("https://github.com/docker/buildx/releases/download/%[1]s/buildx-%[1]s.%[2]s", version, flavor)
	if nodeps.IsWindows() {
		composerURL = composerURL + ".exe"
	}
	shasumURL = fmt.Sprintf("https://github.com/docker/buildx/releases/download/%s/checksums.txt", version)

	return composerURL, shasumURL, nil
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

// PullImages pulls images in parallel if they don't exist locally
// If pullAlways is true, it will always pull
// Otherwise, it will only pull if the image doesn't exist
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

	if !output.JSONOutput && isatty.IsTerminal(os.Stdin.Fd()) {
		err = ComposeWithStreams(&ComposeCmdOpts{
			ComposeYaml: composeYamlPull,
			Action:      []string{"pull"},
			Progress:    true,
			Env:         []string{"COMPOSE_DISABLE_ENV_FILE=1"},
		}, nil, os.Stdout, os.Stderr)
	} else {
		_, _, err = ComposeCmd(&ComposeCmdOpts{
			ComposeYaml: composeYamlPull,
			Action:      []string{"pull"},
			Progress:    true,
			Env:         []string{"COMPOSE_DISABLE_ENV_FILE=1"},
		})
	}

	return err
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
