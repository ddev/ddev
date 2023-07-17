package ddevapp

import (
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	osexec "os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/version"
	"github.com/ddev/ddev/pkg/versionconstants"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"
)

const mutagenSignatureLabelName = `com.ddev.volume-signature`
const mutagenConfigFileHashLabelName = `com.ddev.config-hash`

// SetMutagenVolumeOwnership chowns the volume in use to the current user.
// The mutagen volume is mounted both in /var/www (where it gets used) and
// also on /tmp/project_mutagen (where it can be chowned without accidentally hitting
// lots of bind-mounted files).
func SetMutagenVolumeOwnership(app *DdevApp) error {
	// Make sure that if we have a volume mount it's got proper ownership
	uidStr, gidStr, _ := util.GetContainerUIDGid()
	util.Verbose("chowning mutagen docker volume for user %s", uidStr)
	_, _, err := app.Exec(
		&ExecOpts{
			Dir: "/tmp",
			Cmd: fmt.Sprintf("sudo chown -R %s:%s /tmp/project_mutagen", uidStr, gidStr),
		})
	if err != nil {
		util.Warning("Failed to chown mutagen volume: %v", err)
	}
	util.Verbose("done chowning mutagen docker volume; result=%v", err)

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

// GetMutagenConfigFileHash returns the SHA1 hash of the mutagen.yml
func GetMutagenConfigFileHash(app *DdevApp) (string, error) {
	f := GetMutagenConfigFilePath(app)
	hash, err := fileutil.FileHash(f)
	if err != nil {
		return "", err
	}
	return hash, nil
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

	sessionExists, err := mutagenSyncSessionExists(app)
	if err != nil {
		return err
	}
	if sessionExists {
		util.Verbose("Resume mutagen sync if session already exists")
		err := ResumeMutagenSync(app)
		if err != nil {
			return err
		}
	} else {
		vLabel, err := GetMutagenVolumeLabel(app)
		if err != nil {
			return err
		}

		hLabel, err := GetMutagenConfigFileHash(app)
		if err != nil {
			return err
		}
		// TODO: Consider using a function to specify the docker beta
		args := []string{"sync", "create", app.AppRoot, fmt.Sprintf("docker:/%s/var/www/html", container.Names[0]), "--no-global-configuration", "--name", syncName, "--label", mutagenSignatureLabelName + "=" + vLabel, "--label", mutagenConfigFileHashLabelName + "=" + hLabel}
		if configFile != "" {
			args = append(args, fmt.Sprintf(`--configuration-file=%s`, configFile))
		}
		// On Windows, permissions can't be inferred from what is on the host side, so just force 777 for
		// most things
		if runtime.GOOS == "windows" {
			args = append(args, []string{"--permissions-mode=manual", "--default-file-mode-beta=0777", "--default-directory-mode-beta=0777"}...)
		}
		util.Debug("Creating mutagen sync: mutagen %v", args)
		out, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), args...)
		if err != nil {
			return fmt.Errorf("Failed to mutagen %v (%v), output=%s", args, err, out)
		}
	}

	util.Verbose("Flushing mutagen sync session '%s'", syncName)
	flushErr := make(chan error, 1)
	stopGoroutine := make(chan bool, 1)
	firstOutputReceived := make(chan bool, 1)
	defer close(flushErr)
	defer close(stopGoroutine)
	defer close(firstOutputReceived)

	go func() {
		err = app.MutagenSyncFlush()
		util.Verbose("gofunc flushed mutagen sync session '%s' err=%v", syncName, err)
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
	util.Verbose("Resuming mutagen sync: mutagen %v", args)
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
	res, err := exec.RunHostCommandSeparateStreams(globalconfig.GetMutagenPath(), "sync", "list", "--template", "{{ json (index . 0) }}", syncName)
	if err != nil {
		if exitError, ok := err.(*osexec.ExitError); ok {
			// If we got an error, but it's that there were no sessions, return false, no err
			if strings.Contains(string(exitError.Stderr), "did not match any sessions") {
				return false, nil
			}
		}
		return false, err
	}
	session := make(map[string]interface{})
	err = json.Unmarshal([]byte(res), &session)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshall mutagen sync list results '%v': %v", res, err)
	}

	// Find out if mutagen session labels has label we found in docker volume
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
// Can return any of those or "nosession" (with more info) if we didn't find a session at all
func (app *DdevApp) MutagenStatus() (status string, shortResult string, mapResult map[string]interface{}, err error) {
	syncName := MutagenSyncName(app.Name)

	mutagenDataDirectory := os.Getenv("MUTAGEN_DATA_DIRECTORY")
	fullJSONResult, err := exec.RunHostCommandSeparateStreams(globalconfig.GetMutagenPath(), "sync", "list", "--template", `{{ json (index . 0) }}`, syncName)
	if err != nil {
		stderr := ""
		if exitError, ok := err.(*osexec.ExitError); ok {
			stderr = string(exitError.Stderr)
		}
		return fmt.Sprintf("nosession for MUTAGEN_DATA_DIRECTORY=%s", mutagenDataDirectory), fullJSONResult, nil, fmt.Errorf("failed to mutagen sync list %s: stderr='%s', err=%v", syncName, stderr, err)
	}
	session := make(map[string]interface{})
	err = json.Unmarshal([]byte(fullJSONResult), &session)
	if err != nil {
		return fmt.Sprintf("nosession for MUTAGEN_DATA_DIRECTORY=%s; failed to unmarshall mutagen sync list results '%v'", mutagenDataDirectory, fullJSONResult), fullJSONResult, nil, err
	}

	if paused, ok := session["paused"].(bool); ok && paused == true {
		return "paused", "paused", session, nil
	}
	var ok bool
	if shortResult, ok = session["status"].(string); !ok {
		return "failing", shortResult, session, fmt.Errorf("mutagen sessions may be in invalid state, please `ddev mutagen reset`")
	}
	shortResult = session["status"].(string)

	// In the odd case where somebody enabled mutagen when it wasn't actually running
	// show a simpler result
	mounted, err := IsMutagenVolumeMounted(app)
	if !mounted {
		return "not enabled", "", session, nil
	}
	if err != nil {
		return "", "", nil, err
	}

	problems := false
	if alpha, ok := session["alpha"].(map[string]interface{}); ok {
		if _, ok = alpha["scanProblems"]; ok {
			problems = true
		}
	}
	if beta, ok := session["beta"].(map[string]interface{}); ok {
		if _, ok = beta["scanProblems"]; ok {
			problems = true
		}
	}
	if _, ok := session["conflicts"]; ok {
		problems = true
	}

	// We're going to assume that if it's applying changes things are still OK,
	// even though there may be a whole list of problems.
	// States from json are in https://github.com/mutagen-io/mutagen/blob/bc07f2f0f3f0aba0aff0514bd4739d75444091fe/pkg/synchronization/state.go#L47-L79
	switch shortResult {
	case "paused":
		return "paused", shortResult, session, nil
	case "transitioning":
		fallthrough
	case "staging-alpha":
		fallthrough
	case "connecting-beta":
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
		return status, shortResult, session, nil
	}
	return "failing", shortResult, session, nil
}

