package nodeps

import (
	"slices"
	"strings"
)

// fallbackTerm is used when the host TERM is not in ContainerTerminfoEntries.
const fallbackTerm = "xterm-256color"

// ContainerTerminfoEntries is the terminfo database that ncurses-base installs
// in the DDEV images. TestContainerTerminfo checks it against the real images.
// Custom services use their own image and can have a smaller database.
var ContainerTerminfoEntries = []string{
	"Eterm", "Eterm-color", "ansi", "cons25", "cygwin", "dumb", "hurd", "linux",
	"mach", "mach-bold", "mach-color", "mach-gnu", "mach-gnu-color", "pcansi",
	"rxvt", "rxvt-basic", "rxvt-m", "rxvt-unicode", "rxvt-unicode-256color",
	"screen", "screen-256color", "screen-256color-bce", "screen-bce", "screen-s",
	"screen-w", "screen.xterm-256color", "sun", "tmux", "tmux-256color",
	"vt100", "vt102", "vt220", "vt52", "wsvt25", "wsvt25m",
	"xterm", "xterm-256color", "xterm-color", "xterm-debian", "xterm-mono",
	"xterm-r5", "xterm-r6", "xterm-vt220", "xterm-xfree86",
}

// TerminalExecEnv adds the terminal environment variables of the host to
// existingEnv, for an interactive container session. Variables already in
// existingEnv win. A TERM the container has no terminfo entry for is replaced,
// because forwarding it would leave the shell with a TERM it cannot resolve.
func TerminalExecEnv(existingEnv []string, hostTerm string, hostColorterm string) []string {
	if hostTerm == "" {
		return existingEnv
	}
	execEnv := slices.Clone(existingEnv)
	if !envHasKey(existingEnv, "TERM") {
		containerTerm := hostTerm
		if !ArrayContainsString(ContainerTerminfoEntries, hostTerm) {
			containerTerm = fallbackTerm
		}
		execEnv = append(execEnv, "TERM="+containerTerm)
	}
	if hostColorterm != "" && !envHasKey(existingEnv, "COLORTERM") {
		execEnv = append(execEnv, "COLORTERM="+hostColorterm)
	}
	return execEnv
}

// envHasKey reports whether env contains an entry for key, either as
// "key=value" or as bare "key".
func envHasKey(env []string, key string) bool {
	for _, entry := range env {
		entryKey, _, _ := strings.Cut(entry, "=")
		if entryKey == key {
			return true
		}
	}
	return false
}
