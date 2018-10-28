package ddevapp

import (
	"bytes"
	"fmt"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/fatih/color"
	"github.com/fsouza/go-dockerclient"
	"html/template"
	"os"
	"path"
)

// SSHAuthName is the "machine name" of the ddev-ssh-agent docker-compose service
const SSHAuthName = "ddev-ssh-agent"

// SSHAuthComposeYAMLPath returns the full filepath to the ssh-auth docker-compose yaml file.
func SSHAuthComposeYAMLPath() string {
	ddevDir := util.GetGlobalDdevDir()
	dest := path.Join(ddevDir, "ssh-auth-compose.yaml")
	return dest
}

// EnsureSSHAgentContainer ensures the ssh-auth container is running.
func EnsureSSHAgentContainer() error {
	sshContainer, err := findDdevSSHAuth()
	if err != nil {
		return err
	}
	// If we already have a running ssh container, there's nothing to do.
	if sshContainer != nil && sshContainer.State == "running" {
		return nil
	}

	CreateSSHAuthComposeFile()

	sshAuthComposePath := SSHAuthComposeYAMLPath()

	// run docker-compose up -d
	// This will force-recreate, discarding existing auth if there is a stopped container.
	_, _, err = dockerutil.ComposeCmd([]string{sshAuthComposePath}, "-p", SSHAuthName, "up", "--force-recreate", "-d")
	if err != nil {
		return fmt.Errorf("failed to start ddev-ssh-agent: %v", err)
	}

	// ensure we have a happy sshAuth
	label := map[string]string{"com.docker.compose.project": SSHAuthName}
	err = dockerutil.ContainerWait(containerWaitTimeout, label)
	if err != nil {
		return fmt.Errorf("ddev-ssh-agent failed to become ready: %v", err)
	}

	util.Warning("ssh-agent container is running: If you want to add authentication to to the ssh-agent container, run 'ddev auth ssh' to enable your keys.")
	return nil
}

// StopSSHAgentContainer brings down the ddev-ssh-agent if it's running.
func StopSSHAgentContainer() error {
	sshContainer, err := findDdevSSHAuth()
	if err != nil {
		return err
	}
	// If we don't have a container, there's nothing to do.
	if sshContainer == nil {
		return nil
	}

	// Otherwise we'll "rm"
	CreateSSHAuthComposeFile()

	sshAuthComposePath := SSHAuthComposeYAMLPath()

	// run docker-compose rm -f
	_, _, err = dockerutil.ComposeCmd([]string{sshAuthComposePath}, "-p", SSHAuthName, "down")
	if err != nil {
		return fmt.Errorf("failed to rm ddev-ssh-agent: %v", err)
	}

	util.Warning("The ddev-ssh-agent container has been removed. When you start it again you will have to use 'ddev auth ssh' to provide key authentication again.")
	return nil
}

// CreateSSHAuthComposeFile creates the docker-compose file for the ddev-ssh-agent
func CreateSSHAuthComposeFile() error {

	sshAuthComposePath := SSHAuthComposeYAMLPath()

	var doc bytes.Buffer
	f, ferr := os.Create(sshAuthComposePath)
	if ferr != nil {
		return ferr
	}
	defer util.CheckClose(f)

	templ := template.New("compose template")
	templ, err := templ.Parse(DdevSSHAuthTemplate)
	if err != nil {
		return err
	}

	templateVars := map[string]interface{}{
		"ssh_auth_image":  version.SSHAuthImage,
		"ssh_auth_tag":    version.SSHAuthTag,
		"compose_version": version.DockerComposeFileFormatVersion,
	}

	err = templ.Execute(&doc, templateVars)
	util.CheckErr(err)
	_, err = f.WriteString(doc.String())
	util.CheckErr(err)
	return nil
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
	case SiteNotFound:
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
		return SiteNotFound
	}
	if container == nil {
		return SiteNotFound
	}
	return dockerutil.GetContainerHealth(*container)

}
