package ddevapp

import (
	"bufio"
	"embed"
	"encoding/json"
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

const mutagenSignatureLabelName = `com.ddev.volume-signature`

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

// TerminateMutagenSync destroys a mutagen sync session
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

// PauseMutagenSync pauses a mutagen sync session
func PauseMutagenSync(app *DdevApp) error {
	syncName := MutagenSyncName(app.Name)
	if MutagenSyncExists(app) {
		_, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "pause", syncName)
		if err != nil {
			return err
		}
		util.Debug("Paused mutagen sync session '%s'", syncName)
	}
	return nil
}

// SyncAndPauseMutagenSession syncs and pauses a mutagen sync session
func SyncAndPauseMutagenSession(app *DdevApp) error {
	if app.Name == "" {
		return fmt.Errorf("No app.Name provided to SyncAndPauseMutagenSession")
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
	if projStatus == SiteRunning && (mutagenStatus == "ok" || mutagenStatus == "problems") {
		err := app.MutagenSyncFlush()
		if err != nil {
			util.Error("Error on 'mutagen sync flush %s': %v", syncName, err)
		}
	}
	err = PauseMutagenSync(app)
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

// CreateOrResumeMutagenSync creates or resumes a sync session
// It detects problems with the sync and errors if there are problems
// and returns the output of `mutagen sync list <syncname>` along with error info
func CreateOrResumeMutagenSync(app *DdevApp) error {
	syncName := MutagenSyncName(app.Name)
	configFile := GetMutagenConfigFile(app)
	if configFile != "" {
		util.Debug("Using mutagen config file %s", configFile)
	}

	container, err := GetContainer(app, "web")
	if err != nil {
		return err
	}
	if container == nil {
		return fmt.Errorf("web container for %s not found", app.Name)
	}
	if container.State != "running" {
		// TODO: Improve or debug this temporary debug usage
		util.Warning("web container is not running, logs follow")
		logsErr := app.Logs("web", false, false, "100")
		if logsErr != nil {
			util.Warning("error from getting logs: %v", logsErr)
		}
		return fmt.Errorf("Cannot start mutagen sync because web container is not running: %v", container)
	}

	//TODO: Detect if an appropriate mutagen sync session exists
	sessionExists, err := mutagenSyncSessionExists(app)
	if err != nil {
		return err
	}
	if sessionExists {
		util.Debug("Resume mutagen sync if session already exists")
		err := ResumeMutagenSync(app)
		if err != nil {
			return err
		}
	} else {
		vLabel, err := GetMutagenVolumeLabel(app)
		if err != nil {
			return err
		}
		// TODO: Consider using a function to specify the docker beta
		args := []string{"sync", "create", app.AppRoot, fmt.Sprintf("docker:/%s/var/www/html", container.Names[0]), "--no-global-configuration", "--name", syncName, "--label", mutagenSignatureLabelName + "=" + vLabel}
		if configFile != "" {
			args = append(args, fmt.Sprintf(`--configuration-file=%s`, configFile))
		}
		util.Debug("Creating mutagen sync: mutagen %v", args)
		out, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), args...)
		if err != nil {
			return fmt.Errorf("Failed to mutagen %v (%v), output=%s", args, err, out)
		}
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

func ResumeMutagenSync(app *DdevApp) error {
	args := []string{"sync", "resume", MutagenSyncName(app.Name)}
	util.Debug("Resuming mutagen sync: mutagen %v", args)
	out, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), args...)
	if err != nil {
		return fmt.Errorf("Failed to mutagen %v (%v), output=%s", args, err, out)
	}
	return nil
}

