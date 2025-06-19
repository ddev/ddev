package main

import (
	"fmt"
	"github.com/ddev/ddev/pkg/hostname"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

func main() {
	RootCmd.Execute()
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

var removeHostnameFlag bool
var removeInactiveFlag bool
var checkHostnameFlag bool

// RootCmd is the ddev_hostname command
var RootCmd = &cobra.Command{
	Use:   "hostname [hostname] [ip]",
	Short: "Manage your hostfile entries.",
	Example: `
ddev_hostname hostname junk.example.com 127.0.0.1
ddev_hostname hostname -r junk.example.com 127.0.0.1
ddev_hostname hostname --check junk.example.com 127.0.0.1
ddev_hostname hostname --remove-inactive
`,
	Long: `Manage your hostfile entries. Managing host names has security and usability
implications and requires elevated privileges. You may be asked for a password
to allow ddev_hostname to modify your hosts file. If you are connected to the internet and using the domain ddev.site this is generally not necessary, because the hosts file never gets manipulated.`,
	Run: func(_ *cobra.Command, args []string) {

		// Unless DDEV_NONINTERACTIVE is set (tests) then we need to be admin
		if os.Getenv("DDEV_NONINTERACTIVE") == "" && os.Geteuid() != 0 && !checkHostnameFlag && !removeInactiveFlag && runtime.GOOS != "windows" {
			util.Failed("'ddev hostname %s' must be run with administrator privileges", strings.Join(args, " "))
		}

		// TODO: Reimplement this, figure out how to know what's inactive
		// This may not be useful and not need to be implemented.
		// If requested, remove all inactive host names and exit
		//if removeInactiveFlag {
		//	if len(args) > 0 {
		//		util.Failed("Invalid arguments supplied. 'ddev hostname --remove-all' accepts no arguments.")
		//	}
		//
		//	util.Warning("Attempting to remove inactive custom hostnames for projects which are registered but not running")
		//	//removeInactiveHostnames()
		//	return
		//}

		// If operating on one host name, two arguments are required
		if len(args) != 2 {
			util.Failed("Invalid arguments supplied. Please use 'ddev_hostname hostname [hostname] [ip]'")
		}

		name, dockerIP := args[0], args[1]
		var err error

		util.Debug("Escalating privileges to add host entry %s -> %s", name, dockerIP)
		escalateIfNeeded()

		// If requested, remove the provided host name and exit
		if removeHostnameFlag {
			err = hostname.RemoveHostEntry(name, dockerIP)
			if err != nil {
				util.Warning("Failed to remove host entry %s: %v", name, err)
			}
			return
		}
		if checkHostnameFlag {
			exists, err := hostname.IsHostnameInHostsFile(name)
			if exists {
				return
			}
			if err != nil {
				util.Warning("Could not check existence in hosts file: %v", err)
			}
			os.Exit(1)
		}
		// By default, add a host name
		err = hostname.AddHostEntry(name, dockerIP)

		if err != nil {
			util.Warning("Failed to add hosts entry %s: %v", name, err)
		}
	},
}

// removeInactiveHostnames will remove all host names except those current in use by active projects.
//func removeInactiveHostnames() {
//	apps, err := ddevapp.GetInactiveProjects()
//	if err != nil {
//		util.Warning("Unable to run GetInactiveProjects: %v", err)
//		return
//	}
//	for _, app := range apps {
//		err := app.RemoveHostsEntriesIfNeeded()
//		if err != nil {
//			util.Warning("Unable to remove hosts entries for project '%s': %v", app.Name, err)
//		}
//	}
//}

func init() {
	RootCmd.Flags().BoolVarP(&removeHostnameFlag, "remove", "r", false, "Remove the provided host name - ip correlation")
	RootCmd.Flags().BoolVarP(&checkHostnameFlag, "check", "c", false, "Check to see if provided hostname is already in hosts file")
	//RootCmd.Flags().BoolVarP(&removeInactiveFlag, "remove-inactive", "R", false, "Remove host names of inactive projects")
}

func escalateIfNeeded() {
	// If we’re not root (UID 0), re‐exec via sudo
	if syscall.Geteuid() != 0 {
		// Prepend our own path to the args
		args := append([]string{os.Args[0]}, os.Args[1:]...)
		cmd := exec.Command("sudo", args...)
		// Pass through the terminal’s stdin/stdout/stderr
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to escalate: %v\n", err)
			os.Exit(1)
		}
		// If sudo succeeds, it will have done the real work,
		// so we just exit in the parent process.
		os.Exit(0)
	}
	// else: we’re already root, continue
}

// +build windows

import (
"fmt"
"os"
"syscall"
"unsafe"

"golang.org/x/sys/windows"
)

func isElevated() bool {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		return false
	}
	defer token.Close()

	var elevation windows.TokenElevation
	var retLen uint32
	if err := windows.GetTokenInformation(token, windows.TokenElevation, (*byte)(unsafe.Pointer(&elevation)), uint32(unsafe.Sizeof(elevation)), &retLen); err != nil {
		return false
	}
	return elevation.IsElevated != 0
}

func elevateSelf() {
	verbPtr, _ := syscall.UTF16PtrFromString("runas")
	exePath, _ := os.Executable()
	exePtr, _ := syscall.UTF16PtrFromString(exePath)

	// Reconstruct command-line arguments
	args := ""
	if len(os.Args) > 1 {
		args = " " + windows.EscapeArg(os.Args[1:])
	}
	argsPtr, _ := syscall.UTF16PtrFromString(args)

	var sei windows.ShellExecuteInfo
	sei.CbSize = uint32(unsafe.Sizeof(sei))
	sei.FMask = windows.SEE_MASK_NOCLOSEPROCESS
	sei.LpVerb = verbPtr
	sei.LpFile = exePtr
	sei.LpParameters = argsPtr
	sei.NShow = windows.SW_NORMAL

	if err := windows.ShellExecuteEx(&sei); err != nil {
		fmt.Fprintln(os.Stderr, "Elevation failed:", err)
		os.Exit(1)
	}
	// Wait for elevated process to finish
	windows.WaitForSingleObject(sei.HProcess, windows.INFINITE)

	// Propagate its exit code
	var code uint32
	windows.GetExitCodeProcess(sei.HProcess, &code)
	os.Exit(int(code))
}


