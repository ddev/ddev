package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
)

// skipNestedProject returns the registered project appRoot is nested inside, so a
// .ddev/config.yaml arriving with a dependency or submodule can't take over the project
// it sits in. With nothing registered above it, appRoot is all DDEV has to go on.
func skipNestedProject(appRoot string) string {
	if isRegisteredProject(appRoot) {
		return appRoot
	}
	for dir := filepath.Dir(appRoot); filepath.Dir(dir) != dir; dir = filepath.Dir(dir) {
		if isRegisteredProject(dir) {
			if useNestedProject(appRoot, dir) {
				break
			}
			return dir
		}
	}
	return appRoot
}

// isRegisteredProject reports whether dir is the root of a project in `ddev list`.
func isRegisteredProject(dir string) bool {
	for _, p := range globalconfig.DdevProjectList {
		if p != nil && filepath.Clean(p.AppRoot) == filepath.Clean(dir) {
			return true
		}
	}
	return false
}

// nestedAnswer remembers the prompt answer, since a command can resolve the project
// more than once.
var nestedAnswer *bool

// useNestedProject reports whether to use the unregistered project in nested rather than
// the registered project in outer. Only bare `ddev start` offers that choice, and it runs
// before flags are parsed, so os.Args is what there is to go on.
func useNestedProject(nested, outer string) bool {
	command := ""
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	if len(os.Args) == 2 && command == "start" {
		if nestedAnswer == nil {
			answer := util.ConfirmTo(fmt.Sprintf("The DDEV project in %q is nested inside the project in %q.\nUse the nested project?", nested, outer), false)
			nestedAnswer = &answer
		}
		return *nestedAnswer
	}
	// `ddev config` configures the directory it's in, so the warning would contradict it,
	// even though the custom commands of the project around it are still loaded.
	if !output.JSONOutput && command != "config" {
		util.WarningOnce(util.ColorizeText("Using the DDEV project in %[2]q; the nested project in %[1]q is not registered.\nTo use the nested project instead, run `ddev start` here and confirm.", "magenta"), nested, outer)
	}
	return false
}