// GetMutagenSyncID() returns the project sync ID
func (app *DdevApp) GetMutagenSyncID() (id string, err error) {
	syncName := MutagenSyncName(app.Name)

	identifier, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "list", `--template='{{ range . }}{{ .Identifier }}{{ break }}{{ end }}'`, syncName)
	if err != nil {
		return "", fmt.Errorf("failed RunHostCommand, output='%s': %v", identifier, err)
	}

	return identifier, nil
}

// MutagenSyncFlush performs a mutagen sync flush, waits for result, and checks for errors
func (app *DdevApp) MutagenSyncFlush() error {
	if !app.IsMutagenEnabled() {
		return nil
	}

	container, err := GetContainer(app, "web")
	if err != nil {
		return fmt.Errorf("failed to get web container, err='%v'", err)
	}

	// Discussions of container.State in
	// https://stackoverflow.com/questions/32427684/what-are-the-possible-states-for-a-docker-container
	// and https://medium.com/@BeNitinAgarwal/lifecycle-of-docker-container-d2da9f85959
	if container.State != "running" {
		return fmt.Errorf("mutagenSyncFlush() not mutagen-syncing project %s with web container is in state %s, but must be 'running'", app.Name, container.State)
	}
	syncName := MutagenSyncName(app.Name)
	if !MutagenSyncExists(app) {
		return errors.Errorf("Mutagen sync session '%s' does not exist", syncName)
	}
	if status, shortResult, session, err := app.MutagenStatus(); err == nil {
		util.Verbose("mutagen sync %s status='%s', shortResult='%v', session='%v', err='%v'", syncName, status, shortResult, session, err)
		switch status {
		case "paused":
			util.Debug("mutagen sync %s is paused, so not flushing", syncName)
			return nil
		case "failing":
			util.Warning("mutagen sync session %s has status '%s': shortResult='%v', err=%v, session contents='%v'", syncName, status, shortResult, err, session)
		default:
			// This extra sync resume recommended by @xenoscopic to catch situation where
			// not paused but also not connected, in which case the flush will fail.
			util.Verbose("default case resuming mutagen sync session '%s'", syncName)
			out, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "resume", syncName)
			if err != nil {
				return fmt.Errorf("mutagen resume flush %s failed, output=%s, err=%v", syncName, out, err)
			}
			util.Verbose("default case flushing mutagen sync session '%s'", syncName)
			out, err = exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "flush", syncName)
			if err != nil {
				return fmt.Errorf("mutagen sync flush %s failed, output=%s, err=%v", syncName, out, err)
			}
			util.Verbose("default case output of mutagen sync='%s'", out)
		}
	}

	status, short, _, err := app.MutagenStatus()
	util.Verbose("mutagen sync status %s in MutagenSyncFlush(): status='%s', short='%s', err='%v'", syncName, status, short, err)
	if (status != "ok" && status != "problems" && status != "paused" && status != "failing") || err != nil {
		return err
	}
	util.Verbose("Flushed mutagen sync session '%s'", syncName)
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
		mutagenDataDirectory := os.Getenv("MUTAGEN_DATA_DIRECTORY")
		out, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), "daemon", "stop")
		if err != nil && !strings.Contains(out, "unable to connect to daemon") {
			util.Warning("Unable to stop mutagen daemon: %v; MUTAGEN_DATA_DIRECTORY=%s", err, mutagenDataDirectory)
		}
		util.Success("Stopped mutagen daemon")
	}
}

