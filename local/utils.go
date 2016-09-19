package local

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/drud/drud-go/utils"
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

	basePath := path.Join(homedir, ".drud", app.Path())
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

	basePath := path.Join(homedir, ".drud", app.Path(), "src")

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
