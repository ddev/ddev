package ddevapp

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	ddevImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
	"github.com/docker/compose/v5/cmd/display"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/moby/moby/api/types/container"
)

// SSHAuthName is the "machine name" of the ddev-ssh-agent docker-compose service
const SSHAuthName = "ddev-ssh-agent"

// sshAgentUpstreamLabel records the upstream socket a ddev-ssh-agent container
// relays to, so a changed ssh_agent_upstream recreates the container.
const sshAgentUpstreamLabel = "com.ddev.ssh-agent-upstream"

// hostServicesSSHAuthSock is where Docker Desktop, OrbStack, and Colima
// (started with --ssh-agent) expose the macOS host's SSH agent to containers.
const hostServicesSSHAuthSock = "/run/host-services/ssh-auth.sock"

// SSHAgentUpstreamSocket returns the socket, as seen by the Docker host, that
// ddev-ssh-agent relays to when ssh_agent_upstream is set, or "" when
// ddev-ssh-agent runs its own agent.
func SSHAgentUpstreamSocket() (string, error) {
	upstream := globalconfig.DdevGlobalConfig.SSHAgentUpstream
	switch upstream {
	case "":
		return "", nil
	case "host":
		if dockerutil.IsDockerDesktop() || dockerutil.IsOrbStack() || dockerutil.IsColima() {
			return hostServicesSSHAuthSock, nil
		}
		// Other macOS and Windows providers run Docker in a VM that cannot
		// reach a host socket.
		if runtime.GOOS != "linux" {
			return "", fmt.Errorf("ssh_agent_upstream=host is supported on this OS only with Docker Desktop, OrbStack, or Colima")
		}
		sock := os.Getenv("SSH_AUTH_SOCK")
		if sock == "" {
			return "", fmt.Errorf("ssh_agent_upstream=host but SSH_AUTH_SOCK is not set; start or forward an SSH agent first")
		}
		return sock, nil
	default:
		sock, _ := util.ExpandHomedir(upstream)
		if !filepath.IsAbs(sock) {
			return "", fmt.Errorf("ssh_agent_upstream must be empty, 'host', or an absolute socket path, not '%s'", upstream)
		}
		return sock, nil
	}
}

// SSHAuthComposeYAMLPath returns the filepath to the base .ssh-auth-compose yaml file.
func SSHAuthComposeYAMLPath() string {
	globalDir := globalconfig.GetGlobalDdevDir()
	dest := path.Join(globalDir, ".ssh-auth-compose.yaml")
	return dest
}

// FullRenderedSSHAuthComposeYAMLPath returns the filepath to the rendered
// .ssh-auth-compose-full.yaml file.
func FullRenderedSSHAuthComposeYAMLPath() string {
	globalDir := globalconfig.GetGlobalDdevDir()
	dest := path.Join(globalDir, ".ssh-auth-compose-full.yaml")
	return dest
}

// EnsureSSHAgentContainer ensures the ssh-auth container is running.
func (app *DdevApp) EnsureSSHAgentContainer() error {
	RunUpgradeCheck()

	util.Debug("Ensuring ddev-ssh-agent container is running with the current image")
	upstream, err := SSHAgentUpstreamSocket()
	if err != nil {
		return err
	}
	sshContainer, err := findDdevSSHAuth()
	if err != nil {
		return err
	}
	// Nothing to do if the ssh container is running with the current image and upstream.
	// Use HasSuffix to handle registry prefixes (e.g. docker.io/) that Podman includes in image names.
	if sshContainer != nil &&
		strings.HasSuffix(sshContainer.Image, ddevImages.GetSSHAuthImage()) &&
		sshContainer.Labels[sshAgentUpstreamLabel] == upstream &&
		(sshContainer.State == "running" || sshContainer.State == "starting") {
		return nil
	}

	dockerutil.EnsureDdevNetwork()

	composeFile, err := app.CreateSSHAuthComposeFile()
	if err != nil {
		return err
	}

	_ = app.DockerEnv()

	downProject, loadErr := dockerutil.LoadComposeProject([]string{composeFile}, api.ProjectLoadOptions{
		ProjectName: SSHAuthName,
	})
	if loadErr != nil {
		util.Warning("failed to load compose project for %s: %v", composeFile, loadErr)
	} else {
		downCtx, downSvc, svcErr := dockerutil.NewComposeService()
		if svcErr != nil {
			util.Warning("failed to create compose service: %v", svcErr)
		} else if downErr := downSvc.Down(downCtx, downProject.Name, api.DownOptions{Project: downProject, RemoveOrphans: true}); downErr != nil {
			util.Warning("failed to docker-compose down on %s: %v", composeFile, downErr)
		}
	}

	err = dockerutil.Pull(ddevImages.GetSSHAuthImage())
	if err != nil {
		return err
	}

	// Now restart ddev-ssh-agent
	upProject, err := dockerutil.LoadComposeProject([]string{composeFile}, api.ProjectLoadOptions{
		ProjectName: SSHAuthName,
	})
	if err != nil {
		return fmt.Errorf("failed to start ddev-ssh-agent: %v", err)
	}
	upCtx, upSvc, err := dockerutil.NewComposeService()
	if err != nil {
		return fmt.Errorf("failed to start ddev-ssh-agent: %v", err)
	}
	progress := display.ModeQuiet
	if globalconfig.DdevVerbose {
		progress = display.ModePlain
	}
	err = upSvc.Up(upCtx, upProject, api.UpOptions{
		Create: api.CreateOptions{
			Build:         &api.BuildOptions{Progress: progress},
			RemoveOrphans: true,
		},
		Start: api.StartOptions{Project: upProject},
	})
	if err != nil {
		return fmt.Errorf("failed to start ddev-ssh-agent: %v", err)
	}

	// ensure we have a happy sshAuth
	label := map[string]string{
		"com.docker.compose.project": SSHAuthName,
		"com.docker.compose.oneoff":  "False",
	}
	sshWaitTimeout := 60
	util.Debug(`Waiting for ddev-ssh-agent to become ready, timeout=%v`, sshWaitTimeout)
	logOutput, err := dockerutil.ContainerWait(sshWaitTimeout, label)
	if err != nil {
		return fmt.Errorf("ddev-ssh-agent failed to become ready; log=%s, err=%v", logOutput, err)
	}

	if upstream != "" {
		util.Success("ssh-agent container is relaying to the SSH agent at %s", upstream)
	} else {
		util.Warning("ssh-agent container is running: If you want to add authentication to the ssh-agent container, run 'ddev auth ssh' to enable your keys.")
	}
	return nil
}

