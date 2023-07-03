package ddevapp

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/jedib0t/go-pretty/v6/table"
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

// RenderHomeRootedDir shortens a directory name to replace homedir with ~
func RenderHomeRootedDir(path string) string {
	userDir, err := os.UserHomeDir()
	util.CheckErr(err)
	result := strings.Replace(path, userDir, "~", 1)
	result = strings.Replace(result, "\\", "/", -1)
	return result
}

// RenderAppRow will add an application row to an existing table for describe and list output.
func RenderAppRow(t table.Writer, row map[string]interface{}) {
	status := fmt.Sprint(row["status_desc"])
	urls := ""
	mutagenStatus := ""
	if row["status"] == SiteRunning {
		urls = row["primary_url"].(string)
		if row["mutagen_enabled"] == true {
			if _, ok := row["mutagen_status"]; ok {
				mutagenStatus = row["mutagen_status"].(string)
			} else {
				mutagenStatus = "not enabled"
			}
			if mutagenStatus != "ok" {
				mutagenStatus = util.ColorizeText(mutagenStatus, "red")
			}
			status = fmt.Sprintf("%s (%s)", status, mutagenStatus)
		}
	}

	status = FormatSiteStatus(status)

	t.AppendRow(table.Row{
		row["name"], status, row["shortroot"], urls, row["type"],
	})

}

// Cleanup will remove ddev containers and volumes even if docker-compose.yml
// has been deleted.
func Cleanup(app *DdevApp) error {
	client := dockerutil.GetDockerClient()

	// Find all containers which match the current site name.
	labels := map[string]string{
		"com.ddev.site-name": app.GetName(),
	}

	// remove project network
	// "docker-compose down" - removes project network and any left-overs
	// There can be awkward cases where we're doing an app.Stop() but the rendered
	// yaml does not exist, all in testing situations.
	if fileutil.FileExists(app.DockerComposeFullRenderedYAMLPath()) {
		_, _, err := dockerutil.ComposeCmd([]string{app.DockerComposeFullRenderedYAMLPath()}, "down")
		if err != nil {
			util.Warning("Failed to docker-compose down: %v", err)
		}
	}

	// If any leftovers or lost souls, find them as well
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
	// Always kill the temporary volumes on ddev remove
	vols := []string{app.GetNFSMountVolumeName(), "ddev-" + app.Name + "-snapshots", app.Name + "-ddev-config"}

	for _, volName := range vols {
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
		sigFound, err := fileutil.FgrepStringInFile(gitIgnoreFilePath, nodeps.DdevFileSignature)
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
		sigFound, err := fileutil.FgrepStringInFile(p, nodeps.DdevFileSignature)
		if sigFound || err != nil {
			generatedIgnores = append(generatedIgnores, p)
		}
	}

	tmpl, err := template.New("gitignore").Funcs(getTemplateFuncMap()).Parse(gitIgnoreTemplate)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(gitIgnoreFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer util.CheckClose(file)

	parms := ignoreTemplateContents{
		Signature:    nodeps.DdevFileSignature,
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
	tarSuffixes := []string{".tar", ".tar.gz", ".tar.bz2", ".tar.xz", ".tgz"}
	for _, suffix := range tarSuffixes {
		if strings.HasSuffix(filepath, suffix) {
			return true
		}
	}

	return false
}

// isZip determines if the object at hte filepath is a .zip.
func isZip(filepath string) bool {
	return strings.HasSuffix(filepath, ".zip")
}

// GetErrLogsFromApp is used to do app.Logs on an app after an error has
// been received, especially on app.Start. This is really for testing only
// returns logs, healthcheck history, error
func GetErrLogsFromApp(app *DdevApp, errorReceived error) (string, string, error) {
	var serviceName string
	if errorReceived == nil {
		return "no error detected", "", nil
	}
	errString := errorReceived.Error()
	errString = strings.Replace(errString, "Received unexpected error:", "", -1)
	errString = strings.Replace(errString, "\n", "", -1)
	errString = strings.Trim(errString, " \t\n\r")
	if strings.Contains(errString, "container failed") || strings.Contains(errString, "container did not become ready") || strings.Contains(errString, "failed to become ready") {
		splitError := strings.Split(errString, " ")
		if len(splitError) > 0 && nodeps.ArrayContainsString([]string{"web", "db", "ddev-router", "ddev-ssh-agent"}, splitError[0]) {
			serviceName = splitError[0]
			health := ""
			if containerID, err := dockerutil.FindContainerByName(serviceName); err == nil {
				_, health = dockerutil.GetContainerHealth(containerID)
			}
			logs, err := app.CaptureLogs(serviceName, false, "10")
			if err != nil {
				return "", "", err
			}
			return logs, health, nil
		}
	}
	return "", "", fmt.Errorf("no logs found for service %s (Inspected err=%v)", serviceName, errorReceived)
}

// CheckForMissingProjectFiles returns an error if the project's configuration or project root cannot be found
func CheckForMissingProjectFiles(project *DdevApp) error {
	status, _ := project.SiteStatus()
	if status == SiteConfigMissing || status == SiteDirMissing {
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

		status, _ := app.SiteStatus()
		if !activeOnly || (status != SiteStopped && status != SiteConfigMissing && status != SiteDirMissing) {
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

// GetInactiveProjects returns projects that are currently running
func GetInactiveProjects() ([]*DdevApp, error) {
	var inactiveApps []*DdevApp

	apps, err := GetProjects(false)

	if err != nil {
		return nil, err
	}

	for _, app := range apps {
		status, _ := app.SiteStatus()
		if status != SiteRunning {
			inactiveApps = append(inactiveApps, app)
		}
	}

	return inactiveApps, nil
}

// ExtractProjectNames returns a list of names by a bunch of projects
func ExtractProjectNames(apps []*DdevApp) []string {
	var names []string
	for _, app := range apps {
		names = append(names, app.Name)
	}

	return names
}

// GetRelativeWorkingDirectory returns the relative working directory relative to project root
// Note that the relative dir is returned as unix-style forward-slashes
func (app *DdevApp) GetRelativeWorkingDirectory() string {
	pwd, _ := os.Getwd()

	// Find the relative dir
	relativeWorkingDir := strings.TrimPrefix(pwd, app.AppRoot)
	// Convert to slash/linux/macos notation, should work everywhere
	relativeWorkingDir = filepath.ToSlash(relativeWorkingDir)
	// remove any leading /
	relativeWorkingDir = strings.TrimLeft(relativeWorkingDir, "/")

	return relativeWorkingDir
}
