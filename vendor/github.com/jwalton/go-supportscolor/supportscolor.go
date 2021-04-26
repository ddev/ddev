// Package supportscolor detects whether a terminal supports color, and enables ANSI color support in recent Windows 10 builds.
//
// This is a port of the Node.js package supports-color (https://github.com/chalk/supports-color) by
// Sindre Sorhus and Josh Junon.
//
// Returns a `supportscolor.Support` with a `Stdout()` and `Stderr()` function for
// testing either stream.  Note that on recent Windows 10 machines, these
// functions will also set the `ENABLE_VIRTUAL_TERMINAL_PROCESSING` console mode
// if required, which will enable support for normal ANSI escape codes on stdout
// and stderr.
//
// The `Stdout()`/`Stderr()` objects specify a level of support for color through
// a `.Level` property and a corresponding flag:
//
//     - `.Level = None` and `.SupportsColor = false`: No color support
//     - `.Level = Basic` and `.SupportsColor = true`: Basic color support (16 colors)
//     - `.Level = Ansi256` and `.Has256 = true`: 256 color support
//     - `.Level = Ansi16m` and `.Has16m = true`: True color support (16 million colors)
//
// Additionally, `supportscolor` exposes the `.SupportsColor()` function that
// takes an arbitrary file descriptor (e.g. `os.Stdout.Fd()`) and options, and will
// (re-)evaluate color support for an arbitrary stream.
//
// For example, `supportscolor.Stdout()` is the equivalent of `supportscolor.SupportsColor(os.Stdout.Fd())`.
//
// Available options are:
//
// `supportscolor.IsTTYOption(isTTY bool)` - Force whether the given file
// should be considered a TTY or not. If this not specified, TTY status will
// be detected automatically via `term.IsTerminal()`.
//
// `supportscolor.SniffFlagsOption(sniffFlags bool)` - By default it is `true`,
// which instructs `SupportsColor()` to sniff `os.Args` for the multitude of
// `--color` flags (see Info section in README.md). If `false`, then `os.Args`
// is not considered when determining color support.
//
package supportscolor

