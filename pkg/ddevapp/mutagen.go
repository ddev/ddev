package ddevapp

import (
	"bufio"
	"embed"
	"fmt"
	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/drud/ddev/pkg/versionconstants"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"
	"os"
	osexec "os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"
)

// SetMutagenVolumeOwnership chowns the volume in use to the current user.
// The mutagen volume is mounted both in /var/www (where it gets used) and
// also on /tmp/project_mutagen (where it can be chowned without accidentally hitting
// lots of bind-mounted files).
func SetMutagenVolumeOwnership(app *DdevApp) error {
	// Make sure that if we have a volume mount it's got proper ownership
	uidStr, gidStr, _ := util.GetContainerUIDGid()
	util.Debug("chowning mutagen docker volume for user %s", uidStr)
	_, _, err := app.Exec(
		&ExecOpts{
			Dir: "/tmp",
			Cmd: fmt.Sprintf("sudo chown -R %s:%s /tmp/project_mutagen", uidStr, gidStr),
		})
	if err != nil {
		util.Warning("Failed to chown mutagen volume: %v", err)
	}
	util.Debug("done chowning mutagen docker volume; result=%v", err)
	return err
}

// MutagenSyncName transforms a projectname string into
// an acceptable mutagen sync "name"
// See restrictions on sync name at https://mutagen.io/documentation/introduction/names-labels-identifiers
// The input must be a valid DNS name (valid ddev project name)
func MutagenSyncName(name string) string {
	name = strings.ReplaceAll(name, ".", "")
	if len(name) > 0 && unicode.IsNumber(rune(name[0])) {
		name = "a" + name
	}
	return name
}

// TerminateMutagenSync terminates the mutagen sync
// It is not an error if the sync session does not exist
func TerminateMutagenSync(app *DdevApp) error {
	syncName := MutagenSyncName(app.Name)
	if MutagenSyncExists(app) {
		_, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "terminate", syncName)
		if err != nil {
			return err
		}
		util.Debug("Terminated mutagen sync session '%s'", syncName)
	}
	return nil
}

// SyncAndTerminateMutagenSession syncs and terminates the mutagen sync session
func SyncAndTerminateMutagenSession(app *DdevApp) error {
	if app.Name == "" {
		return fmt.Errorf("No app.Name provided to SyncAndTerminateMutagenSession")
	}
	syncName := MutagenSyncName(app.Name)

	projStatus, _ := app.SiteStatus()

	if !MutagenSyncExists(app) {
		return nil
	}

	mutagenStatus, shortResult, longResult, err := app.MutagenStatus()
	if err != nil {
		return fmt.Errorf("MutagenStatus failed, rv=%v, shortResult=%s, longResult=%s, err=%v", mutagenStatus, shortResult, longResult, err)
	}

	// We don't want to flush if the web container isn't running
	// because mutagen flush will hang forever - disconnected
	if projStatus == SiteRunning && !strings.Contains(longResult, "Connection state: Disconnected") {
		err := app.MutagenSyncFlush()
		if err != nil {
			util.Error("Error on 'mutagen sync flush %s': %v", syncName, err)
		}
	}
	err = TerminateMutagenSync(app)
	return err
}

// GetMutagenConfigFilePath returns the canonical location where the mutagen.yml lives
func GetMutagenConfigFilePath(app *DdevApp) string {
	return filepath.Join(app.GetConfigPath("mutagen"), "mutagen.yml")
}

// GetMutagenConfigFile looks to see if there's a project .mutagen.yml
// If nothing is found, returns empty
func GetMutagenConfigFile(app *DdevApp) string {
	projectConfig := GetMutagenConfigFilePath(app)
	if fileutil.FileExists(projectConfig) {
		return projectConfig
	}
	return ""
}

