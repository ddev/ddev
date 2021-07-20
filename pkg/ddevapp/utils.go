package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/fatih/color"
	"github.com/fsouza/go-dockerclient"
	"github.com/gosuri/uitable"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"os"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	gohomedir "github.com/mitchellh/go-homedir"
)

// GetActiveProjects returns an array of ddev projects
// that are currently live in docker.
func GetActiveProjects() []*DdevApp {
	apps := make([]*DdevApp, 0)
	labels := map[string]string{
		"com.ddev.platform":          "ddev",
		"com.docker.compose.service": "web",
	}
	containers, err := dockerutil.FindContainersByLabels(labels)

	if err == nil {
		for _, siteContainer := range containers {
			approot, ok := siteContainer.Labels["com.ddev.approot"]
			if !ok {
				break
			}

			app, err := NewApp(approot, true)

			// Artificially populate sitename and apptype based on labels
			// if NewApp() failed.
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
	table.AddRow("NAME", "TYPE", "LOCATION", "URL", "STATUS")
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
	case strings.Contains(status, SitePaused):
		status = color.YellowString(status)
	case strings.Contains(status, SiteStopped):
		status = color.RedString(status)
	case strings.Contains(status, SiteDirMissing):
		status = color.RedString(status)
	case strings.Contains(status, SiteConfigMissing):
		status = color.RedString(status)
	default:
		status = color.CyanString(status)
	}

	urls := ""
	if row["status"] == SiteRunning {
		if globalconfig.GetCAROOT() != "" {
			urls = row["httpsurl"].(string)
		} else {
			urls = row["httpurl"].(string)
		}
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
	// Always kill the nfs volume on ddev remove
	for _, volName := range []string{app.GetNFSMountVolName()} {
		_ = dockerutil.RemoveVolume(volName)
	}

	err = StopRouterIfNoContainers()
	return err
}

// CheckForConf checks for a config.yaml at the cwd or parent dirs.
func CheckForConf(confPath string) (string, error) {
	if fileutil.FileExists(filepath.Join(confPath, ".ddev", "config.yaml")) {
		return confPath, nil
	}

	// Keep going until we can't go any higher
	for filepath.Dir(confPath) != confPath {
		confPath = filepath.Dir(confPath)
		if fileutil.FileExists(filepath.Join(confPath, ".ddev", "config.yaml")) {
			return confPath, nil
		}
	}

	return "", fmt.Errorf("no %s file was found in this directory or any parent", filepath.Join(".ddev", "config.yaml"))
}

// ddevContainersRunning determines if any ddev-controlled containers are currently running.
func ddevContainersRunning() (bool, error) {
	labels := map[string]string{
		"com.ddev.platform": "ddev",
	}
	containers, err := dockerutil.FindContainersByLabels(labels)
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
# You can remove the above line if you want to edit and maintain this file yourself.
/.gitignore
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
		if err != nil {
			return err
		}

		// If we sigFound the file and did not find the signature in .ddev/.gitignore, warn about it.
		if !sigFound {
			util.Warning("User-managed %s will not be managed/overwritten by ddev", gitIgnoreFilePath)
			return nil
		}
		// Otherwise, remove the existing file to prevent surprising template results
		err = os.Remove(gitIgnoreFilePath)
		if err != nil {
			return err
		}
	}
	err := os.MkdirAll(targetDir, 0777)
	if err != nil {
		return err
	}

	generatedIgnores := []string{}
	for _, p := range ignores {
		sigFound, err := fileutil.FgrepStringInFile(p, DdevFileSignature)
		if sigFound || err != nil {
			generatedIgnores = append(generatedIgnores, p)
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
		IgnoredItems: generatedIgnores,
	}

	//nolint: revive
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

// GetErrLogsFromApp is used to do app.Logs on an app after an error has
// been received, especially on app.Start. This is really for testing only
func GetErrLogsFromApp(app *DdevApp, errorReceived error) (string, error) {
	var serviceName string
	if errorReceived == nil {
		return "no error detected", nil
	}
	errString := errorReceived.Error()
	errString = strings.Replace(errString, "Received unexpected error:", "", -1)
	errString = strings.Replace(errString, "\n", "", -1)
	errString = strings.Trim(errString, " \t\n\r")
	if strings.Contains(errString, "container failed") || strings.Contains(errString, "container did not become ready") || strings.Contains(errString, "failed to become ready") {
		splitError := strings.Split(errString, " ")
		if len(splitError) > 0 && nodeps.ArrayContainsString([]string{"web", "db", "ddev-router", "ddev-ssh-agent"}, splitError[0]) {
			serviceName = splitError[0]
			logs, err := app.CaptureLogs(serviceName, false, "")
			if err != nil {
				return "", err
			}
			return logs, nil
		}
	}
	return "", fmt.Errorf("no logs found for service %s (Inspected err=%v)", serviceName, errorReceived)
}

// CheckForMissingProjectFiles returns an error if the project's configuration or project root cannot be found
func CheckForMissingProjectFiles(project *DdevApp) error {
	if strings.Contains(project.SiteStatus(), SiteConfigMissing) || strings.Contains(project.SiteStatus(), SiteDirMissing) {
		return fmt.Errorf("ddev can no longer find your project files at %s. If you would like to continue using ddev to manage this project please restore your files to that directory. If you would like to make ddev forget this project, you may run 'ddev stop --unlist %s'", project.GetAppRoot(), project.GetName())
	}

	return nil
}

// GetProjects returns projects that are listed
// in globalconfig projectlist (or in docker container labels, or both)
// if activeOnly is true, only show projects that aren't stopped
// (or broken, missing config, missing files)
func GetProjects(activeOnly bool) ([]*DdevApp, error) {
	apps := make(map[string]*DdevApp)
	projectList := globalconfig.GetGlobalProjectList()

	// First grab the GetActiveApps (docker labels) version of the projects and make sure it's
	// included. Hopefully docker label information and global config information will not
	// be out of sync very often.
	dockerActiveApps := GetActiveProjects()
	for _, app := range dockerActiveApps {
		apps[app.Name] = app
	}

	// Now get everything we can find in global project list
	for name, info := range projectList {
		// Skip apps already found running in docker
		if _, ok := apps[name]; ok {
			continue
		}

		app, err := NewApp(info.AppRoot, true)
		if err != nil {
			util.Warning("unable to create project at project root '%s': %v", info.AppRoot, err)
			continue
		}

		// If the app we just loaded was already found with a different name, complain
		if _, ok := apps[app.Name]; ok {
			util.Warning(`Project '%s' was found in configured directory %s and it is already used by project '%s'. If you have changed the name of the project, please "ddev stop --unlist %s" `, app.Name, app.AppRoot, name, name)
			continue
		}

		if !activeOnly || (app.SiteStatus() != SiteStopped && app.SiteStatus() != SiteConfigMissing && app.SiteStatus() != SiteDirMissing) {
			apps[app.Name] = app
		}
	}

	appSlice := []*DdevApp{}
	for _, v := range apps {
		appSlice = append(appSlice, v)
	}
	sort.Slice(appSlice, func(i, j int) bool { return appSlice[i].Name < appSlice[j].Name })

	return appSlice, nil
}