// RemoveSSHAgentContainer brings down the ddev-ssh-agent if it's running.
func RemoveSSHAgentContainer() error {
	// Stop the container if it exists
	err := dockerutil.RemoveContainer(globalconfig.DdevSSHAgentContainer)
	if err != nil {
		if ok := dockerutil.IsErrNotFound(err); !ok {
			return err
		}
	}
	if globalconfig.DdevGlobalConfig.SSHAgentUpstream == "" {
		util.Warning("The ddev-ssh-agent container has been removed. When you start it again you will have to use 'ddev auth ssh' to provide key authentication again.")
	}
	return nil
}

// CreateSSHAuthComposeFile creates the docker-compose file for the ddev-ssh-agent
func (app *DdevApp) CreateSSHAuthComposeFile() (string, error) {
	var doc bytes.Buffer
	f, ferr := os.Create(SSHAuthComposeYAMLPath())
	if ferr != nil {
		return "", ferr
	}
	defer util.CheckClose(f)

	uid, gid, _ := dockerutil.GetContainerUser()
	timezone, _ := util.GetLocalTimezone()

	_ = app.DockerEnv()

	upstream, err := SSHAgentUpstreamSocket()
	if err != nil {
		return "", err
	}
	templateVars := map[string]any{
		"SSHAuthImage":   ddevImages.GetSSHAuthImage(),
		"UID":            uid,
		"GID":            gid,
		"Timezone":       timezone,
		"UseKeepID":      dockerutil.UseKeepID(),
		"UpstreamLabel":  sshAgentUpstreamLabel,
		"UpstreamSocket": upstream,
	}
	if upstream != "" {
		// Mount the directory rather than the socket: a file bind mount pins the
		// inode and goes stale when the upstream agent recreates its socket.
		templateVars["UpstreamDir"] = filepath.Dir(upstream)
		templateVars["UpstreamName"] = filepath.Base(upstream)
	}
	t, err := template.New("ssh_auth_compose_template.yaml").Funcs(getTemplateFuncMap()).ParseFS(bundledAssets, "ssh_auth_compose_template.yaml")
	if err != nil {
		return "", err
	}
	err = t.Execute(&doc, templateVars)
	util.CheckErr(err)
	_, err = f.WriteString(doc.String())
	util.CheckErr(err)

	fullHandle, err := os.Create(FullRenderedSSHAuthComposeYAMLPath())
	if err != nil {
		return "", err
	}

	userFiles, err := filepath.Glob(filepath.Join(globalconfig.GetGlobalDdevDir(), "ssh-auth-compose.*.yaml"))
	if err != nil {
		return "", err
	}
	files := append([]string{SSHAuthComposeYAMLPath()}, userFiles...)
	project, err := dockerutil.LoadComposeProject(files, api.ProjectLoadOptions{
		ProjectName: SSHAuthName,
	})
	if err != nil {
		return "", err
	}
	if err = project.CheckContainerNameUnicity(); err != nil {
		return "", err
	}
	injectDdevLabels(project, nil)
	fullContentsBytes, err := project.MarshalYAML()
	if err != nil {
		return "", err
	}
	fullContentsBytes = util.EscapeDollarSign(fullContentsBytes)
	_, err = fullHandle.Write(fullContentsBytes)
	if err != nil {
		return "", err
	}
	return FullRenderedSSHAuthComposeYAMLPath(), nil
}

// findDdevSSHAuth uses FindContainerByLabels to get our sshAuth container and
// return it (or nil if it doesn't exist yet)
func findDdevSSHAuth() (*container.Summary, error) {
	containerQuery := map[string]string{
		"com.docker.compose.project": SSHAuthName,
		"com.docker.compose.oneoff":  "False",
	}

	c, err := dockerutil.FindContainerByLabels(containerQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute findContainersByLabels, %v", err)
	}
	return c, nil
}

// RenderSSHAuthStatus returns a user-friendly string showing sshAuth-status
func RenderSSHAuthStatus() string {
	status := GetSSHAuthStatus()
	var renderedStatus string

	switch status {
	case "healthy":
		renderedStatus = util.ColorizeText(status, "green")
	case "exited":
		fallthrough
	default:
		renderedStatus = util.ColorizeText(status, "red")
	}
	return fmt.Sprintf("\nssh-auth status: %v", renderedStatus)
}

// GetSSHAuthStatus outputs sshAuth status and warning if not
// running or healthy, as applicable.
func GetSSHAuthStatus() string {
	label := map[string]string{
		"com.docker.compose.project": SSHAuthName,
		"com.docker.compose.oneoff":  "False",
	}
	c, err := dockerutil.FindContainerByLabels(label)

	if err != nil {
		util.Error("Failed to execute FindContainerByLabels(%v): %v", label, err)
		return SiteStopped
	}
	if c == nil {
		return SiteStopped
	}
	health, _ := dockerutil.GetContainerHealth(c)
	return health
}
