package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/pkg/errors"
	"strings"
)

func SetMutagenVolumeOwnership(app *DdevApp) error {
	// Make sure that if we have a volume mount it's got proper ownership
	uidStr, gidStr, _ := util.GetContainerUIDGid()
	output.UserOut.Printf("chowning mutagen docker volume for user %s", uidStr)
	_, _, err := app.Exec(
		&ExecOpts{
			Dir: "/tmp",
			Cmd: fmt.Sprintf("sudo chown -R %s:%s /var/www/html", uidStr, gidStr),
		})
	output.UserOut.Printf("done chowning mutagen docker volume, result=%v", err)
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
			_, err := exec.RunCommand(bashPath, []string{"-c", fmt.Sprintf("mutagen sync terminate %s", syncName)})
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

	output.UserOut.Printf("Terminating mutagen sync if session already exists")
	err := TerminateMutagenSync(app)
	if err != nil {
		return err
	}
	output.UserOut.Printf("Starting mutagen sync %s", syncName)
	_, err = exec.RunCommand(bashPath, []string{"-c", fmt.Sprintf(`mutagen sync create "%s" docker://ddev-%s-web/var/www/html --sync-mode=two-way-resolved --name=%s >/dev/null`, app.AppRoot, app.Name, syncName)})
	if err != nil {
		return err
	}
	output.UserOut.Printf("Flushing mutagen sync %s", syncName)
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

	longResult, err = exec.ExecCommand(bashPath, "-c", fmt.Sprintf(`mutagen sync list %s`, syncName))
	shortResult = parseMutagenStatusLine(longResult)
	if err != nil {
		return false, shortResult, longResult, err
	}

	// We're going to assume that if it's applying changes things are still OK,
	// even though there may be a whole list of problems.
	if strings.Contains(shortResult, "Applying changes") || strings.Contains(shortResult, "Staging files on") || strings.Contains(shortResult, "Scanning files") || strings.Contains(shortResult, "Watching for changes") {
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

		_, err = exec.RunCommand(bashPath, []string{"-c", fmt.Sprintf("mutagen sync flush %s", syncName)})
		if err != nil {
			return err
		}

		status, _, long, err = app.MutagenStatus()
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

	_, err := exec.RunCommand(bashPath, []string{"-c", fmt.Sprintf("mutagen sync list %s >/dev/null 2>&1", syncName)})
	return err == nil
}
