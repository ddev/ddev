package ddevapp

import (
	"bufio"
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
	"github.com/pkg/errors"
	"golang.org/x/term"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// SetMutagenVolumeOwnership chowns the volume in use to the current user.
func SetMutagenVolumeOwnership(app *DdevApp) error {
	// Make sure that if we have a volume mount it's got proper ownership
	uidStr, gidStr, _ := util.GetContainerUIDGid()
	util.Debug("chowning mutagen docker volume for user %s", uidStr)
	_, _, err := app.Exec(
		&ExecOpts{
			Dir: "/tmp",
			Cmd: fmt.Sprintf("sudo chown -R %s:%s /var/www/html", uidStr, gidStr),
		})
	util.Debug("done chowning mutagen docker volume, result=%v", err)
	return err
}

// MutagenSyncName transforms a projectname string into
// an acceptable mutagen sync "name"
// See restrictions on sync name at https://mutagen.io/documentation/introduction/names-labels-identifiers
// The input must be a valid DNS name (valid ddev project name)
func MutagenSyncName(name string) string {
	return strings.ReplaceAll(name, ".", "")
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
		util.Success("Terminated mutagen sync session '%s'", syncName)
	}
	return nil
}

// SyncAndTerminateMutagenSession syncs and terminates the mutagen sync session
func SyncAndTerminateMutagenSession(app *DdevApp) error {
	syncName := MutagenSyncName(app.Name)

	if !MutagenSyncExists(app) {
		return nil
	}
	err := app.MutagenSyncFlush()
	if err != nil {
		util.Error("Error on 'mutagen sync flush %s': %v", syncName, err)
	}
	err = TerminateMutagenSync(app)
	if err != nil {
		return err
	}
	return nil
}

// GetMutagenConfigFile looks to see if there's a project .mutagen.yml
// If nothing is found, returns empty
func GetMutagenConfigFile(app *DdevApp) string {
	projectConfig := filepath.Join(app.GetConfigPath("mutagen.yml"))
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
		util.Success("Using mutagen config file %s", configFile)
	}

	util.Debug("Terminating mutagen sync if session already exists")
	err := TerminateMutagenSync(app)
	if err != nil {
		return err
	}
	container, err := dockerutil.FindContainerByName(fmt.Sprintf("ddev-%s-web", app.Name))
	if err != nil {
		return err
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
	defer close(flushErr)
	defer close(stopGoroutine)

	go func() {
		err = app.MutagenSyncFlush()
		flushErr <- err
		return
	}()

	// In tests or other non-interactive environments we don't need to show the
	// mutagen sync monitor output (and it fills up the test logs)

	if term.IsTerminal(int(os.Stderr.Fd())) {
		go func() {
			cmd := osexec.Command(globalconfig.GetMutagenPath(), "sync", "monitor", syncName)
			stdout, _ := cmd.StdoutPipe()
			err = cmd.Start()
			buf := bufio.NewReader(stdout)
			for {
				select {
				case <-stopGoroutine:
					return
				default:
					line, err := buf.ReadBytes('\r')
					if err != nil {
						return
					}
					l := string(line)
					if strings.HasPrefix(l, "Status:") {
						_, _ = fmt.Fprintf(os.Stderr, "%s", l)
					}
				}
			}
		}()
	}

	for {
		select {
		// Complete when the MutagenSyncFlush() completes
		case err = <-flushErr:
			return err

		// Show dots when it seems like nothing is happening
		case <-time.After(1 * time.Second):
			_, _ = fmt.Fprintf(os.Stderr, ".")
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
	if strings.Contains(longResult, "problems") || strings.Contains(longResult, "Conflicts") || strings.Contains(longResult, "error") || strings.Contains(shortResult, "Halted") {
		util.Error("mutagen sync session '%s' is not working correctly: %s", syncName, longResult)
		return "failing", shortResult, longResult, errors.Errorf("mutagen sync session '%s' is not working correctly, use 'mutagen sync list %s' for details", syncName, syncName)
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
	if app.MutagenEnabled || app.MutagenEnabledGlobal {
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
		util.Success("Flushed mutagen sync session '%s'", syncName)
	}
	return nil
}

// MutagenSyncExists detects whether the named sync exists
func MutagenSyncExists(app *DdevApp) bool {
	syncName := MutagenSyncName(app.Name)

	if !fileutil.FileExists(globalconfig.GetMutagenPath()) {
		return false
	}
	_, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "list", syncName)
	return err == nil
}

// DownloadMutagen gets the mutagen binary and related and puts it into
// ~/.ddev/.bin
func DownloadMutagen() error {
	StopMutagenDaemon()
	flavor := runtime.GOOS + "_" + runtime.GOARCH
	globalMutagenDir := filepath.Dir(globalconfig.GetMutagenPath())
	destFile := filepath.Join(globalMutagenDir, "mutagen.tgz")
	mutagenURL := fmt.Sprintf("https://github.com/mutagen-io/mutagen/releases/download/v%s/mutagen_%s_v%s.tar.gz", nodeps.RequiredMutagenVersion, flavor, nodeps.RequiredMutagenVersion)
	output.UserOut.Printf("Downloading %s", mutagenURL)
	_ = os.MkdirAll(globalMutagenDir, 0777)
	err := util.DownloadFile(destFile, mutagenURL, true)
	if err != nil {
		return err
	}
	err = archive.Untar(destFile, globalMutagenDir, "")
	_ = os.Remove(destFile)
	if err != nil {
		return err
	}
	err = os.Chmod(globalconfig.GetMutagenPath(), 0755)
	// Stop daemon in case it was already running somewhere else
	StopMutagenDaemon()
	return err
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
	if !app.MutagenEnabled || app.MutagenEnabledGlobal {
		return nil
	}
	curVersion, err := version.GetMutagenVersion()
	if err != nil || curVersion != nodeps.RequiredMutagenVersion {
		err = DownloadMutagen()
		if err != nil {
			return err
		}
	}
	return nil
}

// MutagenReset stops (with flush), removes the docker volume, starts again (with flush)
func MutagenReset(app *DdevApp) error {
	if app.MutagenEnabled || app.MutagenEnabledGlobal {
		err := app.Stop(false, false)
		if err != nil {
			return errors.Errorf("Failed to stop project %s: %v", app.Name, err)
		}
		err = dockerutil.RemoveVolume(GetMutagenVolumeName(app))
		if err != nil {
			return err
		}
		output.UserOut.Printf("Removed docker volume %s", GetMutagenVolumeName(app))
		err = app.Start()
		if err != nil {
			return err
		}
	}
	return nil
}

// GetMutagenVolumeName returns the name for the mutagen docker volume
func GetMutagenVolumeName(app *DdevApp) string {
	return app.Name + "_" + "project_mutagen"
}

// MutagenMonitor shows the ouput of `mutagen sync monitor <syncName>`
func MutagenMonitor(app *DdevApp) {
	syncName := MutagenSyncName(app.Name)

	// This doesn't actually return; you have to <ctrl-c> to end it
	c := osexec.Command(globalconfig.GetMutagenPath(), "sync", "monitor", syncName)
	c.Stdout = os.Stdout
	_ = c.Run()
}
