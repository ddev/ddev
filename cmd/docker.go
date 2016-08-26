package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

// GetDockerMachineIP returns the IP of a docker-machine.
func GetDockerMachineIP(machineName string) (string, error) {
	ipoutput, err := exec.Command("docker-machine", "ip", machineName).Output()

	if err != nil {
		fmt.Println("Unable to obtain default docker-machine ip.")
		return "", err
	}

	// TrimSpace needed to get rid of trailing linebreak on defaultip.
	return strings.TrimSpace(string(ipoutput)), nil
}

// GetDockerMachineEndpoint returns the docker endpoint for a docker-machine.
func GetDockerMachineEndpoint(machineName string) (string, error) {
	dockerip, err := GetDockerMachineIP(machineName)
	if err != nil {
		return "", err
	}
	return "tcp://" + dockerip + ":2376", nil
}

// GetDockerCertPaths returns the ca, cert, and key paths.
func GetDockerCertPaths() (ca string, cert string, key string) {

	certpath := os.Getenv("DOCKER_CERT_PATH")
	if certpath == "" {
		certpath = path.Join(homedir, ".docker", "machine", "certs")
	}
	ca = fmt.Sprintf("%s/ca.pem", certpath)
	cert = fmt.Sprintf("%s/cert.pem", certpath)
	key = fmt.Sprintf("%s/key.pem", certpath)
	return
}

// GetDockerClient returns a docker client for a docker-machine.
func GetDockerClient() (*docker.Client, error) {
	// Create a new docker client talking to the default docker-machine.
	client, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		log.Fatal(err)
	}
	return client, err
}

// PullDockerImage pulls a docker image.
func PullDockerImage(client *docker.Client, imageName string) error {
	auth, err := docker.NewAuthConfigurationsFromDockerCfg()
	if err != nil {
		return err
	}
	imgOps := docker.PullImageOptions{
		Repository: imageName,
	}

	err = client.PullImage(imgOps, auth.Configs["https://index.docker.io/v1/"])
	if err != nil {
		return err
	}
	return nil
}

// CreateStartDockerContainer creates and then starts a docker container.
func CreateStartDockerContainer(client *docker.Client, imageName string, containerBinds []string) (*docker.Container, error) {
	// @see https://github.com/fsouza/go-dockerclient/blob/master/container.go#L200
	config := docker.Config{
		Image: imageName,
	}
	// @see https://github.com/fsouza/go-dockerclient/blob/master/container.go#L474
	hostConfig := docker.HostConfig{
		Binds:           containerBinds,
		PublishAllPorts: true,
	}
	// @see https://github.com/fsouza/go-dockerclient/blob/master/container.go#L377
	containerOptions := docker.CreateContainerOptions{
		//Name:       "@todo",
		Config:     &config,
		HostConfig: &hostConfig,
	}

	// Create the container.
	container, err := client.CreateContainer(containerOptions)
	if err != nil {
		fmt.Printf("Failed to create docker container. Error: %s\n", err)
	}

	// Start the container.
	err = client.StartContainer(container.ID, &hostConfig)
	if err != nil {
		fmt.Printf("Failed to start docker container. Error:  %s\n", err)
	}
	return container, err
}

// CopyFromDockerContainer creates an archive from a directory within a container and then extracts it locally.
func CopyFromDockerContainer(client *docker.Client, container *docker.Container, sourcePath string, destinationPath string) error {
	var archive bytes.Buffer
	dlOpts := docker.DownloadFromContainerOptions{
		Path:         sourcePath,
		OutputStream: &archive,
	}
	if err := client.DownloadFromContainer(container.ID, dlOpts); err != nil {
		return err
	}

	fileTarPath := path.Join(os.TempDir(), "files.tar")

	if err := ioutil.WriteFile(fileTarPath, archive.Bytes(), 0644); err != nil {
		return err
	}

	// Ensure destination directory exists.
	if err := exec.Command("mkdir", "-p", destinationPath).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	fmt.Println("Extracting the files directory archive.")

	// From everything I've seen it's not easy to untar a file natively in go, so exec.Command it is.
	if err := exec.Command("tar", "-C", destinationPath, "-xzf", fileTarPath).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	// Remove the tar.
	if err := exec.Command("rm", fileTarPath).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	// Ensure destination has open permissions.
	// @todo Determine if there is a way perms can be set to something besides 777.
	if err := exec.Command("chmod", "-R", "777", destinationPath).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

// RemoveDockerContainer removes a docker container.
func RemoveDockerContainer(client *docker.Client, container *docker.Container) error {
	removeOpts := docker.RemoveContainerOptions{
		ID:    container.ID,
		Force: true,
	}
	if err := client.RemoveContainer(removeOpts); err != nil {
		return err
	}
	return nil
}

// ListDockerContainers lists docker containers.
func ListDockerContainers(client *docker.Client, filters map[string][]string) ([]docker.APIContainers, error) {
	listOptions := docker.ListContainersOptions{
		All:     false,
		Size:    true,
		Filters: filters,
	}

	return client.ListContainers(listOptions)
}

// GetDockerPublicPort returns the public port for a specified private port within the docker container.
func GetDockerPublicPort(container docker.APIContainers, privatePort int64) (int64, error) {
	publicPort := int64(0)
	for _, p := range container.Ports {
		if p.PrivatePort == privatePort {
			publicPort = p.PublicPort
		}
	}
	if publicPort == 0 {
		return 0, fmt.Errorf("Public Port %d not found in container %v", publicPort, container.ID)
	}
	return publicPort, nil
}
