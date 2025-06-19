//go:build windows
// +build windows

package main

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
