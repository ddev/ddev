package main

import (
	"os/exec"
	"syscall"
	"time"
)

// createNoWindow gives a console program a console without a window.
const createNoWindow = 0x08000000

// shareHiddenConsole attaches this app to a console that has no window. A
// GUI app has no console, so Windows would open a new one, in a window, for
// each console program that the app or DDEV's own code starts. They share
// this one instead. The console belongs to a helper that waits on its
// standard input, so it ends when the app does.
func shareHiddenConsole() {
	helper := exec.Command("cmd.exe", "/d", "/c", "pause")
	helper.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
	if _, err := helper.StdinPipe(); err != nil {
		return
	}
	if err := helper.Start(); err != nil {
		return
	}
	attach := syscall.NewLazyDLL("kernel32.dll").NewProc("AttachConsole")
	// The helper's console exists a moment after the helper starts.
	for range 50 {
		if ok, _, _ := attach.Call(uintptr(helper.Process.Pid)); ok != 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}
