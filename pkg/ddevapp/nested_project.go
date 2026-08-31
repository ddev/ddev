package ddevapp

import (
	"fmt"
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

// nestedProjectChoice holds what one run decided about an unregistered nested project.
// Loading custom commands resolves the project before Cobra can say which command is
// running, so the app roots are recorded then and reported once that's known.
type nestedProjectChoice struct {
	askUser              bool
	userSaidYes          *bool
	skippedNestedAppRoot string
	registeredAppRoot    string
	alreadyWarned        bool
}

var nestedProjectState nestedProjectChoice

// PromptToUseNestedProject defaults to the registered project around the nested one, and
// with skipConfirmation keeps it unasked, warning instead since nothing later will.
func PromptToUseNestedProject(skipConfirmation bool) {
	nestedProjectState.askUser = !skipConfirmation
	if skipConfirmation {
		ShowNestedProjectWarning()
	}
}

// ShowNestedProjectWarning reports a nested project passed over in favor of the registered
// one around it, once per run.
func ShowNestedProjectWarning() {
	if nestedProjectState.skippedNestedAppRoot == "" || output.JSONOutput {
		return
	}
	util.WarningWithColor("magenta", "Using the DDEV project in %[2]q; the nested project in %[1]q is not registered.\nTo use the nested project instead, run `ddev start` here and confirm.", nestedProjectState.skippedNestedAppRoot, nestedProjectState.registeredAppRoot)
	nestedProjectState.skippedNestedAppRoot = ""
	nestedProjectState.alreadyWarned = true
}

// ResetNestedProjectState forgets the choice, for tests exercising several invocations.
func ResetNestedProjectState() {
	nestedProjectState = nestedProjectChoice{}
}

// useNestedProject reports whether to use the unregistered project in nested rather than
// the registered project in outer.
func useNestedProject(nested, outer string) bool {
	if nestedProjectState.askUser {
		if nestedProjectState.userSaidYes == nil {
			answer := util.ConfirmTo(fmt.Sprintf("The DDEV project in %q is nested inside the project in %q.\nUse the nested project?", nested, outer), false)
			nestedProjectState.userSaidYes = &answer
		}
		return *nestedProjectState.userSaidYes
	}
	if !nestedProjectState.alreadyWarned {
		nestedProjectState.skippedNestedAppRoot, nestedProjectState.registeredAppRoot = nested, outer
	}
	return false
}
