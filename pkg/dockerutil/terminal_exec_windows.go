//go:build windows

package dockerutil

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// killWithParent puts p in a job object that Windows closes when ddev exits,
// however it exits, so p ends with ddev. Stop-Process and Task Manager end
// ddev without a console event, so the interrupt handler never runs.
func killWithParent(p *os.Process) (release func()) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return func() {}
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	_, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)))
	if err == nil {
		if h, openErr := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(p.Pid)); openErr == nil {
			_ = windows.AssignProcessToJobObject(job, h)
			_ = windows.CloseHandle(h)
		}
	}
	return func() { _ = windows.CloseHandle(job) }
}