// CreateMutagenSync creates a sync (after making sure it doesn't exist)
// It detects problems with the sync and errors if there are problems
// and returns the output of `mutagen sync list <syncname>` along with error info
func CreateMutagenSync(app *DdevApp) error {
	syncName := MutagenSyncName(app.Name)
	configFile := GetMutagenConfigFile(app)
	if configFile != "" {
		util.Debug("Using mutagen config file %s", configFile)
	}

	util.Debug("Terminating mutagen sync if session already exists")
	err := TerminateMutagenSync(app)
	if err != nil {
		return err
	}
	container, err := GetContainer(app, "web")
	if err != nil {
		return err
	}
	if container == nil {
		return errors.Errorf("web container not found")
	}

	args := []string{"sync", "create", app.AppRoot, fmt.Sprintf("docker://%s/var/www/html", container.ID), "--no-global-configuration", "--name", syncName}
	if configFile != "" {
		args = append(args, fmt.Sprintf(`--configuration-file=%s`, configFile))
	}
	util.Debug("Creating mutagen sync: mutagen %v", args)
	out, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), args...)
	if err != nil {
		return fmt.Errorf("Failed to mutagen %v (%v), output=%s", args, err, out)
	}
	util.Debug("Flushing mutagen sync session '%s'", syncName)

	flushErr := make(chan error, 1)
	stopGoroutine := make(chan bool, 1)
	firstOutputReceived := make(chan bool, 1)
	defer close(flushErr)
	defer close(stopGoroutine)
	defer close(firstOutputReceived)

	go func() {
		err = app.MutagenSyncFlush()
		flushErr <- err
		return
	}()

	// In tests or other non-interactive environments we don't need to show the
	// mutagen sync monitor output (and it fills up the test logs)

	if os.Getenv("DDEV_NONINTERACTIVE") != "true" {
		go func() {
			previousStatus := ""
			curStatus := ""
			sigSent := false
			cmd := osexec.Command(globalconfig.GetMutagenPath(), "sync", "monitor", syncName)
			stdout, _ := cmd.StdoutPipe()
			err = cmd.Start()
			buf := bufio.NewReader(stdout)
			for {
				select {
				case <-stopGoroutine:
					_ = cmd.Process.Kill()
					_, _ = cmd.Process.Wait()
					return
				default:
					line, err := buf.ReadBytes('\r')
					if err != nil {
						return
					}
					l := string(line)
					if strings.HasPrefix(l, "Status:") {
						// If we haven't already notified that output is coming in,
						// then notify.
						if !sigSent {
							firstOutputReceived <- true
							sigSent = true
							_, _ = fmt.Fprintf(os.Stderr, "\n")
						}

						_, _ = fmt.Fprintf(os.Stderr, "%s", l)
						t := strings.Replace(l, " ", "", 2)
						c := strings.Split(t, " ")
						curStatus = c[0]
						if previousStatus != curStatus {
							_, _ = fmt.Fprintf(os.Stderr, "\n")
						}
						previousStatus = curStatus
					}
				}
			}
		}()
	}

	outputComing := false
	for {
		select {
		// Complete when the MutagenSyncFlush() completes
		case err = <-flushErr:
			return err
		case outputComing = <-firstOutputReceived:

		// If we haven't yet received any "Status:" output, do a dot every second
		case <-time.After(1 * time.Second):
			if !outputComing {
				_, _ = fmt.Fprintf(os.Stderr, ".")
			}
		}
	}
}

// MutagenStatus checks to see if there is an error case in mutagen
// We don't want to do a flush yet in that case.
// Note that the available statuses are at https://github.com/mutagen-io/mutagen/blob/94b9862a06ab44970c7149aa0000628a6adf54d5/pkg/synchronization/state.go#L9
// in func (s Status) Description()
func (app *DdevApp) MutagenStatus() (status string, shortResult string, longResult string, err error) {
	syncName := MutagenSyncName(app.Name)

	longResult, err = exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "list", syncName)
	shortResult = parseMutagenStatusLine(longResult)
	if err != nil {
		// In the odd case where somebody enabled mutagen when it wasn't actually running
		// show a simpler result
		mounted, err := IsMutagenVolumeMounted(app)
		if !mounted {
			return "not enabled", "", "", nil
		}
		return "failing", shortResult, longResult, err
	}

	// We're going to assume that if it's applying changes things are still OK,
	// even though there may be a whole list of problems.
	if strings.Contains(shortResult, "Applying changes") || strings.Contains(shortResult, "Staging files on") || strings.Contains(shortResult, "Reconciling changes") || strings.Contains(shortResult, "Scanning files") || strings.Contains(shortResult, "Watching for changes") || strings.Contains(shortResult, "Saving archive") {
		rv := "ok"
		if strings.Contains(longResult, "problems:") {
			rv = "problems"
		}
		return rv, shortResult, longResult, nil
	}
	if strings.Contains(longResult, "problems") || strings.Contains(longResult, "Conflicts") || strings.Contains(longResult, "error") || strings.Contains(shortResult, "Halted") || strings.Contains(shortResult, "Waiting 5 seconds for rescan") || strings.Contains(longResult, "raw POSIX symbolic links not supported") {
		util.Error("mutagen sync session '%s' is not working correctly: %s", syncName, longResult)
		return "failing", shortResult, longResult, nil
	}
	return "ok", shortResult, longResult, nil
}

