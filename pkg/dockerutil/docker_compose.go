package dockerutil

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
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

// ComposeWithStreams executes a docker-compose command but allows the caller to specify
// stdin/stdout/stderr
func ComposeWithStreams(cmd *ComposeCmdOpts, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	defer util.TimeTrack()()

	var arg []string

	_, err := DownloadDockerComposeIfNeeded()
	if err != nil {
		return err
	}

	if cmd.ProjectName != "" {
		arg = append(arg, "-p", cmd.ProjectName)
	}

	if cmd.ComposeYaml != nil {
		// Read from stdin
		arg = append(arg, "-f", "-")
	} else {
		for _, file := range cmd.ComposeFiles {
			arg = append(arg, "-f", file)
		}
	}

	arg = append(arg, cmd.Action...)

	path, err := globalconfig.GetDockerComposePath()
	if err != nil {
		return err
	}
	proc := exec.Command(path, arg...)
	proc.Stdout = stdout
	proc.Stderr = stderr
	if cmd.ComposeYaml != nil {
		yamlBytes, err := cmd.ComposeYaml.MarshalYAML()
		if err != nil {
			return err
		}
		yamlBytes = util.EscapeDollarSign(yamlBytes)
		proc.Stdin = strings.NewReader(string(yamlBytes))
	} else {
		proc.Stdin = stdin
	}
	proc.Env = append(os.Environ(), cmd.Env...)

	err = proc.Run()
	return err
}

// ComposeCmd executes docker-compose commands via shell.
// returns stdout, stderr, error/nil
func ComposeCmd(cmd *ComposeCmdOpts) (string, string, error) {
	var arg []string
	var stdout bytes.Buffer
	var stderr string

	_, err := DownloadDockerComposeIfNeeded()
	if err != nil {
		return "", "", err
	}

	if cmd.ProjectName != "" {
		arg = append(arg, "-p", cmd.ProjectName)
	}

	if cmd.ComposeYaml != nil {
		// Read from stdin
		arg = append(arg, "-f", "-")
	} else {
		for _, file := range cmd.ComposeFiles {
			arg = append(arg, "-f", file)
		}
	}

	for _, profile := range cmd.Profiles {
		arg = append(arg, "--profile", profile)
	}

	arg = append(arg, cmd.Action...)

	path, err := globalconfig.GetDockerComposePath()
	if err != nil {
		return "", "", err
	}

	ctx := context.Background()
	if cmd.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cmd.Timeout)
		defer cancel()
	}
	proc := exec.CommandContext(ctx, path, arg...)
	proc.Stdout = &stdout
	if cmd.ComposeYaml != nil {
		yamlBytes, err := cmd.ComposeYaml.MarshalYAML()
		if err != nil {
			return "", "", err
		}
		yamlBytes = util.EscapeDollarSign(yamlBytes)
		proc.Stdin = strings.NewReader(string(yamlBytes))
	} else {
		proc.Stdin = os.Stdin
	}
	proc.Env = append(os.Environ(), cmd.Env...)

	stderrPipe, err := proc.StderrPipe()
	if err != nil {
		return "", "", fmt.Errorf("failed to proc.StderrPipe(): %v", err)
	}

	if err = proc.Start(); err != nil {
		return "", "", fmt.Errorf("failed to exec docker-compose: %v", err)
	}

	stderrOutput := bufio.NewScanner(stderrPipe)

	// Ignore chatty things from docker-compose like:
	// Container (or Volume) ... Creating or Created or Stopping or Starting or Removing
	// Container Stopped or Created
	// No resource found to remove (when doing a stop and no project exists)
	ignoreRegex := "(^ *(Network|Container|Volume|Service) .* (Creat|Start|Stopp|Remov|Build|Buil|Runn)(ing|t)$|.* Built$|^Container .*(Stopp|Creat)(ed|ing)$|Warning: No resource found to remove|Pulling fs layer|Waiting|Downloading|Extracting|Verifying Checksum|Download complete|Pull complete)"
	downRE, err := regexp.Compile(ignoreRegex)
	if err != nil {
		util.Warning("Failed to compile regex %v: %v", ignoreRegex, err)
	}

	var done chan bool
	if cmd.Progress {
		done = util.ShowDots()
	}
	for stderrOutput.Scan() {
		line := stderrOutput.Text()
		if len(stderr) > 0 {
			stderr = stderr + "\n"
		}
		stderr = stderr + line
		line = strings.Trim(line, "\n\r")
		switch {
		case downRE.MatchString(line):
			break
		default:
			output.UserOut.Println(line)
		}
	}

	err = proc.Wait()
	if cmd.Progress {
		done <- true
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return stdout.String(), stderr, fmt.Errorf("composeCmd timed out after %v and failed to run 'COMPOSE_PROJECT_NAME=%s docker-compose %v', action='%v', err='%v', stdout='%s', stderr='%s'", cmd.Timeout, os.Getenv("COMPOSE_PROJECT_NAME"), strings.Join(arg, " "), cmd.Action, err, stdout.String(), stderr)
	}
	if err != nil {
		return stdout.String(), stderr, fmt.Errorf("composeCmd failed to run 'COMPOSE_PROJECT_NAME=%s docker-compose %v', action='%v', err='%v', stdout='%s', stderr='%s'", os.Getenv("COMPOSE_PROJECT_NAME"), strings.Join(arg, " "), cmd.Action, err, stdout.String(), stderr)
	}
	return stdout.String(), stderr, nil
}

