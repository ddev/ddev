package ddevapp

import (
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/exec"
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
		bashPath := util.FindBashPath()
		syncName := MutagenSyncName(app.Name)
		if MutagenSyncExists(app) {
			_, err := exec.RunHostCommand(bashPath, "-c", fmt.Sprintf("mutagen sync terminate %s", syncName))
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

// CreateMutagenSync creates a sync (after making sure it doesn't exist)
// It detects problems with the sync and errors if there are problems
// and returns the output of `mutagen sync list <syncname>` along with error info
func CreateMutagenSync(app *DdevApp) error {
	syncName := MutagenSyncName(app.Name)
	bashPath := util.FindBashPath()

	util.Debug("Terminating mutagen sync if session already exists")
	err := TerminateMutagenSync(app)
	if err != nil {
		return err
	}
	util.Debug("Starting mutagen sync %s", syncName)
	_, err = exec.RunHostCommand(bashPath, "-c", fmt.Sprintf(`mutagen sync create "%s" docker://ddev-%s-web/var/www/html --sync-mode=two-way-resolved --name=%s >/dev/null`, app.AppRoot, app.Name, syncName))
	if err != nil {
		return err
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
	bashPath := util.FindBashPath()

	longResult, err = exec.RunHostCommand(bashPath, "-c", fmt.Sprintf(`mutagen sync list %s`, syncName))
	shortResult = parseMutagenStatusLine(longResult)
	if err != nil {
		return false, shortResult, longResult, err
	}

	// We're going to assume that if it's applying changes things are still OK,
	// even though there may be a whole list of problems.
	if strings.Contains(shortResult, "Applying changes") || strings.Contains(shortResult, "Staging files on") || strings.Contains(shortResult, "Reconciling changes") || strings.Contains(shortResult, "Scanning files") || strings.Contains(shortResult, "Watching for changes") {
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
	statusLine = statusLine[:statusLineEnd]
	pieces := strings.Split(statusLine, ":")
	if len(pieces) < 2 {
		return ""
	}
	return strings.Trim(pieces[1], " \n\r")
}

// MutagenSyncFlush performs a mutagen sync flush, waits for result, and checks for errors
func (app *DdevApp) MutagenSyncFlush() error {
	if app.MutagenEnabled || app.MutagenEnabledGlobal {
		bashPath := util.FindBashPath()
		syncName := MutagenSyncName(app.Name)
		if !MutagenSyncExists(app) {
			return errors.Errorf("Mutagen sync %s does not exist", syncName)
		}
		status, _, long, err := app.MutagenStatus()
		if !status || err != nil {
			return errors.Errorf("Mutagen sync %s is in error state: %s (%v)", syncName, long, err)
		}

		_, err = exec.RunHostCommand(bashPath, "-c", fmt.Sprintf("mutagen sync flush %s", syncName))
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
	bashPath := util.FindBashPath()
	syncName := MutagenSyncName(app.Name)

	_, err := exec.RunHostCommand(bashPath, "-c", fmt.Sprintf("mutagen sync list %s >/dev/null 2>&1", syncName))
	return err == nil
}

// DownloadMutagen gets the mutagen binary and related and puts it into
// ~/.ddev/.bin
func DownloadMutagen() error {
	flavor := runtime.GOOS + "_" + runtime.GOARCH
	globalMutagenDir := filepath.Join(globalconfig.GetGlobalDdevDir(), ".bin")
	destFile := filepath.Join(globalMutagenDir, "mutagen.tgz")
	mutagenURL := fmt.Sprintf("https://github.com/mutagen-io/mutagen/releases/download/%s/mutagen_%s_v%s.tar.gz", nodeps.RequiredMutagenVersion, flavor, nodeps.RequiredMutagenVersion)
	output.UserOut.Printf("Downloading %s", mutagenURL)
	_ = os.MkdirAll(globalMutagenDir, 0777)
	err := util.DownloadFile(destFile, mutagenURL, true)
	if err != nil {
		return err
	}
	err = archive.Untar(destFile, globalMutagenDir, "")
	return err
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
