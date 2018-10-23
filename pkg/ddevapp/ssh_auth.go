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

// SshAuthName is the "machine name" of the ddev-ssh-agent docker-compose service
const SshAuthName = "ddev-ssh-agent"

// SshAuthComposeYAMLPath returns the full filepath to the ssh-auth docker-compose yaml file.
func SshAuthComposeYAMLPath() string {
	ddevDir := util.GetGlobalDdevDir()
	dest := path.Join(ddevDir, "ssh-auth-compose.yaml")
	return dest
}

// EnsureSshAuthContainer ensures the ssh-auth container is running.
func EnsureSshAuthContainer() error {
	sshContainer, err := findDdevSshAuth()
	if err != nil {
		return err
	}
	// If we already have an ssh container, there's nothing to do.
	if sshContainer != nil {
		return nil
	}
	sshAuthComposePath := SshAuthComposeYAMLPath()

	var doc bytes.Buffer
	f, ferr := os.Create(sshAuthComposePath)
	if ferr != nil {
		return ferr
	}
	defer util.CheckClose(f)

	templ := template.New("compose template")
	templ, err = templ.Parse(DdevSshAuthTemplate)
	if err != nil {
		return err
	}

	templateVars := map[string]interface{}{
		"ssh_auth_image":  version.SshAuthImage,
		"ssh_auth_tag":    version.SshAuthTag,
		"compose_version": version.DockerComposeFileFormatVersion,
	}

	err = templ.Execute(&doc, templateVars)
	util.CheckErr(err)
	_, err = f.WriteString(doc.String())
	util.CheckErr(err)

	// run docker-compose up -d in the newly created directory
	_, _, err = dockerutil.ComposeCmd([]string{sshAuthComposePath}, "-p", SshAuthName, "up", "-d")
	if err != nil {
		return fmt.Errorf("failed to start ddev-ssh-agent: %v", err)
	}

	// ensure we have a happy sshAuth
	// (ddev-ssh-agent doesn't currently have healthcheck so this doesn't work)
	//label := map[string]string{"com.docker.compose.project": SshAuthName}
	//err = dockerutil.ContainerWait(containerWaitTimeout, label)
	//if err != nil {
	//	return fmt.Errorf("ddev-ssh-agent failed to become ready: %v", err)
	//}

	// TODO: Update this warning so people know what to do.
	util.Warning("ssh-agent container is running: If you want to add authentication to to the ssh-agent container, do something about that.")
	return nil
}

// findDdevSshAuth usees FindContainerByLabels to get our sshAuth container and
// return it (or nil if it doesn't exist yet)
func findDdevSshAuth() (*docker.APIContainers, error) {
	containerQuery := map[string]string{
		"com.docker.compose.project": SshAuthName,
	}

	container, err := dockerutil.FindContainerByLabels(containerQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute findContainersByLabels, %v", err)
	}
	return container, nil
}

// RenderSshAuthStatus returns a user-friendly string showing sshAuth-status
func RenderSshAuthStatus() string {
	status := GetSshAuthStatus()
	var renderedStatus string
	badSshAuth := "\nThe sshAuth is not currently running. Your sites are likely inaccessible at this time.\nTry running 'ddev start' on a site to recreate the sshAuth."

	switch status {
	case SiteNotFound:
		renderedStatus = color.RedString(status) + badSshAuth
	case "healthy":
		renderedStatus = color.CyanString(status)
	case "exited":
		fallthrough
	default:
		renderedStatus = color.RedString(status) + badSshAuth
	}
	return fmt.Sprintf("\nDDEV ROUTER STATUS: %v", renderedStatus)
}

// GetSshAuthStatus outputs sshAuth status and warning if not
// running or healthy, as applicable.
func GetSshAuthStatus() string {
	var status string

	label := map[string]string{"com.docker.compose.service": "ddev-ssh-auth"}
	container, err := dockerutil.FindContainerByLabels(label)

	if err != nil {
		status = SiteNotFound
	} else {
		status = dockerutil.GetContainerHealth(*container)
	}

	return status
}
