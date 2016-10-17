package local

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/drud/drud-go/utils"
	"github.com/drud/drud-go/utils/try"
	"github.com/fsouza/go-dockerclient"
	"github.com/gosuri/uitable"
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

		fmt.Print("Local copy of site exists, updating... ")

		out, err = utils.RunCommand("git", []string{
			"-C", basePath,
			"pull", "origin", details.Branch,
		})
		if err != nil {
			return fmt.Errorf("%s - %s", err.Error(), string(out))
		}

		fmt.Printf("Updated to latest in %s branch\n", details.Branch)
	}

	if len(out) > 0 {
		log.Info(string(out))
	}

	return nil
}

func GetPort(name string) (int64, error) {
	client, _ := GetDockerClient()
	var publicPort int64

	containers, err := client.ListContainers(docker.ListContainersOptions{All: false})
	if err != nil {
		return publicPort, err
	}

	for _, ctr := range containers {
		if strings.Contains(ctr.Names[0][1:], name) {
			for _, port := range ctr.Ports {
				if port.PublicPort != 0 {
					publicPort = port.PublicPort
					return publicPort, nil
				}
			}
		}
	}
	return publicPort, fmt.Errorf("%s container not ready", name)
}

// GetPodPort clones or pulls a repo
func GetPodPort(name string) (int64, error) {
	var publicPort int64

	err := try.Do(func(attempt int) (bool, error) {
		var err error
		publicPort, err = GetPort(name)
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

func FilterNonDrud(vs []docker.APIContainers) []docker.APIContainers {
	homedir, err := utils.GetHomeDir()
	if err != nil {
		log.Fatalln(err)
	}

	var vsf []docker.APIContainers
	for _, v := range vs {
		clientName := strings.Split(v.Names[0][1:], "-")[0]
		if _, err = os.Stat(path.Join(homedir, ".drud", clientName)); os.IsNotExist(err) {
			continue
		}
		vsf = append(vsf, v)
	}
	return vsf
}

func FilterNonLegacy(vs []docker.APIContainers) []docker.APIContainers {

	var vsf []docker.APIContainers
	for _, v := range vs {
		container := v.Names[0][1:]

		if !strings.HasPrefix(container, "legacy-") {
			continue
		}

		vsf = append(vsf, v)
	}
	return vsf
}

// FormatPlural is a simple wrapper which returns different strings based on the count value.
func FormatPlural(count int, single string, plural string) string {
	if count == 1 {
		return single
	}
	return plural
}

func RenderContainerList(containers []docker.APIContainers) error {
	fmt.Printf("%v %v found.\n", len(containers), FormatPlural(len(containers), "container", "containers"))

	table := uitable.New()
	table.MaxColWidth = 200
	table.AddRow("NAME", "IMAGE", "STATUS", "MISC")

	for _, container := range containers {

		var miscField string
		for _, port := range container.Ports {
			if port.PublicPort != 0 {
				miscField = fmt.Sprintf("port: %d", port.PublicPort)
			}
		}

		table.AddRow(
			strings.Join(container.Names, ", "),
			container.Image,
			fmt.Sprintf("%s - %s", container.State, container.Status),
			miscField,
		)
	}
	fmt.Println(table)

	return nil
}

// DetermineAppType uses some predetermined file checks to determine if a local app
// is of any of the known types
func DetermineAppType(basePath string) (string, error) {
	defaultLocations := map[string]string{
		"docroot/scripts/drupal.sh": "drupal",
		"docroot/wp":                "wp",
	}

	for k, v := range defaultLocations {
		if FileExists(path.Join(basePath, "src", k)) {
			return v, nil
		}
	}

	return "", fmt.Errorf("Couldn't determine app's type!")
}

// FileExists checks a file's existence
// @todo replace this with drud-go/utils version when merged
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