// parseMutagenStatusLine takes the full mutagen sync list output and
// return just the right of the Status: line
func parseMutagenStatusLine(fullStatus string) string {
	statusLineLoc := strings.LastIndex(fullStatus, "\nStatus:")
	statusLine := fullStatus[statusLineLoc+1:]
	statusLineEnd := strings.Index(statusLine, "\n")
	if statusLineEnd != -1 {
		statusLine = statusLine[:statusLineEnd]
	}
	pieces := strings.Split(statusLine, ":")
	if len(pieces) < 2 {
		return ""
	}
	return strings.Trim(pieces[1], " \n\r")
}

// MutagenSyncFlush performs a mutagen sync flush, waits for result, and checks for errors
func (app *DdevApp) MutagenSyncFlush() error {
	if app.IsMutagenEnabled() {
		syncName := MutagenSyncName(app.Name)
		if !MutagenSyncExists(app) {
			return errors.Errorf("Mutagen sync session '%s' does not exist", syncName)
		}
		_, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "flush", syncName)
		if err != nil {
			return err
		}

		status, _, _, err := app.MutagenStatus()
		if status != "ok" || err != nil {
			return err
		}
		util.Debug("Flushed mutagen sync session '%s'", syncName)
	}
	return nil
}

// MutagenSyncExists detects whether the named sync exists
func MutagenSyncExists(app *DdevApp) bool {
	syncName := MutagenSyncName(app.Name)

	if !fileutil.FileExists(globalconfig.GetMutagenPath()) {
		return false
	}
	c := []string{globalconfig.GetMutagenPath(), "sync", "list", syncName}
	out, err := exec.RunHostCommand(c[0], c[1:]...)
	if err != nil && !strings.Contains(out, "Error: unable to locate requested sessions") {
		util.Warning("%v failed: %v output=%v", c, err, out)
	}
	return err == nil
}

// DownloadMutagen gets the mutagen binary and related and puts it into
// ~/.ddev/.bin
func DownloadMutagen() error {
	StopMutagenDaemon()
	flavor := runtime.GOOS + "_" + runtime.GOARCH
	globalMutagenDir := filepath.Dir(globalconfig.GetMutagenPath())
	destFile := filepath.Join(globalMutagenDir, "mutagen.tgz")
	mutagenURL := fmt.Sprintf("https://github.com/mutagen-io/mutagen/releases/download/v%s/mutagen_%s_v%s.tar.gz", versionconstants.RequiredMutagenVersion, flavor, versionconstants.RequiredMutagenVersion)
	output.UserOut.Printf("Downloading %s ...", mutagenURL)

	// Remove the existing file. This may help on macOS to prevent the Gatekeeper's
	// caching bug from confusing with a previously downloaded file?
	// Discussion in https://github.com/mutagen-io/mutagen/issues/290#issuecomment-906612749
	_ = os.Remove(globalconfig.GetMutagenPath())

	_ = os.MkdirAll(globalMutagenDir, 0777)
	err := util.DownloadFile(destFile, mutagenURL, "true" != os.Getenv("DDEV_NONINTERACTIVE"))
	if err != nil {
		return err
	}
	output.UserOut.Printf("Download complete.")

	err = archive.Untar(destFile, globalMutagenDir, "")
	_ = os.Remove(destFile)
	if err != nil {
		return err
	}
	err = os.Chmod(globalconfig.GetMutagenPath(), 0755)
	if err != nil {
		return err
	}

	// Stop daemon in case it was already running somewhere else
	StopMutagenDaemon()
	return nil
}

// StopMutagenDaemon will try to stop a running mutagen daemon
// But no problem if there wasn't one
func StopMutagenDaemon() {
	if fileutil.FileExists(globalconfig.GetMutagenPath()) {
		out, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), "daemon", "stop")
		if err != nil && !strings.Contains(out, "unable to connect to daemon") {
			util.Warning("Unable to stop mutagen daemon: %v", err)
		}
		util.Success("Stopped mutagen daemon")
	}
}

