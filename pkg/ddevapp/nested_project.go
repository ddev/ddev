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

// BareStartTokens are the os.Args[1] values that could name `ddev start` - its own name and
// its aliases (cobra.Command.Aliases on StartCmd). cmd/ddev/cmd overwrites this from
// StartCmd's real Name() and Aliases once Cobra's command tree is built; this hardcoded
// default only covers callers that never wire that up (e.g. package tests) and the sliver
// of program startup before it's built (see IsBareStartInvocationKnown).
var BareStartTokens = map[string]bool{"start": true, "add": true}

// IsBareStartInvocation reports whether the current CLI invocation is `ddev start` (or an
// alias for it) with no explicit target - the only case useNestedProject() offers the
// opt-in prompt for. cmd/ddev/cmd overrides this once Cobra's command tree is built, since
// only Cobra knows start's real flags and aliases; this default is a plain os.Args check
// for callers that never wire up that override (e.g. package tests).
var IsBareStartInvocation = func() bool {
	return len(os.Args) == 2 && BareStartTokens[os.Args[1]]
}

// IsBareStartInvocationKnown reports whether IsBareStartInvocation can be trusted to tell a
// `ddev start`/`ddev add` with flags apart from a truly bare one. A couple of packages
// resolve the active project from their own init(), before cmd/ddev/cmd has finished
// building Cobra's command tree and overridden IsBareStartInvocation; until then, its
// default os.Args check can say "false" for what's actually about to prompt as bare start.
var IsBareStartInvocationKnown = false

// useNestedProject reports whether to use the unregistered project in nested rather than
// the registered project in outer. Only bare `ddev start` offers that choice.
func useNestedProject(nested, outer string) bool {
	command := ""
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	if IsBareStartInvocation() {
		if nestedAnswer == nil {
			answer := util.ConfirmTo(fmt.Sprintf("The DDEV project in %q is nested inside the project in %q.\nUse the nested project?", nested, outer), false)
			nestedAnswer = &answer
		}
		return *nestedAnswer
	}
	// `ddev config` configures the directory it's in, so the warning would contradict it,
	// even though the custom commands of the project around it are still loaded. Likewise,
	// a "false" here for `start`/`add` isn't trustworthy until IsBareStartInvocationKnown -
	// warning would contradict the prompt that's about to follow once Cobra actually
	// dispatches the command.
	uncertain := BareStartTokens[command] && !IsBareStartInvocationKnown
	if !output.JSONOutput && command != "config" && !uncertain {
		util.WarningOnce(util.ColorizeText("Using the DDEV project in %[2]q; the nested project in %[1]q is not registered.\nTo use the nested project instead, run `ddev start` here and confirm.", "magenta"), nested, outer)
	}
	return false
}
