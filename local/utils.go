package local

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/drud/drud-go/utils"
	"github.com/fsouza/go-dockerclient"
)

// PrepLocalSiteDirs creates a site's directories for local dev in ~/.drud/client/site
func PrepLocalSiteDirs(base string) error {
	err := os.MkdirAll(base, os.FileMode(int(0774)))
	if err != nil {
		return err
	}

	dirs := []string{
		"src",
		"files",
		"data",
	}
	for _, d := range dirs {
		dirPath := path.Join(base, d)
		err := os.Mkdir(dirPath, os.FileMode(int(0774)))
		if err != nil {
			if !strings.Contains(err.Error(), "file exists") {
				return err
			}
		}
	}

	return nil
}

// WriteLocalAppYAML writes docker-compose.yaml to $HOME/.drud/app.Path()
func WriteLocalAppYAML(app App) error {
	homedir, err := utils.GetHomeDir()
	if err != nil {
		log.Fatalln(err)
	}

	basePath := path.Join(homedir, ".drud", app.RelPath())
	err = PrepLocalSiteDirs(basePath)
	if err != nil {
		log.Fatalln(err)
	}

	f, err := os.Create(path.Join(basePath, "docker-compose.yaml"))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	rendered, err := app.RenderComposeYAML()
	if err != nil {
		return err
	}
	f.WriteString(rendered)
	return nil
}

// CloneSource clones or pulls a repo
func CloneSource(app App) error {
	homedir, err := utils.GetHomeDir()
	if err != nil {
		log.Fatalln(err)
	}

	details, err := app.GetRepoDetails()
	if err != nil {
		return err
	}

	coneURL, err := details.GetCloneURL()
	if err != nil {
		return err
	}

	basePath := path.Join(homedir, ".drud", app.RelPath(), "src")

	out, err := utils.RunCommand("git", []string{
		"clone", "-b", details.Branch, coneURL, basePath,
	})
	if err != nil {
		if !strings.Contains(string(out), "already exists") {
			return fmt.Errorf("%s - %s", err.Error(), string(out))
		}

		log.Println("Local copy of site exists...updating")

		out, err = utils.RunCommand("git", []string{
			"-C", basePath,
			"pull", "origin", details.Branch,
		})
		if err != nil {
			return fmt.Errorf("%s - %s", err.Error(), string(out))
		}
	}

	if len(out) > 0 {
		fmt.Println(string(out))
	}

	return nil
}

func GetPod(app App) (int64, error) {
	client, _ := GetDockerClient()
	var publicPort int64

	containers, err := client.ListContainers(docker.ListContainersOptions{All: false})
	if err != nil {
		return publicPort, err
	}

	for _, ctr := range containers {
		if strings.Contains(ctr.Names[0][1:], app.ContainerName()+"-web") {
			for _, port := range ctr.Ports {
				if port.PublicPort != 0 {
					publicPort = port.PublicPort
					return publicPort, nil
				}
			}
		}
	}
	return publicPort, fmt.Errorf("%s container not ready", app.ContainerName())
}

// GetPodPort clones or pulls a repo
func GetPodPort(app App) (int64, error) {
	var publicPort int64

	err := utils.Do(func(attempt int) (bool, error) {
		var err error
		publicPort, err = GetPod(app)
		if err != nil {
			time.Sleep(2 * time.Second) // wait a couple seconds
		}
		return attempt < 70, err
	})
	if err != nil {
		return publicPort, err
	}

	return publicPort, nil
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