// GetDockerComposeVersion runs docker-compose -v to get the current version
func GetDockerComposeVersion() (string, error) {
	if globalconfig.DockerComposeVersion != "" {
		return globalconfig.DockerComposeVersion, nil
	}

	return GetLiveDockerComposeVersion()
}

// GetLiveDockerComposeVersion runs `docker-compose --version` and caches result
func GetLiveDockerComposeVersion() (string, error) {
	if globalconfig.DockerComposeVersion != "" {
		return globalconfig.DockerComposeVersion, nil
	}

	composePath, err := globalconfig.GetDockerComposePath()
	if err != nil {
		return "", err
	}

	if !fileutil.FileExists(composePath) {
		globalconfig.DockerComposeVersion = ""
		return globalconfig.DockerComposeVersion, fmt.Errorf("docker-compose does not exist at %s", composePath)
	}
	out, err := exec.Command(composePath, "version", "--short").Output()
	if err != nil {
		return "", err
	}
	v := strings.Trim(string(out), "\r\n")

	// docker-compose v1 and v2.3.3 return a version without the prefix "v", so add it.
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}

	globalconfig.DockerComposeVersion = v
	return globalconfig.DockerComposeVersion, nil
}

// DownloadDockerComposeIfNeeded downloads the proper version of docker-compose
// if it's either not yet installed or has the wrong version.
// Returns downloaded bool (true if it did the download) and err
func DownloadDockerComposeIfNeeded() (bool, error) {
	requiredVersion := globalconfig.GetRequiredDockerComposeVersion()
	var err error
	if requiredVersion == "" {
		util.Debug("globalconfig use_docker_compose_from_path is set, so not downloading")
		return false, nil
	}
	curVersion, err := GetLiveDockerComposeVersion()
	if err != nil || curVersion != requiredVersion {
		err = DownloadDockerCompose()
		if err == nil {
			return true, err
		}
	}
	return false, err
}

// DownloadDockerCompose gets the docker-compose binary and puts it into
// ~/.ddev/.bin
func DownloadDockerCompose() error {
	globalBinDir := globalconfig.GetDDEVBinDir()
	destFile, _ := globalconfig.GetDockerComposePath()

	composeURL, shasumURL, err := dockerComposeDownloadLink()
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
	output.UserOut.Printf("Download complete.")

	// Remove the cached DockerComposeVersion
	globalconfig.DockerComposeVersion = ""

	err = util.Chmod(destFile, 0755)
	if err != nil {
		return err
	}

	return nil
}

// dockerComposeDownloadLink returns the URL and SHASUM-file link for docker-compose
func dockerComposeDownloadLink() (composeURL string, shasumURL string, err error) {
	arch := runtime.GOARCH

	switch arch {
	case "arm64":
		arch = "aarch64"
	case "amd64":
		arch = "x86_64"
	default:
		return "", "", fmt.Errorf("only ARM64 and AMD64 architectures are supported for docker-compose, not %s", arch)
	}
	flavor := runtime.GOOS + "-" + arch
	composerURL := fmt.Sprintf("https://github.com/docker/compose/releases/download/%s/docker-compose-%s", globalconfig.GetRequiredDockerComposeVersion(), flavor)
	if nodeps.IsWindows() {
		composerURL = composerURL + ".exe"
	}
	shasumURL = fmt.Sprintf("https://github.com/docker/compose/releases/download/%s/checksums.txt", globalconfig.GetRequiredDockerComposeVersion())

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
			Env:         []string{"COMPOSE_DISABLE_ENV_FILE=1"},
		}, nil, os.Stdout, os.Stderr)
	} else {
		_, _, err = ComposeCmd(&ComposeCmdOpts{
			ComposeYaml: composeYamlPull,
			Action:      []string{"pull"},
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
