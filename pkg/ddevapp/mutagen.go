package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/util"
	"github.com/pkg/errors"
	"strings"
)

// MutagenSyncName transforms a projectname string into
// an acceptable mutagen sync "name"
// See restrictions on sync name at https://mutagen.io/documentation/introduction/names-labels-identifiers
// The input must be a valid DNS name (valid ddev project name)
func MutagenSyncName(name string) string {
	return strings.ReplaceAll(name, ".", "")
}

// TerminateMutagen terminates the mutagen sync
// It is not an error if the sync session does not exist, and stderr is buried
func TerminateMutagen(app *DdevApp) {
	if app.MutagenEnabled || app.MutagenEnabledGlobal {
		bashPath := util.FindBashPath()
		syncName := MutagenSyncName(app.Name)
		_, _ = exec.RunCommand(bashPath, []string{"-c", fmt.Sprintf("mutagen sync terminate %s 2>/dev/null", syncName)})
	}
}

// SyncAndTerminateMutagen syncs and terminates the mutagen sync
func SyncAndTerminateMutagen(app *DdevApp) error {
	if app.MutagenEnabled || app.MutagenEnabledGlobal {
		bashPath := util.FindBashPath()
		syncName := MutagenSyncName(app.Name)
		_, err := exec.RunCommand(bashPath, []string{"-c", fmt.Sprintf("mutagen sync flush %s", syncName)})
		if err != nil {
			return err
		}
		TerminateMutagen(app)
	}
	return nil
}

// CreateMutagenSync creates a sync (after making sure it doesn't exist)
// It detects problems with the sync and errors if there are problems
// and returns the output of `mutagen sync list <syncname>` along with error info
func CreateMutagenSync(app *DdevApp) (string, error) {
	syncName := MutagenSyncName(app.Name)
	bashPath := util.FindBashPath()

	TerminateMutagen(app)
	_, err := exec.RunCommand(bashPath, []string{"-c", fmt.Sprintf(`mutagen sync create "%s" docker://ddev-%s-web/var/www/html --sync-mode=two-way-resolved --symlink-mode=posix-raw --name=%s >/dev/null && mutagen sync flush %s >/dev/null`, app.Docroot, app.Name, syncName, syncName)})
	if err != nil {
		return "", err
	}
	out, err := exec.RunCommand(bashPath, []string{"-c", fmt.Sprintf(`mutagen sync list %s`, syncName)})
	if err != nil {
		return out, err
	}

	if strings.Contains(out, "problems") || strings.Contains(out, "Conflicts") {
		util.Error("mutagen sync %s is not working correctly: %s", out)
		return out, errors.Errorf("mutagen sync %s is not working correctly, use 'mutagen sync list %s' for details", syncName, syncName)
	}
	return out, nil
}
