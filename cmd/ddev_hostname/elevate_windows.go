//go:build windows
// +build windows

package main

import (
	"fmt"
	"github.com/ddev/ddev/pkg/util"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modShell32          = windows.NewLazySystemDLL("shell32.dll")
	procShellExecuteExW = modShell32.NewProc("ShellExecuteExW")
)

const (
	SEE_MASK_NOCLOSEPROCESS = 0x00000040
)

type shellExecuteInfo struct {
	cbSize         uint32
	fMask          uint32
	hwnd           uintptr
	lpVerb         *uint16
	lpFile         *uint16
	lpParameters   *uint16
	lpDirectory    *uint16
	nShow          int32
	hInstApp       uintptr
	lpIDList       uintptr
	lpClass        *uint16
	hkeyClass      uintptr
	dwHotKey       uint32
	hIconOrMonitor uintptr
	hProcess       windows.Handle
}

func elevateIfNeeded() {
	if !isElevated() {
		fmt.Fprintln(os.Stderr, "This program requires elevated privileges. Attempting to elevate.")
		elevate()
	} else {
		fmt.Println("Running with elevated privileges.")
	}
}

func isElevated() bool {
	// Open this process’s token with QUERY access
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		return false
	}
	defer token.Close()

	// Query the TokenElevation field
	var elevation struct{ TokenIsElevated uint32 }
	var retLen uint32
	err := windows.GetTokenInformation(
		token,
		windows.TokenElevation,              // info class 20
		(*byte)(unsafe.Pointer(&elevation)), // out buffer
		uint32(unsafe.Sizeof(elevation)),    // buffer length
		&retLen,
	)
	if err != nil {
		return false
	}
	return elevation.TokenIsElevated != 0
}

func elevate() {
	util.Debug("elevate() called")
	// Prepare UTF-16 pointers
	verbPtr, _ := syscall.UTF16PtrFromString("runas")
	exePath, _ := os.Executable()
	exePtr, _ := syscall.UTF16PtrFromString(exePath)
	paramStr := strings.Join(os.Args[1:], " ")
	paramsPtr, _ := syscall.UTF16PtrFromString(paramStr)

	sei := shellExecuteInfo{
		cbSize:       uint32(unsafe.Sizeof(shellExecuteInfo{})),
		fMask:        SEE_MASK_NOCLOSEPROCESS,
		hwnd:         0,
		lpVerb:       verbPtr,
		lpFile:       exePtr,
		lpParameters: paramsPtr,
		lpDirectory:  nil,
		nShow:        windows.SW_NORMAL,
		// hInstApp, lpIDList, lpClass, hkeyClass, dwHotKey, hIconOrMonitor, hProcess are zeroed
	}

	ret, _, lastErr := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&sei)))
	if ret == 0 {
		fmt.Fprintln(os.Stderr, "Elevation failed:", lastErr)
		os.Exit(1)
	}

	if sei.hProcess != 0 {
		_, err := windows.WaitForSingleObject(sei.hProcess, windows.INFINITE)
		if err != nil {
			util.Failed("Failed to wait for elevated process: %v", err)
		}
		var exitCode uint32
		err = windows.GetExitCodeProcess(sei.hProcess, &exitCode)
		if err != nil {
			util.Failed("Failed to get exit code: %v", err)
		}
		os.Exit(int(exitCode))
	} else {
		// If hProcess is not set, just exit
		os.Exit(0)
	}
}
