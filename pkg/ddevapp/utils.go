package ddevapp

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/semver/v3"
	"github.com/Masterminds/sprig/v3"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	dockerContainer "github.com/docker/docker/api/types/container"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// GetActiveProjects returns an array of DDEV projects
// that are currently live in docker.
func GetActiveProjects() []*DdevApp {
	apps := make([]*DdevApp, 0)
	labels := map[string]string{
		"com.ddev.platform":          "ddev",
		"com.docker.compose.service": "web",
		"com.docker.compose.oneoff":  "False",
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

// Cleanup will remove DDEV containers and volumes even if docker-compose.yml
// has been deleted.
func Cleanup(app *DdevApp) error {
	ctx, client := dockerutil.GetDockerClient()

	// Find all containers which match the current site name.
	labels := map[string]string{
		"com.ddev.site-name": app.GetName(),
	}

	// remove project network
	// "docker-compose down" - removes project network and any left-overs
	// There can be awkward cases where we're doing an app.Stop() but the rendered
	// yaml does not exist, all in testing situations.
	if fileutil.FileExists(app.DockerComposeFullRenderedYAMLPath()) {
		_, _, err := dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
			ComposeFiles: []string{app.DockerComposeFullRenderedYAMLPath()},
			Profiles:     []string{`*`},
			Action:       []string{"down"},
		})
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
		removeOpts := dockerContainer.RemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}
		output.UserOut.Printf("Removing container: %s", containerName)
		if err = client.ContainerRemove(ctx, containers[i].ID, removeOpts); err != nil {
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
	m["templateCanUse"] = templateCanUse

	return m
}

// templateCanUse will return true if the given feature is available.
// This is used in YAML templates to determine whether to use a feature or not.
func templateCanUse(feature string) bool {
	// healthcheck.start_interval requires Docker Engine v25 or later
	// See https://github.com/docker/compose/pull/10939
	if feature == "healthcheck.start_interval" {
		if err := dockerutil.CheckDockerVersion(dockerutil.DockerVersionMatrix{APIVersion: "1.44", Version: "25.0"}); err == nil {
			return true
		}
	}
	return false
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
	existingContent := ""

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
		// Read the existing content for future comparison.
		if gitIgnoreFileBytes, err := os.ReadFile(gitIgnoreFilePath); err == nil {
			existingContent = string(gitIgnoreFileBytes)
		}
		// Otherwise, remove the existing file to prevent surprising template results
		if existingContent == "" {
			err = os.Remove(gitIgnoreFilePath)
			if err != nil {
				return err
			}
		}
	}

	err := os.MkdirAll(targetDir, 0777)
	if err != nil {
		return err
	}

	// Get the content for the .gitignore file.
	var generatedIgnores []string
	for _, p := range ignores {
		pFullPath := filepath.Join(targetDir, p)
		sigFound, err := fileutil.FgrepStringInFile(pFullPath, nodeps.DdevFileSignature)
		if sigFound || err != nil {
			generatedIgnores = append(generatedIgnores, p)
		}
	}

	t, err := template.New("gitignore").Funcs(getTemplateFuncMap()).Parse(gitIgnoreTemplate)
	if err != nil {
		return err
	}

	// Execute the template into the buffer.
	var buf bytes.Buffer
	ignoredItems := ignoreTemplateContents{
		Signature:    nodeps.DdevFileSignature,
		IgnoredItems: generatedIgnores,
	}
	if err = t.Execute(&buf, ignoredItems); err != nil {
		return err
	}
	// Only write the file if the generated content differs from the existing content.
	if buf.String() != existingContent {
		file, err := os.OpenFile(gitIgnoreFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer util.CheckClose(file)

		// Write the new content to the file.
		if _, err = buf.WriteTo(file); err != nil {
			return err
		}
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
	errString = strings.ReplaceAll(errString, "Received unexpected error:", "")
	errString = strings.ReplaceAll(errString, "\n", "")
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
		return fmt.Errorf("ddev can no longer find your project files at %s. If you would like to continue using DDEV to manage this project please restore your files to that directory. If you would like to make DDEV forget this project, you may run 'ddev stop --unlist %s'", project.GetAppRoot(), project.GetName())
	}

	return nil
}

// GetProjects returns projects that are listed
// in globalconfig projectlist (or in Docker container labels, or both)
// if activeOnly is true, only show projects that aren't stopped
// (or broken, missing config, missing files)
func GetProjects(activeOnly bool) ([]*DdevApp, error) {
	apps := make(map[string]*DdevApp)
	projectList := globalconfig.GetGlobalProjectList()

	// First grab the GetActiveApps (Docker labels) version of the projects and make sure it's
	// included. Hopefully Docker label information and global config information will not
	// be out of sync very often.
	dockerActiveApps := GetActiveProjects()
	for _, app := range dockerActiveApps {
		apps[app.Name] = app
	}

	// Now get everything we can find in global project list
	for name, info := range projectList {
		// Skip apps already found running in Docker
		if _, ok := apps[name]; ok {
			continue
		}

		app, err := NewApp(info.AppRoot, true)
		if err != nil {
			if os.IsNotExist(err) {
				util.Warning("The project '%s' no longer exists in the filesystem, removing it from registry", info.AppRoot)
				err = globalconfig.RemoveProjectInfo(name)
				if err != nil {
					util.Warning("unable to RemoveProjectInfo(%s): %v", name, err)
				}
			} else {
				util.Warning("Something went wrong with %s: %v", info.AppRoot, err)
			}
			continue
		}

		// If the app we loaded was already found with a different name, complain
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

// GetProjectNamesFunc returns a function for autocompleting project names
// for command arguments.
// If status is "inactive" or "active", only names of inactive or active
// projects respectively are returned.
// If status is "all", all project names are returned.
// If numArgs is 0, completion will be provided for infinite arguments,
// otherwise it will only be provided for the numArgs number of arguments.
func GetProjectNamesFunc(status string, numArgs int) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		// Don't provide completions if the user keeps hitting space after
		// exhausting all of the valid arguments.
		if numArgs > 0 && len(args)+1 > numArgs {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// Get all of the projects we're interested in for this completion function.
		var apps []*DdevApp
		var err error
		if status == "inactive" {
			apps, err = GetInactiveProjects()
		} else if status == "active" {
			apps, err = GetProjects(true)
		} else if status == "all" {
			apps, err = GetProjects(false)
		} else {
			// This is an error state - but we just return nothing
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		// Return nothing if we have nothing, or return all of the project names.
		// Note that if there's nothing to return, we don't let cobra pick completions
		// from the files in the cwd.
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// Don't show arguments that are already written on the command line
		var projectNames []string
		for _, name := range ExtractProjectNames(apps) {
			if !slices.Contains(args, name) {
				projectNames = append(projectNames, name)
			}
		}
		return projectNames, cobra.ShellCompDirectiveNoFileComp
	}
}

// GetServiceNamesFunc returns a function for autocompleting service names for service flag.
// If existingOnly is true, only names of existing services will be returned.
func GetServiceNamesFunc(existingOnly bool) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		// the project name can be passed as an argument
		projectName := ""
		if len(args) > 0 {
			projectName = args[0]
		}
		app, err := GetActiveApp(projectName)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if app.ComposeYaml == nil || app.ComposeYaml.Services == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var services []string
		for service := range app.ComposeYaml.Services {
			if existingOnly {
				c, err := app.FindContainerByType(service)
				if err == nil && c != nil {
					services = append(services, service)
				}
			} else {
				services = append(services, service)
			}
		}
		return services, cobra.ShellCompDirectiveNoFileComp
	}
}

// GetRelativeDirectory returns the directory relative to project root
// Note that the relative dir is returned as unix-style forward-slashes
func (app *DdevApp) GetRelativeDirectory(path string) string {
	// Find the relative dir
	relativeWorkingDir := strings.TrimPrefix(path, app.AppRoot)
	// Convert to slash/linux/macos notation, should work everywhere
	relativeWorkingDir = filepath.ToSlash(relativeWorkingDir)
	// Remove any leading /
	relativeWorkingDir = strings.TrimLeft(relativeWorkingDir, "/")

	return relativeWorkingDir
}

// GetRelativeWorkingDirectory returns the relative working directory relative to project root
// Note that the relative dir is returned as unix-style forward-slashes
func (app *DdevApp) GetRelativeWorkingDirectory() string {
	pwd, _ := os.Getwd()
	return app.GetRelativeDirectory(pwd)
}

// HasCustomCert returns true if the project uses a custom certificate
func (app *DdevApp) HasCustomCert() bool {
	customCertsPath := app.GetConfigPath("custom_certs")
	certFileName := fmt.Sprintf("%s.crt", app.Name)
	return fileutil.FileExists(filepath.Join(customCertsPath, certFileName))
}

// CanUseHTTPOnly returns true if the project can be accessed via http only
func (app *DdevApp) CanUseHTTPOnly() bool {
	switch {
	// Gitpod and Codespaces have their own router with TLS termination
	case nodeps.IsGitpod() || nodeps.IsCodespaces():
		return false
	// If we have no router, then no https otherwise
	case IsRouterDisabled(app):
		return true
	// If a custom cert, we can do https, so false
	case app.HasCustomCert():
		return false
	// If no mkcert installed, no https
	case globalconfig.GetCAROOT() == "":
		return true
	}
	// Default case is OK to use https
	return false
}

// Turn a slice of *DdevApp into a map keyed by name
func AppSliceToMap(appList []*DdevApp) map[string]*DdevApp {
	nameMap := make(map[string]*DdevApp)
	for _, app := range appList {
		nameMap[app.Name] = app
	}
	return nameMap
}

// CheckDdevVersionConstraint validates if the given constraint matches the current DDEV version.
// If the version constraint includes pre-releases, it will normalize the constraint before checking.
// Returns an error if the version doesn't meet the constraint or if the constraint is invalid.
func CheckDdevVersionConstraint(constraint string, errorPrefix string, errorSuffix string) error {
	normalizedConstraint := constraint
	if strings.Contains(versionconstants.DdevVersion, "-") {
		// Pre-releases need '-0' added for validation
		normalizedConstraint = normalizeConstraint(constraint)
	}
	util.Debug("Comparing constraint '%s' against version '%s'", normalizedConstraint, versionconstants.DdevVersion)
	if errorPrefix == "" {
		errorPrefix = "error"
	}
	c, err := semver.NewConstraint(normalizedConstraint)
	if err != nil {
		return fmt.Errorf("%s: the '%s' constraint is not valid. See https://github.com/Masterminds/semver#checking-version-constraints for valid constraints format", errorPrefix, constraint).(invalidConstraint)
	}
	// Make sure we do this check with valid released versions
	v, err := semver.NewVersion(versionconstants.DdevVersion)
	if err == nil && !c.Check(v) {
		return fmt.Errorf("%s: your DDEV version '%s' doesn't meet the constraint '%s'. Please update to a DDEV version that meets this constraint %s", errorPrefix, versionconstants.DdevVersion, constraint, strings.TrimSpace(errorSuffix))
	}
	return nil
}

// normalizeConstraint adds '-0' to version expressions that don't contain a prerelease identifier
// See https://github.com/Masterminds/semver#working-with-prerelease-versions
func normalizeConstraint(constraint string) string {
	// remove all commas, so we can split by spaces
	constraintNoCommas := strings.ReplaceAll(constraint, ",", " ")
	// Split the constraint into tokens based on spaces
	tokens := strings.Fields(constraintNoCommas)
	for i, token := range tokens {
		last := token[len(token)-1]
		// If the token represents a version number (ends with a digit or is a wildcard)
		// and doesn't contain a suffix '-0', append '-0'
		if !strings.HasSuffix(token, "-0") && strings.Contains("0123456789xX*", string(last)) {
			tokens[i] = token + "-0"
		}
	}
	return strings.Join(tokens, " ")
}
