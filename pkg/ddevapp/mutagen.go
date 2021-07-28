package ddevapp

import (
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	if app.MutagenEnabled || app.MutagenEnabledGlobal {
		syncName := MutagenSyncName(app.Name)
		if MutagenSyncExists(app) {
			_, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "terminate", syncName)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// SyncAndTerminateMutagen syncs and terminates the mutagen sync
func SyncAndTerminateMutagen(app *DdevApp) error {
	if app.MutagenEnabled || app.MutagenEnabledGlobal {
		syncName := MutagenSyncName(app.Name)

		if !MutagenSyncExists(app) {
			return errors.Errorf("Sync %v does nto exist", syncName)
		}
		err := app.MutagenSyncFlush()
		if err != nil {
			util.Error("Error on mutagen sync flush %s: %v", syncName, err)
		}
		err = TerminateMutagenSync(app)
		if err != nil {
			return err
		}
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
	args := []string{"sync", "create", app.AppRoot, fmt.Sprintf("docker://ddev-%s-web/var/www/html", app.Name), "--no-global-configuration", "--name", syncName}
	if configFile != "" {
		args = append(args, fmt.Sprintf(`--configuration-file=%s`, configFile))
	}
	util.Debug("Creating mutagen sync: mutagen %v", args)
	out, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), args...)
	if err != nil {
		return fmt.Errorf("Failed to mutagen %v (%v), output=%s", args, err, out)
	}
	util.Debug("Flushing mutagen sync %s", syncName)
	err = app.MutagenSyncFlush()
	if err != nil {
		return err
	}
	return nil
}

// MutagenStatus checks to see if there is an error case in mutagen
// We don't want to do a flush yet in that case.
func (app *DdevApp) MutagenStatus() (status bool, shortResult string, longResult string, err error) {
	syncName := MutagenSyncName(app.Name)

	longResult, err = exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "list", syncName)
	shortResult = parseMutagenStatusLine(longResult)
	if err != nil {
		return false, shortResult, longResult, err
	}

	// We're going to assume that if it's applying changes things are still OK,
	// even though there may be a whole list of problems.
	if strings.Contains(shortResult, "Applying changes") || strings.Contains(shortResult, "Staging files on") || strings.Contains(shortResult, "Reconciling changes") || strings.Contains(shortResult, "Scanning files") || strings.Contains(shortResult, "Watching for changes") || strings.Contains(shortResult, "Saving archive") {
		return true, shortResult, longResult, nil
	}
	if strings.Contains(longResult, "problems") || strings.Contains(longResult, "Conflicts") || strings.Contains(longResult, "error") || strings.Contains(shortResult, "Halted") {
		util.Error("mutagen sync %s is not working correctly: %s", syncName, longResult)
		return false, shortResult, longResult, errors.Errorf("mutagen sync %s is not working correctly, use 'mutagen sync list %s' for details", syncName, syncName)
	}
	return true, shortResult, longResult, nil
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
			return errors.Errorf("Mutagen sync %s does not exist", syncName)
		}
		status, _, long, err := app.MutagenStatus()
		if !status || err != nil {
			return errors.Errorf("Mutagen sync %s is in error state: %s (%v)", syncName, long, err)
		}

		_, err = exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "flush", syncName)
		if err != nil {
			return err
		}

		status, _, _, err = app.MutagenStatus()
		if !status || err != nil {
			return err
		}
	}
	return nil
}

// MutagenSyncExists detects whether the named sync exists
func MutagenSyncExists(app *DdevApp) bool {
	syncName := MutagenSyncName(app.Name)

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
	_, _ = exec.RunHostCommand(globalconfig.GetMutagenPath(), "daemon", "stop")
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

// CheckMutagenVersion determines if the mutagen version of the host
//system meets the provided version constraint
func CheckMutagenVersion(versionConstraint string) error {
	currentVersion, err := version.GetMutagenVersion()
	if err != nil {
		return fmt.Errorf("no mutagen")
	}
	v, err := semver.NewVersion(currentVersion)
	if err != nil {
		return err
	}

	constraint, err := semver.NewConstraint(versionConstraint)
	if err != nil {
		return err
	}

	match, errs := constraint.Validate(v)
	if !match {
		if len(errs) <= 1 {
			return errs[0]
		}

		msgs := "\n"
		for _, err := range errs {
			msgs = fmt.Sprint(msgs, err, "\n")
		}
		return fmt.Errorf(msgs)
	}
	return nil
}
