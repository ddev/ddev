package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/util"
	"github.com/pkg/errors"
	"strings"
)

func SetMutagenVolumeOwnership(app *DdevApp) error {
	// Make sure that if we have a volume mount it's got proper ownership
	uidStr, gidStr, _ := util.GetContainerUIDGid()
	_, _, err := app.Exec(
		&ExecOpts{
			Dir: "/tmp",
			Cmd: fmt.Sprintf("sudo chown -R %s:%s /var/www/html", uidStr, gidStr),
		})
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

	err := TerminateMutagenSync(app)
	if err != nil {
		return err
	}
	_, err = exec.RunCommand(bashPath, []string{"-c", fmt.Sprintf(`mutagen sync create "%s" docker://ddev-%s-web/var/www/html --sync-mode=two-way-resolved --name=%s >/dev/null`, app.AppRoot, app.Name, syncName)})
	if err != nil {
		return err
	}
	err = app.MutagenSyncFlush()
	if err != nil {
		return err
	}
	return nil
}

// CheckMutagenErrors checks to see if there is an error case in mutagen
// We can't do a flush in that case.
func CheckMutagenErrors(app *DdevApp) (string, error) {
	syncName := MutagenSyncName(app.Name)
	bashPath := util.FindBashPath()

	out, err := exec.RunCommand(bashPath, []string{"-c", fmt.Sprintf(`mutagen sync list %s`, syncName)})
	if err != nil {
		return out, err
	}

	if strings.Contains(out, "problems") || strings.Contains(out, "Conflicts") || strings.Contains(out, "error") {
		util.Error("mutagen sync %s is not working correctly: %s", syncName, out)
		return out, errors.Errorf("mutagen sync %s is not working correctly, use 'mutagen sync list %s' for details", syncName, syncName)
	}
	return out, nil
}

// MutagenSyncFlush performs a mutagen sync flush, waits for result, and checks for errors
func (app *DdevApp) MutagenSyncFlush() error {
	if app.MutagenEnabled || app.MutagenEnabledGlobal {
		_, err := CheckMutagenErrors(app)
		if err != nil {
			return err
		}
		bashPath := util.FindBashPath()
		syncName := MutagenSyncName(app.Name)
		if !MutagenSyncExists(app) {
			return errors.Errorf("Mutagen sync %s does not exist", syncName)
		}
		out, err := CheckMutagenErrors(app)
		if err != nil {
			return errors.Errorf("Mutagen sync %s is in error state: %s (%v)", syncName, out, err)
		}

		_, err = exec.RunCommand(bashPath, []string{"-c", fmt.Sprintf("mutagen sync flush %s", syncName)})
		if err != nil {
			return err
		}

		_, err = CheckMutagenErrors(app)
		if err != nil {
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