import (
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Support represents the color support available.
//
// Level will be the supported ColorLevel.  SupportsColor will be true if the
// terminal supports basic 16 color ANSI color escape codes.  Has256 will be
// true if the terminal supports ANSI 256 color, and Has16m will be true if the
// terminal supports true color.
//
type Support struct {
	Level         ColorLevel
	SupportsColor bool
	Has256        bool
	Has16m        bool
}

func checkForceColorFlags(env environment) *ColorLevel {
	var flagForceColor ColorLevel = None
	var flagForceColorPreset bool = false
	// TODO: It would be very nice if `HasFlag` supported `--color false`.
	if env.HasFlag("no-color") ||
		env.HasFlag("no-colors") ||
		env.HasFlag("color=false") ||
		env.HasFlag("color=never") {
		flagForceColor = None
		flagForceColorPreset = true
	} else if env.HasFlag("color") ||
		env.HasFlag("colors") ||
		env.HasFlag("color=true") ||
		env.HasFlag("color=always") {
		flagForceColor = Basic
		flagForceColorPreset = true
	}

	if flagForceColorPreset {
		return &flagForceColor
	}
	return nil
}

func checkForceColorEnv(env environment) *ColorLevel {
	forceColor, present := env.LookupEnv("FORCE_COLOR")

	if present {
		if forceColor == "true" || forceColor == "" {
			result := Basic
			return &result
		}

		if forceColor == "false" {
			result := None
			return &result
		}

		forceColorInt, err := strconv.ParseInt(forceColor, 10, 8)
		if err == nil {
			var result ColorLevel
			if forceColorInt <= 0 {
				result = None
			} else if forceColorInt >= 3 {
				result = Ansi16m
			} else {
				result = ColorLevel(forceColorInt)
			}
			return &result
		}
	}

	return nil
}

func translateLevel(level ColorLevel) Support {
	return Support{
		Level:         level,
		SupportsColor: level >= 1,
		Has256:        level >= 2,
		Has16m:        level >= 3,
	}
}

func supportsColor(config *configuration) ColorLevel {
	env := config.env

	// TODO: We don't have to call `checkForceColorFlags` multiple times,
	// as it's not common practice to modify `os.Args`.  We can call it once
	// and cache the result, in say `init()`.
	flagForceColor := checkForceColorFlags(env)
	noFlagForceColor := checkForceColorEnv(env)

	// Env preferences should override flags
	if noFlagForceColor != nil {
		flagForceColor = noFlagForceColor
	}

	var forceColor *ColorLevel
	if config.sniffFlags {
		forceColor = flagForceColor
	} else {
		forceColor = noFlagForceColor
	}

	if forceColor != nil && *forceColor == None {
		return None
	}

	if config.sniffFlags {
		if env.HasFlag("color=16m") ||
			env.HasFlag("color=full") ||
			env.HasFlag("color=truecolor") {
			env.osEnableColor()
			return Ansi16m
		}

		if env.HasFlag("color=256") {
			env.osEnableColor()
			return Ansi256
		}
	}

	if !config.isTTY && forceColor == nil {
		return None
	}

	min := None
	if forceColor != nil {
		min = *forceColor
	}

	term := env.Getenv("TERM")
	if term == "dumb" {
		return min
	}

	osColorEnabled := env.osEnableColor()
	if (!osColorEnabled) && forceColor == nil {
		return None
	}

	if env.getGOOS() == "windows" {
		// If we couldn't get windows to enable color, return basic.
		if !osColorEnabled {
			return None
		}

		// Windows 10 build 10586 is the first Windows release that supports 256 colors.
		// Windows 10 build 14931 is the first release that supports 16m/True color.
		major, minor, build := env.getWindowsVersion()
		if (major == 10 && minor >= 1) || major > 10 {
			// Optimistically hope that future versions of windows won't backslide.
			return Ansi16m
		} else if major >= 10 && build >= 14931 {
			return Ansi16m
		} else if major >= 10 && build >= 10586 {
			return Ansi256
		}

		// We should be able to return Basic here - if the terminal doesn't support
		// basic ANSI escape codes, we should have gotten false from `osEnableColor()`
		// because we should have gotten an error when we tried to set
		// ENABLE_VIRTUAL_TERMINAL_PROCESSING.
		// TODO: Make sure this is really true on an old version of Windows.
		return Basic
	}

	if _, ci := env.LookupEnv("CI"); ci {
		var ciEnvNames = []string{"TRAVIS", "CIRCLECI", "APPVEYOR", "GITLAB_CI", "GITHUB_ACTIONS", "BUILDKITE", "DRONE"}
		for _, ciEnvName := range ciEnvNames {
			_, exists := env.LookupEnv(ciEnvName)
			if exists {
				return Basic
			}
		}

		if env.Getenv("CI_NAME") == "codeship" {
			return Basic
		}

		return min
	}

	if teamCityVersion, isTeamCity := env.LookupEnv("TEAMCITY_VERSION"); isTeamCity {
		versionRegex := regexp.MustCompile(`^(9\.(0*[1-9]\d*)\.|\d{2,}\.)`)
		if versionRegex.MatchString(teamCityVersion) {
			return Basic
		}
		return None
	}

	if env.Getenv("COLORTERM") == "truecolor" {
		return Ansi16m
	}

	termProgram, termProgramPreset := env.LookupEnv("TERM_PROGRAM")
	if termProgramPreset {
		switch termProgram {
		case "iTerm.app":
			termProgramVersion := strings.Split(env.Getenv("TERM_PROGRAM_VERSION"), ".")
			version, err := strconv.ParseInt(termProgramVersion[0], 10, 64)
			if err == nil && version >= 3 {
				return Ansi16m
			}
			return Ansi256
		case "Apple_Terminal":
			return Ansi256

		default:
			// No default
		}
	}

	var term256Regex = regexp.MustCompile("(?i)-256(color)?$")
	if term256Regex.MatchString(term) {
		return Ansi256
	}

	var termBasicRegex = regexp.MustCompile("(?i)^screen|^xterm|^vt100|^vt220|^rxvt|color|ansi|cygwin|linux")

	if termBasicRegex.MatchString(term) {
		return Basic
	}

	if _, colorTerm := env.LookupEnv("COLORTERM"); colorTerm {
		return Basic
	}

	return min
}

type configuration struct {
	isTTY      bool
	forceIsTTY bool
	sniffFlags bool
	env        environment
}

// Option is the type for an option which can be passed to SupportsColor().
type Option func(*configuration)

// IsTTYOption is an option which can be passed to `SupportsColor` to force
// whether the given file should be considered a TTY or not.  If this not
// specified, TTY status will be detected automatically via `term.IsTerminal()`.
func IsTTYOption(isTTY bool) Option {
	return func(config *configuration) {
		config.forceIsTTY = true
		config.isTTY = isTTY
	}
}

func setEnvironment(env environment) Option {
	return func(config *configuration) {
		config.env = env
	}
}

// SniffFlagsOption can be passed to SupportsColor to enable or disable checking
// command line flags to force supporting color.  If set true (the default), then
// the following flags will disable color support:
//
//     --no-color
//     --no-colors
//     --color=false
//     --color=never
//
// And the following will force color support
//
//     --colors
//     --color=true
//     --color=always
//     --color=256       // Ansi 256 color mode
//     --color=16m       // 16.7 million color support
//     --color=full      // 16.7 million color support
//     --color=truecolor // 16.7 million color support
//
func SniffFlagsOption(sniffFlags bool) Option {
	return func(config *configuration) {
		config.sniffFlags = sniffFlags
	}
}

// SupportsColor returns color support information for the given file handle.
func SupportsColor(fd uintptr, options ...Option) Support {
	config := configuration{sniffFlags: true, env: &defaultEnvironment}
	for _, opt := range options {
		opt(&config)
	}

	if !config.forceIsTTY {
		config.isTTY = config.env.IsTerminal(int(fd))
	}

	level := supportsColor(&config)
	return translateLevel(level)
}

var stdout *Support

// Stdout returns color support information for os.Stdout.
func Stdout() Support {
	if stdout == nil {
		result := SupportsColor(os.Stdout.Fd())
		stdout = &result
	}
	return *stdout
}

var stderr *Support

// Stderr returns color support information for os.Stderr.
func Stderr() Support {
	if stderr == nil {
		result := SupportsColor(os.Stderr.Fd())
		stderr = &result
	}
	return *stderr
}
