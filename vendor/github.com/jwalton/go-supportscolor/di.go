package supportscolor

import (
	"os"
	"runtime"

	hasflag "github.com/jwalton/go-supportscolor/pkg/hasFlag"
	"golang.org/x/term"
)

type environment interface {
	LookupEnv(name string) (string, bool)
	Getenv(name string) string
	HasFlag(name string) bool
	IsTerminal(fd int) bool
	getWindowsVersion() (majorVersion, minorVersion, buildNumber uint32)
	osEnableColor() bool
	getGOOS() string
}

type defaultEnvironmentType struct{}

func (*defaultEnvironmentType) LookupEnv(name string) (string, bool) {
	return os.LookupEnv(name)
}

func (*defaultEnvironmentType) Getenv(name string) string {
	return os.Getenv(name)
}

func (*defaultEnvironmentType) HasFlag(flag string) bool {
	return hasflag.HasFlag(flag)
}

func (*defaultEnvironmentType) IsTerminal(fd int) bool {
	// TODO: Replace with github.com/mattn/go-isatty?
	return term.IsTerminal(int(fd))
}

func (*defaultEnvironmentType) getWindowsVersion() (majorVersion, minorVersion, buildNumber uint32) {
	return getWindowsVersion()
}

func (*defaultEnvironmentType) osEnableColor() bool {
	return enableColor()
}

func (*defaultEnvironmentType) getGOOS() string {
	return runtime.GOOS
}

var defaultEnvironment = defaultEnvironmentType{}