// DownloadMutagenIfNeeded downloads the proper version of mutagen
// if it's either not yet installed or has the wrong version.
func DownloadMutagenIfNeeded(app *DdevApp) error {
	if !app.IsMutagenEnabled() {
		return nil
	}
	curVersion, err := version.GetLiveMutagenVersion()
	if err != nil || curVersion != versionconstants.RequiredMutagenVersion {
		err = DownloadMutagen()
		if err != nil {
			return err
		}
	}
	return nil
}

// MutagenReset stops (with flush), removes the docker volume, starts again (with flush)
func MutagenReset(app *DdevApp) error {
	if app.IsMutagenEnabled() {
		err := app.Stop(false, false)
		if err != nil {
			return errors.Errorf("Failed to stop project %s: %v", app.Name, err)
		}
		err = dockerutil.RemoveVolume(GetMutagenVolumeName(app))
		if err != nil {
			return err
		}
		output.UserOut.Printf("Removed docker volume %s", GetMutagenVolumeName(app))
	}
	return nil
}

// GetMutagenVolumeName returns the name for the mutagen docker volume
func GetMutagenVolumeName(app *DdevApp) string {
	return app.Name + "_" + "project_mutagen"
}

// MutagenMonitor shows the output of `mutagen sync monitor <syncName>`
func MutagenMonitor(app *DdevApp) {
	syncName := MutagenSyncName(app.Name)

	// This doesn't actually return; you have to <ctrl-c> to end it
	c := osexec.Command(globalconfig.GetMutagenPath(), "sync", "monitor", syncName)
	// We only need all three of these because of Windows behavior on git-bash with no pty
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	_ = c.Run()
}

//go:embed mutagen_config_assets
var mutagenConfigAssets embed.FS

// GenerateMutagenYml generates the .ddev/mutagen.yml
func (app *DdevApp) GenerateMutagenYml() error {
	// Prevent running as root for most cases
	// We really don't want ~/.ddev to have root ownership, breaks things.
	if os.Geteuid() == 0 {
		util.Warning("not generating mutagen config file because running with root privileges")
		return nil
	}
	mutagenYmlPath := GetMutagenConfigFilePath(app)
	if sigExists, err := fileutil.FgrepStringInFile(mutagenYmlPath, nodeps.DdevFileSignature); err == nil && !sigExists {
		// If the signature doesn't exist, they have taken over the file, so return
		return nil
	}

	c, err := mutagenConfigAssets.ReadFile(path.Join("mutagen_config_assets", "mutagen.yml"))
	if err != nil {
		return err
	}
	content := string(c)

	// It's impossible to use posix-raw on traditional windows.
	// But this means that there will be errors with rooted symlinks in the container on windows
	symlinkMode := "posix-raw"
	if runtime.GOOS == "windows" {
		symlinkMode = "portable"
	}
	err = os.MkdirAll(filepath.Dir(mutagenYmlPath), 0755)
	if err != nil {
		return err
	}

	uploadDir := ""
	if app.GetUploadDir() != "" {
		uploadDir = path.Join(app.Docroot, app.GetUploadDir())
	}

	templateMap := map[string]interface{}{
		"SymlinkMode": symlinkMode,
		"UploadDir":   uploadDir,
	}
	// If no bind mounts, then we can't ignore UploadDir, must sync it
	if globalconfig.DdevGlobalConfig.NoBindMounts {
		templateMap["UploadDir"] = ""
	}

	err = fileutil.TemplateStringToFile(content, templateMap, mutagenYmlPath)
	return err
}

// IsMutagenVolumeMounted checks to see if the mutagen volume is mounted
func IsMutagenVolumeMounted(app *DdevApp) (bool, error) {
	client := dockerutil.GetDockerClient()
	container, err := dockerutil.FindContainerByName("ddev-" + app.Name + "-web")
	// If there is no web container found, the volume is not mounted
	if err != nil || container == nil {
		return false, nil
	}
	inspect, err := client.InspectContainerWithOptions(docker.InspectContainerOptions{
		ID: container.ID,
	})
	if err != nil {
		return false, err
	}
	for _, m := range inspect.Mounts {
		if m.Name == app.Name+"_project_mutagen" {
			return true, nil
		}
	}
	return false, nil
}

// IsMutagenEnabled returns true if mutagen is enabled locally or globally
// It's also required and set if NoBindMounts is set, since we have to have a way
// to get code on there.
func (app *DdevApp) IsMutagenEnabled() bool {
	return app.MutagenEnabled || app.MutagenEnabledGlobal || globalconfig.DdevGlobalConfig.NoBindMounts
}
