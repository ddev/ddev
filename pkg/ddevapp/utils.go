package ddevapp

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/fsouza/go-dockerclient"
	"github.com/gosuri/uitable"

	"errors"

	"os"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	gohomedir "github.com/mitchellh/go-homedir"
)

// GetApps returns an array of ddev applications.
func GetApps() []*DdevApp {
	apps := make([]*DdevApp, 0)
	labels := map[string]string{
		"com.ddev.platform":          "ddev",
		"com.docker.compose.service": "web",
	}
	containers, err := dockerutil.FindContainersByLabels(labels)

	if err == nil {
		for _, siteContainer := range containers {
			app := &DdevApp{}
			approot, ok := siteContainer.Labels["com.ddev.approot"]
			if !ok {
				break
			}

			err = app.Init(approot)

			// Artificially populate sitename and apptype based on labels
			// if app.Init() failed.
			if err != nil {
				app.Name = siteContainer.Labels["com.ddev.site-name"]
				app.Type = siteContainer.Labels["com.ddev.app-type"]
				app.AppRoot = siteContainer.Labels["com.ddev.approot"]
			}
			apps = append(apps, app)
		}
	}

	return apps
}

// CreateAppTable will create a new app table for describe and list output
func CreateAppTable() *uitable.Table {
	table := uitable.New()
	table.MaxColWidth = 140
	table.Separator = "  "
	table.Wrap = true
	table.AddRow("NAME", "TYPE", "LOCATION", "URL(s)", "STATUS")
	return table
}

// RenderHomeRootedDir shortens a directory name to replace homedir with ~
func RenderHomeRootedDir(path string) string {
	userDir, err := gohomedir.Dir()
	util.CheckErr(err)
	result := strings.Replace(path, userDir, "~", 1)
	result = strings.Replace(result, "\\", "/", -1)
	return result
}

// RenderAppRow will add an application row to an existing table for describe and list output.
func RenderAppRow(table *uitable.Table, row map[string]interface{}) {
	status := fmt.Sprint(row["status"])

	switch {
	case strings.Contains(status, SiteStopped):
		status = color.YellowString(status)
	case strings.Contains(status, SiteNotFound):
		status = color.RedString(status)
	case strings.Contains(status, SiteDirMissing):
		status = color.RedString(status)
	case strings.Contains(status, SiteConfigMissing):
		status = color.RedString(status)
	default:
		status = color.CyanString(status)
	}

	urls := row["httpurl"].(string)
	if row["httpsurl"] != "" {
		urls = urls + "\n" + row["httpsurl"].(string)
	}
	table.AddRow(
		row["name"],
		row["type"],
		row["shortroot"],
		urls,
		status,
	)

}

// Cleanup will remove ddev containers and volumes even if docker-compose.yml
// has been deleted.
func Cleanup(app *DdevApp) error {
	client := dockerutil.GetDockerClient()

	// Find all containers which match the current site name.
	labels := map[string]string{
		"com.ddev.site-name": app.GetName(),
	}
	containers, err := dockerutil.FindContainersByLabels(labels)
	if err != nil {
		return err
	}

	// First, try stopping the listed containers if they are running.
	for i := range containers {
		containerName := containers[i].Names[0][1:len(containers[i].Names[0])]
		removeOpts := docker.RemoveContainerOptions{
			ID:            containers[i].ID,
			RemoveVolumes: true,
			Force:         true,
		}
		output.UserOut.Printf("Removing container: %s", containerName)
		if err = client.RemoveContainer(removeOpts); err != nil {
			return fmt.Errorf("could not remove container %s: %v", containerName, err)
		}
	}

	err = StopRouterIfNoContainers()
	return err
}

// CheckForConf checks for a config.yaml at the cwd or parent dirs.
func CheckForConf(confPath string) (string, error) {
	if fileutil.FileExists(confPath + "/.ddev/config.yaml") {
		return confPath, nil
	}
	pathList := strings.Split(confPath, "/")

	for range pathList {
		confPath = filepath.Dir(confPath)
		if fileutil.FileExists(confPath + "/.ddev/config.yaml") {
			return confPath, nil
		}
	}

	return "", errors.New("no .ddev/config.yaml file was found in this directory or any parent")
}

// ddevContainersRunning determines if any ddev-controlled containers are currently running.
func ddevContainersRunning() (bool, error) {
	containers, err := dockerutil.GetDockerContainers(false)
	if err != nil {
		return false, err
	}

	for _, container := range containers {
		if _, ok := container.Labels["com.ddev.platform"]; ok {
			return true, nil
		}
	}
	return false, nil
}

// getTemplateFuncMap will return a map of useful template functions.
func getTemplateFuncMap() map[string]interface{} {
	// Use sprig's template function map as a base
	m := sprig.FuncMap()

	// Add helpful utilities on top of it
	m["joinPath"] = path.Join

	return m
}

// gitIgnoreTemplate will write a .gitignore file.
// This template expects string slice to be provided, with each string corresponding to
// a line in the resulting .gitignore.
const gitIgnoreTemplate = `{{.Signature}}: Automatically generated ddev .gitignore.
{{range .IgnoredItems}}
/{{.}}{{end}}
`

type ignoreTemplateContents struct {
	Signature    string
	IgnoredItems []string
}

// CreateGitIgnore will create a .gitignore file in the target directory if one does not exist.
// Each value in ignores will be added as a new line to the .gitignore.
func CreateGitIgnore(targetDir string, ignores ...string) error {
	gitIgnoreFilePath := filepath.Join(targetDir, ".gitignore")

	if fileutil.FileExists(gitIgnoreFilePath) {
		sigFound, err := fileutil.FgrepStringInFile(gitIgnoreFilePath, DdevFileSignature)
		util.CheckErr(err)
		// If we sigFound the file and did not find the signature in .ddev/.gitignore, warn about it.
		if !sigFound {
			util.Warning("User-managed .ddev/.gitignore will not be managed by ddev")
			return nil
		}
		// Otherwise, remove the existing file to prevent surprising template results
		err = os.Remove(gitIgnoreFilePath)
		if err != nil {
			return err
		}
	}

	tmpl, err := template.New("gitignore").Funcs(getTemplateFuncMap()).Parse(gitIgnoreTemplate)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(gitIgnoreFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer util.CheckClose(file)

	parms := ignoreTemplateContents{
		Signature:    DdevFileSignature,
		IgnoredItems: ignores,
	}

	if err = tmpl.Execute(file, parms); err != nil {
		return err
	}

	return nil
}

// isTar determines whether the object at the filepath is a .tar archive.
func isTar(filepath string) bool {
	if strings.HasSuffix(filepath, ".tar") {
		return true
	}

	if strings.HasSuffix(filepath, ".tar.gz") {
		return true
	}

	if strings.HasSuffix(filepath, ".tgz") {
		return true
	}

	return false
}

// isZip determines if the object at hte filepath is a .zip.
func isZip(filepath string) bool {
	if strings.HasSuffix(filepath, ".zip") {
		return true
	}

	return false
}