// StartMutagenDaemon will make sure the daemon is running
func StartMutagenDaemon() {
	if fileutil.FileExists(globalconfig.GetMutagenPath()) {
		out, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), "daemon", "start")
		if err != nil {
			util.Warning("Failed to run mutagen daemon start: %v, out=%s", err, out)
		}
	}
}

// DownloadMutagenIfNeededAndEnabled downloads the proper version of mutagen
// if it's enabled and if it's either not yet installed or has the wrong version.
func DownloadMutagenIfNeededAndEnabled(app *DdevApp) error {
	if !app.IsMutagenEnabled() {
		return nil
	}
	return DownloadMutagenIfNeeded()
}

// DownloadMutagenIfNeeded downloads mutagen if we don't have it or there's an update
func DownloadMutagenIfNeeded() error {
	err := os.MkdirAll(globalconfig.GetMutagenDataDirectory(), 0755)
	if err != nil {
		return err
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
	}
	err := TerminateMutagenSync(app)
	if err != nil {
		return err
	}
	util.Debug("Terminated mutagen sync session %s", MutagenSyncName(app.Name))
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

	templateMap := map[string]interface{}{
		"SymlinkMode": symlinkMode,
		"UploadDirs":  app.getUploadDirsRelative(),
	}

	// If no bind mounts, then we can't ignore UploadDir, must sync it
	if globalconfig.DdevGlobalConfig.NoBindMounts {
		templateMap["UploadDirs"] = []string{}
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
	return app.GetPerformanceMode() == types.PerformanceModeMutagen || globalconfig.DdevGlobalConfig.NoBindMounts
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
// Compatible if:
//   - No volume (or no volume and no mutagen sync session)
//   - Volume and mutagen sync exist and Volume label matches mutagen label
//
// Not compatible if
//   - Volume and mutagen sync exist and have different labels
//   - Volume exists (with label) but there's no mutagen sync session matching it. In this case we'd want
//     to start from scratch with a new volume and sync, so we get authoritative files from alpha (host)
//   - Volume has a label that is not based on this docker context.
//
// Return ok, info, where ok true if compatible, info gives reasoning
func CheckMutagenVolumeSyncCompatibility(app *DdevApp) (ok bool, volumeExists bool, info string) {
	mutagenSyncExists := MutagenSyncExists(app)
	volumeLabel, volumeLabelErr := GetMutagenVolumeLabel(app)
	dockerHostID := dockerutil.GetDockerHostID()
	mutagenLabel := ""
	configFileHashLabel := ""
	var mutagenSyncLabelErr error
	var configFileHashLabelErr error

	volumeExists = !(volumeLabelErr != nil && errors.Is(docker.ErrNoSuchVolume, volumeLabelErr))
	calculatedConfigFileHash, err := GetMutagenConfigFileHash(app)
	if err != nil {
		util.Warning("unable to calculate Mutagen config file hash: %v", err)
	}
	if mutagenSyncExists {
		mutagenLabel, mutagenSyncLabelErr = GetMutagenSyncLabel(app)
		if mutagenSyncLabelErr != nil {
			util.Warning("mutagen sync session '%s' exists but unable to get sync label '%s': '%v' This is normal on upgrade from v1.21.6; error=%v", app.Name, mutagenSignatureLabelName, mutagenLabel, mutagenSyncLabelErr)
		}
		configFileHashLabel, configFileHashLabelErr = GetMutagenConfigFileHashLabel(app)
		if configFileHashLabelErr != nil {
			util.Warning("mutagen sync session '%s' exists but unable to get sync label '%s': '%v' This is normal on upgrade from v1.21.6; error=%v", app.Name, mutagenConfigFileHashLabelName, configFileHashLabel, configFileHashLabelErr)
		}
	}
	switch {
	case configFileHashLabel != calculatedConfigFileHash:
		return false, volumeExists, "calculated mutagen.yml hash does not equal session label"
	// If there is no volume, everything is fine, proceed.
	case !volumeExists:
		return true, volumeExists, "no docker volume exists, so compatible"
	case mutagenSyncLabelErr != nil:
		return false, volumeExists, "mutagen sync session exists but does not have label"
	// If the labels do not have the current context as first part of label, we have trouble.
	case !strings.HasPrefix(volumeLabel, dockerHostID) || !strings.HasPrefix(mutagenLabel, dockerHostID):
		return false, volumeExists, fmt.Sprintf("volume label '%s' or sync label '%s' does not start with current dockerHostID (%s)", volumeLabel, mutagenLabel, dockerHostID)
	// if we have labels for both and they match, it's all fine.
	case mutagenLabel == volumeLabel:
		return true, volumeExists, fmt.Sprintf("volume and mutagen sync session have the same label: %s", volumeLabel)
	}

	return false, volumeExists, fmt.Sprintf("CheckMutagenVolumeSyncCompatibility: currentDockerContext=%s mutagenLabel='%s', volumeLabel='%s', mutagenSyncLabelErr='%v', volumeLabelErr='%v'", dockerutil.DockerContext, mutagenLabel, volumeLabel, mutagenSyncLabelErr, volumeLabelErr)
}

// GetMutagenSyncLabel gets the com.ddev.volume-signature label from an existing sync session
func GetMutagenSyncLabel(app *DdevApp) (string, error) {
	status, _, mapResult, err := app.MutagenStatus()

	if strings.HasPrefix(status, "nosession") || err != nil {
		return "", fmt.Errorf("no session %s found: %v", MutagenSyncName(app.Name), status)
	}
	if labels, ok := mapResult["labels"].(map[string]interface{}); ok {
		if label, ok := labels[mutagenSignatureLabelName].(string); ok {
			return label, nil
		}
	}
	return "", fmt.Errorf("sync session label not found for sync session %s", MutagenSyncName(app.Name))
}

// GetMutagenConfigFileHashLabel gets the com.ddev.hash- label from an existing sync session
func GetMutagenConfigFileHashLabel(app *DdevApp) (string, error) {
	status, _, mapResult, err := app.MutagenStatus()

	if strings.HasPrefix(status, "nosession") || err != nil {
		return "", fmt.Errorf("no session %s found: %v", MutagenSyncName(app.Name), status)
	}
	if labels, ok := mapResult["labels"].(map[string]interface{}); ok {
		if label, ok := labels[mutagenConfigFileHashLabelName].(string); ok {
			return label, nil
		}
	}
	return "", fmt.Errorf("configFilehash label not found for sync session %s", MutagenSyncName(app.Name))
}

// TerminateAllMutagenSync terminates all sync sessions
func TerminateAllMutagenSync() {
	if fileutil.FileExists(globalconfig.GetMutagenPath()) {
		out, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "terminate", "-a")
		if err != nil {
			util.Warning("could not terminate all mutagen sessions (mutagen sync terminate -a), output=%s, err=%v", out, err)
		}
	}
}

// GetDefaultMutagenVolumeSignature gets a new volume signature to be applied to mutagen volume
func GetDefaultMutagenVolumeSignature(_ *DdevApp) string {
	return fmt.Sprintf("%s-%v", dockerutil.GetDockerHostID(), time.Now().Unix())
}

// checkMutagenUploadDirs just tells people if they are using mutagen without upload_dir
func (app *DdevApp) checkMutagenUploadDirs() {
	if app.IsMutagenEnabled() && !app.IsUploadDirsWarningDisabled() && len(app.GetUploadDirs()) == 0 {
		util.Warning("You have Mutagen enabled and your '%s' project type doesn't have `upload_dirs` set.", app.Type)
		util.Warning("For faster startup and less disk usage, set upload_dirs to where your user-generated files are stored.")
		util.Warning("If this is intended you can disable this warning with `ddev config --disable-upload-dirs-warning`.")
	}
}
