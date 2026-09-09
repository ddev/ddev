package util

import (
	"os"
	"slices"
)

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

// TerminalEnvVars are the variables besides TERM that TerminalExecEnv forwards
// from the host. Terminals set them to say which program they are, and
// libraries like supports-hyperlinks read them to decide whether to emit OSC 8
// hyperlinks, as output.HasTermHyperlinks does for DDEV itself.
// See https://github.com/chalk/supports-hyperlinks/blob/main/index.js
var TerminalEnvVars = []string{
	"CI",
	"COLORTERM",
	"FORCE_HYPERLINK",
	"KONSOLE_VERSION",
	"TERM_PROGRAM",
	"TERM_PROGRAM_VERSION",
	"VTE_VERSION",
	"WT_SESSION",
}

// TerminalExecEnv adds the terminal environment variables of the host to
// existingEnv, for an interactive container session. Variables already in
// existingEnv win. A TERM the container has no terminfo entry for is replaced,
// because forwarding it would leave the shell with a TERM it cannot resolve.
func TerminalExecEnv(existingEnv []string) []string {
	hostTerm := os.Getenv("TERM")
	if hostTerm == "" {
		return existingEnv
	}
	if !slices.Contains(ContainerTerminfoEntries, hostTerm) {
		hostTerm = "xterm-256color"
	}
	execEnv := []string{"TERM=" + hostTerm}
	for _, varName := range TerminalEnvVars {
		if value := os.Getenv(varName); value != "" {
			execEnv = append(execEnv, varName+"="+value)
		}
	}
	// existingEnv comes last, so that EnvToUniqueEnv keeps it over the host.
	execEnv = append(execEnv, existingEnv...)
	return EnvToUniqueEnv(&execEnv)
}