// mutagenSyncSessionExists determines whether an appropriate mutagen sync session already exists
// if it finds one with invalid label, it destroys the existing session.
func mutagenSyncSessionExists(app *DdevApp) (bool, error) {
	syncName := MutagenSyncName(app.Name)
	res, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "list", "--template", "{{ json . }}", syncName)
	if err != nil {
		if strings.Contains(res, "did not match any sessions") {
			return false, nil
		}
		return false, err
	}
	sessionMap := make([]map[string]interface{}, 2)
	err = json.Unmarshal([]byte(res), &sessionMap)
	if err != nil {
		return false, err
	}
	if len(sessionMap) != 1 {
		return false, fmt.Errorf("mutagen sessions may be in invalid state, please `ddev mutagen reset`")
	}

	// Find out if mutagen session labels has label we found in docker volume
	session := sessionMap[0]

	if l, ok := session["labels"].(map[string]interface{}); ok {
		vLabel, vLabelErr := GetMutagenVolumeLabel(app)
		if s, ok := l[mutagenSignatureLabelName]; ok && vLabelErr == nil && vLabel != "" && vLabel == s {
			return true, nil
		}
		// If we happen to find a mutagen session without matching signature, terminate it.
		_ = TerminateMutagenSync(app)
	}
	return false, nil
}

// MutagenStatus checks to see if there is an error case in mutagen
// We don't want to do a flush yet in that case.
// Note that the available statuses are at https://github.com/mutagen-io/mutagen/blob/master/pkg/synchronization/state.go#L9
// in func (s Status) Description()
// Can return any of those or "nosession" if we didn't find a session at all
func (app *DdevApp) MutagenStatus() (status string, shortResult string, mapResult map[string]interface{}, err error) {
	syncName := MutagenSyncName(app.Name)

	fullJSONResult, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "list", "--template", `{{ json .}}`, syncName)
	if err != nil {
		return "nosession", fullJSONResult, nil, err
	}
	sessionMap := make([]map[string]interface{}, 2)
	err = json.Unmarshal([]byte(fullJSONResult), &sessionMap)
	if err != nil {
		return "nosession", fullJSONResult, nil, err
	}
	if len(sessionMap) < 1 {
		return "", "", nil, fmt.Errorf("mutagen sessions may be in invalid state, please `ddev mutagen reset`")
	}
	mapRes := sessionMap[0]

	if paused, ok := mapRes["paused"].(bool); ok && paused == true {
		return "paused", "paused", mapRes, nil
	}
	var ok bool
	mutagenStatus := ""
	if mutagenStatus, ok = mapRes["status"].(string); !ok {
		return "failing", mutagenStatus, mapRes, fmt.Errorf("mutagen sessions may be in invalid state, please `ddev mutagen reset`")
	}
	// In the odd case where somebody enabled mutagen when it wasn't actually running
	// show a simpler result
	mounted, err := IsMutagenVolumeMounted(app)
	if !mounted {
		return "not enabled", "", mapRes, nil
	}
	if err != nil {
		return "", "", nil, err
	}

	problems := false
	if alpha, ok := mapRes["alpha"].(map[string]interface{}); ok {
		if _, ok = alpha["scanProblems"]; ok {
			problems = true
		}
	}
	if beta, ok := mapRes["beta"].(map[string]interface{}); ok {
		if _, ok = beta["scanProblems"]; ok {
			problems = true
		}
	}
	if _, ok := mapRes["conflicts"]; ok {
		problems = true
	}

	// We're going to assume that if it's applying changes things are still OK,
	// even though there may be a whole list of problems.
	// States from json are in https://github.com/mutagen-io/mutagen/blob/bc07f2f0f3f0aba0aff0514bd4739d75444091fe/pkg/synchronization/state.go#L47-L79
	switch mutagenStatus {
	case "paused":
		return "paused", shortResult, mapRes, nil
	case "transitioning":
		fallthrough
	case "staging-alpha":
		fallthrough
	case "staging-beta":
		fallthrough
	case "reconciling":
		fallthrough
	case "scanning":
		fallthrough
	case "saving":
		fallthrough
	case "watching":
		if !problems {
			status = "ok"
		} else {
			status = "problems"
		}
		return status, shortResult, mapRes, nil
	}
	return "failing", shortResult, mapRes, nil
}

