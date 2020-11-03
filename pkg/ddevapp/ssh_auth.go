package ddevapp

import (
	"bytes"
	"fmt"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/fatih/color"
	"github.com/fsouza/go-dockerclient"
	"html/template"
	"os"
	"path"
	"path/filepath"
)

// SSHAuthName is the "machine name" of the ddev-ssh-agent docker-compose service
const SSHAuthName = "ddev-ssh-agent"

// SSHAuthComposeYAMLPath returns the filepath to the base .ssh-auth-compose yaml file.
func SSHAuthComposeYAMLPath() string {
	globalDir := globalconfig.GetGlobalDdevDir()
	dest := path.Join(globalDir, ".ssh-auth-compose.yaml")
	return dest
}

// FullRenderedSSHAuthComposeYAMLPath returns the filepath to the rendered
//.ssh-auth-compose-full.yaml file.
func FullRenderedSSHAuthComposeYAMLPath() string {
	globalDir := globalconfig.GetGlobalDdevDir()
	dest := path.Join(globalDir, ".ssh-auth-compose-full.yaml")
	return dest
}

// EnsureSSHAgentContainer ensures the ssh-auth container is running.
func (app *DdevApp) EnsureSSHAgentContainer() error {
	sshContainer, err := findDdevSSHAuth()
	if err != nil {
		return err
	}
	// If we already have a running ssh container, there's nothing to do.
	if sshContainer != nil && (sshContainer.State == "running" || sshContainer.State == "starting") {
		return nil
	}

	dockerutil.EnsureDdevNetwork()

	path, err := app.CreateSSHAuthComposeFile()
	if err != nil {
		return err
	}

	app.DockerEnv()

	// run docker-compose up -d
	// This will force-recreate, discarding existing auth if there is a stopped container.
	_, _, err = dockerutil.ComposeCmd([]string{path}, "-p", SSHAuthName, "up", "--build", "--force-recreate", "-d")
	if err != nil {
		return fmt.Errorf("failed to start ddev-ssh-agent: %v", err)
	}

	// ensure we have a happy sshAuth
	label := map[string]string{"com.docker.compose.project": SSHAuthName}
	_, err = dockerutil.ContainerWait(containerWaitTimeout, label)
	if err != nil {
		return fmt.Errorf("ddev-ssh-agent failed to become ready; debug with 'docker logs ddev-ssh-agent'; error: %v", err)
	}

	util.Warning("ssh-agent container is running: If you want to add authentication to the ssh-agent container, run 'ddev auth ssh' to enable your keys.")
	return nil
}

// RemoveSSHAgentContainer brings down the ddev-ssh-agent if it's running.
func RemoveSSHAgentContainer() error {
	// Stop the container if it exists
	err := dockerutil.RemoveContainer(globalconfig.DdevSSHAgentContainer, 0)
	if err != nil {
		if _, ok := err.(*docker.NoSuchContainer); !ok {
			return err
		}
	}
	util.Warning("The ddev-ssh-agent container has been removed. When you start it again you will have to use 'ddev auth ssh' to provide key authentication again.")
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

	templ := template.New("compose template")
	templ, err := templ.Parse(DdevSSHAuthTemplate)
	if err != nil {
		return "", err
	}

	context := filepath.Join(globalconfig.GetGlobalDdevDir(), ".sshimageBuild")
	err = WriteBuildDockerfile(filepath.Join(context, "Dockerfile"), "", nil, "")
	if err != nil {
		return "", err
	}

	uid, gid, username := util.GetContainerUIDGid()

	app.DockerEnv()

	templateVars := map[string]interface{}{
		"ssh_auth_image":        version.SSHAuthImage,
		"ssh_auth_tag":          version.SSHAuthTag,
		"compose_version":       version.DockerComposeFileFormatVersion,
		"AutoRestartContainers": globalconfig.DdevGlobalConfig.AutoRestartContainers,
		"Username":              username,
		"UID":                   uid,
		"GID":                   gid,
		"BuildContext":          context,
	}
	err = templ.Execute(&doc, templateVars)
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
	fullContents, _, err := dockerutil.ComposeCmd(files, "config")
	if err != nil {
		return "", err
	}
	_, err = fullHandle.WriteString(fullContents)
	if err != nil {
		return "", err
	}
	return FullRenderedSSHAuthComposeYAMLPath(), nil
}

// findDdevSSHAuth usees FindContainerByLabels to get our sshAuth container and
// return it (or nil if it doesn't exist yet)
func findDdevSSHAuth() (*docker.APIContainers, error) {
	containerQuery := map[string]string{
		"com.docker.compose.project": SSHAuthName,
	}

	container, err := dockerutil.FindContainerByLabels(containerQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute findContainersByLabels, %v", err)
	}
	return container, nil
}

// RenderSSHAuthStatus returns a user-friendly string showing sshAuth-status
func RenderSSHAuthStatus() string {
	status := GetSSHAuthStatus()
	var renderedStatus string

	switch status {
	case SiteStopped:
		renderedStatus = color.RedString(status)
	case "healthy":
		renderedStatus = color.CyanString(status)
	case "exited":
		fallthrough
	default:
		renderedStatus = color.RedString(status)
	}
	return fmt.Sprintf("\nssh-auth status: %v", renderedStatus)
}

// GetSSHAuthStatus outputs sshAuth status and warning if not
// running or healthy, as applicable.
func GetSSHAuthStatus() string {
	label := map[string]string{"com.docker.compose.project": SSHAuthName}
	container, err := dockerutil.FindContainerByLabels(label)

	if err != nil {
		util.Error("Failed to execute FindContainerByLabels(%v): %v", label, err)
		return SiteStopped
	}
	if container == nil {
		return SiteStopped
	}
	health, _ := dockerutil.GetContainerHealth(container)
	return health

}
