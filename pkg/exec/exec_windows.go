package exec

import (
	"os"
	"strings"
	"syscall"

	"github.com/drud/ddev/pkg/util"
	"golang.org/x/sys/windows"
)

// RunCommandWithRootRights runs a command with sudo on Unix like systems or
// with elevated rights on Windows.
func RunCommandWithRootRights(command string, args []string) (string, error) {
	if HasRootRights() {
		return RunCommand(command, args)
	}

	cwd, _ := os.Getwd()
	joinedArgs := strings.Join(args, " ")

	verbPtr, _ := syscall.UTF16PtrFromString("runas")
	exePtr, _ := syscall.UTF16PtrFromString(command)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(joinedArgs)

	util.Success("Running '%s %s'", command, joinedArgs)

	if isDdev(command) {
		eventNamePtr, _ := syscall.UTF16PtrFromString("Local\\DdevSubprocessTerminated")
		event, _ := windows.CreateEvent(nil, 0, 0, eventNamePtr)

		windows.CreateFileMapping()
		windows.MapViewOfFile()
	}

	output := ""
	err := windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, windows.SW_SHOWNORMAL)

	if isDdev(command) {
		windows.WaitForSingleObject(event, 10000)
	}

	return output, err
}

// HasRootRights returns true if the current context is root on Unix like
// systems or elevated on Windows.
func HasRootRights() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

func isDdev(command string) bool {
	self, _ := os.Executable()
	return command == self
}