// MutagenSyncFlush performs a mutagen sync flush, waits for result, and checks for errors
func (app *DdevApp) MutagenSyncFlush() error {
	if app.IsMutagenEnabled() {
		syncName := MutagenSyncName(app.Name)
		if !MutagenSyncExists(app) {
			return errors.Errorf("Mutagen sync session '%s' does not exist", syncName)
		}
		status, _, _, _ := app.MutagenStatus()
		if status != "paused" {
			out, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "flush", syncName)
			if err != nil {
				return fmt.Errorf("mutagen sync flush %s failed, output=%s, err=%v", syncName, out, err)
			}
		}

		status, _, _, err := app.MutagenStatus()
		if (status != "ok" && status != "problems" && status != "paused") || err != nil {
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
	// List syncs with this name that also match appropriate labels
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
		util.Debug("Removed docker volume %s", GetMutagenVolumeName(app))
		err = TerminateMutagenSync(app)
		if err != nil {
			return err
		}
		util.Debug("Terminated mutagen sync session %s", MutagenSyncName(app.Name))
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

// GetMutagenVolumeLabel returns the com.ddev.volume-signature on the project_mutagen docker volume
func GetMutagenVolumeLabel(app *DdevApp) (string, error) {
	labels, err := dockerutil.VolumeLabels(GetMutagenVolumeName(app))
	if err != nil {
		return "", err
	}
	if labels != nil {
		if l, ok := labels[mutagenSignatureLabelName]; ok {
			return l, nil
		}
	}
	return "", nil
}

// CheckMutagenVolumeSyncCompatibility checks to see if the mutagen label and volume label
// are the same.
// Returns true if they're the same, false if they're different
// If either is empty, then return false. Terminate mutagen session if it has no label.
func CheckMutagenVolumeSyncCompatibility(app *DdevApp) bool {
	mutagenLabel, mutagenSyncLabelErr := GetMutagenSyncLabel(app)
	volumeLabel, volumeLabelErr := GetMutagenVolumeLabel(app)

	switch {
	// If both are errors, we don't have either session or label,
	// so we are compatible.
	case mutagenSyncLabelErr != nil && volumeLabelErr != nil:
		return true

	// If there are legitimate session + volume and labels match, we're good to continue.
	case mutagenSyncLabelErr == nil && volumeLabelErr == nil && mutagenLabel != "" && mutagenLabel == volumeLabel:
		return true

	// If we have a legitimate label (meaning real session) with no label
	// then it's a holdover from earlier ddev version, kill it.
	case mutagenSyncLabelErr == nil && mutagenLabel == "":
		util.Debug("mutagen sync session has empty label, terminating")
		err := TerminateMutagenSync(app)
		if err != nil {
			util.Debug("failed to terminate mutagen sync session: %v", err)
		}
	}

	return false
}

// GetMutagenSyncLabel gets the com.ddev.volume-signature label from an existing sync session
func GetMutagenSyncLabel(app *DdevApp) (string, error) {
	status, _, mapResult, err := app.MutagenStatus()

	if status == "nosession" || err != nil {
		return "", fmt.Errorf("no session %s found", MutagenSyncName(app.Name))
	}
	if labels, ok := mapResult["labels"].(map[string]interface{}); ok {
		if label, ok := labels[mutagenSignatureLabelName].(string); ok {
			return label, nil
		}
	}
	return "", fmt.Errorf("sync session label not found for sync session %s", MutagenSyncName(app.Name))
}

// TerminateAllMutagenSync terminates all sessions that match our signature label
func TerminateAllMutagenSync() {
	out, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "terminate", "--label-selector="+mutagenSignatureLabelName)
	if err != nil {
		util.Warning("could not terminate all mutagen sessions, output=%s, err=%v", out, err)
	}
}
