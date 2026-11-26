package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	user32                 = syscall.NewLazyDLL("user32.dll")
	procFindWindowExW      = user32.NewProc("FindWindowExW")
	procGetWindowThreadPID = user32.NewProc("GetWindowThreadProcessId")
	procPostMessageW       = user32.NewProc("PostMessageW")
	procSendMessageW       = user32.NewProc("SendMessageW")
	procEnumWindows        = user32.NewProc("EnumWindows")
	procGetClassNameW      = user32.NewProc("GetClassNameW")
	procIsWindowVisible    = user32.NewProc("IsWindowVisible")
	procGetMenuItemCount   = user32.NewProc("GetMenuItemCount")
	procGetMenuStringW     = user32.NewProc("GetMenuStringW")
	procGetMenuItemID      = user32.NewProc("GetMenuItemID")
	procGetTopWindow       = user32.NewProc("GetTopWindow")
	procGetSystemMetrics   = user32.NewProc("GetSystemMetrics")
	procGetWindow          = user32.NewProc("GetWindow")
	procGetWindowLongPtrW  = user32.NewProc("GetWindowLongPtrW")
	procIsIconic           = user32.NewProc("IsIconic")
	procGetWindowRect      = user32.NewProc("GetWindowRect")
	procDwmGetWindowAttr   = syscall.NewLazyDLL("dwmapi.dll").NewProc("DwmGetWindowAttribute")
	enumMenuCallback       = syscall.NewCallback(enumMenuWindow)
)

// foundMenu is enumMenuWindow's result; EnumWindows runs the callback on
// the calling goroutine.
var foundMenu uintptr

const (
	hwndMessage    = ^uintptr(2) // HWND_MESSAGE, (HWND)-3
	wmUserSystray  = 0x400 + 1   // Wails' tray callback, WM_USER+1
	wmContextMenu  = 0x7B
	wmLButtonUp    = 0x202
	wmCommand      = 0x111
	wmKeyDown      = 0x100
	vkEscape       = 0x1B
	mnGetHMenu     = 0x1E1
	mfByPosition   = 0x400
	menuClassName  = "#32768"
	trayClassName  = "WailsWebviewWindow"
	trayWaitPeriod = 10 * time.Second
	gwHwndNext     = 2
	gwlExStyle     = ^uintptr(19) // GWL_EXSTYLE, -20
	wsExTopmost    = 0x8
	wsExToolWindow = 0x80
	dwmwaCloaked   = 14
	smCxSmIcon     = 49
)

// traySize is the size, in pixels, of an icon in the notification area.
func traySize() int {
	n, _, _ := procGetSystemMetrics.Call(smCxSmIcon)
	return int(n)
}

// inFront is whether the window is the top one on screen. Always-on-top
// windows don't count, so neither do the taskbar and its overflow flyout,
// which take the focus before a click on the tray icon arrives.
func inFront(window *application.WebviewWindow) (front, known bool) {
	own := uintptr(window.NativeWindow())
	w, _, _ := procGetTopWindow.Call(0)
	for ; w != 0; w, _, _ = procGetWindow.Call(w, gwHwndNext) {
		if w == own {
			return true, true
		}
		if visible, _, _ := procIsWindowVisible.Call(w); visible == 0 {
			continue
		}
		if iconic, _, _ := procIsIconic.Call(w); iconic != 0 {
			continue
		}
		ex, _, _ := procGetWindowLongPtrW.Call(w, gwlExStyle)
		if ex&(wsExTopmost|wsExToolWindow) != 0 {
			continue
		}
		// Windows keeps some windows of Store apps visible but cloaked.
		var cloaked uint32
		if r, _, _ := procDwmGetWindowAttr.Call(w, dwmwaCloaked, uintptr(unsafe.Pointer(&cloaked)), 4); r == 0 && cloaked != 0 {
			continue
		}
		var rect struct{ left, top, right, bottom int32 }
		_, _, _ = procGetWindowRect.Call(w, uintptr(unsafe.Pointer(&rect)))
		if rect.right <= rect.left || rect.bottom <= rect.top {
			continue
		}
		return false, true
	}
	return false, true
}

// waitForTrayWindow returns once Wails has created the tray's window. A
// SetMenu that runs earlier builds the popup menu with no owner window, and
// Wails keeps that owner, so every right-click then fails. The window is
// created on the main thread with the tray's first menu, so a SetMenu
// dispatched after it appears runs after that.
func waitForTrayWindow() {
	if !waitFor(func() bool { return ownMessageWindow(trayClassName) != 0 }) {
		log.Printf("no tray window after %v; the tray menu may not open", trayWaitPeriod)
	}
}

// checkTray clicks the tray icon as the shell does, by posting its callback
// message to the app's own tray window. A left-click must show the hidden
// window, a right-click must open the menu, and "Open DDEV" must show the
// window again.
func checkTray(window *application.WebviewWindow) error {
	tray := ownMessageWindow(trayClassName)
	if tray == 0 {
		return errors.New("no tray window found")
	}

	window.Hide()
	post(tray, wmUserSystray, 0, wmLButtonUp)
	if !waitFor(window.IsVisible) {
		return errors.New("a left-click on the tray icon did not show the window")
	}

	post(tray, wmUserSystray, 0, wmContextMenu)
	var menu uintptr
	if !waitFor(func() bool { menu = ownMenuWindow(); return menu != 0 }) {
		return errors.New("a right-click on the tray icon opened no menu")
	}
	hmenu, _, _ := procSendMessageW.Call(menu, mnGetHMenu, 0, 0)
	items := menuItems(hmenu)
	open, ok := items["Open DDEV"]
	if !ok || !hasKey(items, "Quit DDEV app") {
		return fmt.Errorf("the tray menu lacks Open DDEV or Quit DDEV app: %v", keys(items))
	}
	post(menu, wmKeyDown, vkEscape, 0)
	if !waitFor(func() bool { return ownMenuWindow() == 0 }) {
		return errors.New("the tray menu did not close")
	}

	window.Hide()
	post(tray, wmCommand, open, 0)
	if !waitFor(window.IsVisible) {
		return errors.New("the tray menu's Open DDEV did not show the window")
	}
	return nil
}

func post(hwnd, msg, wParam, lParam uintptr) {
	_, _, _ = procPostMessageW.Call(hwnd, msg, wParam, lParam)
}

func waitFor(cond func() bool) bool {
	for end := time.Now().Add(trayWaitPeriod); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
		if cond() {
			return true
		}
	}
	return false
}

func ownedByThisProcess(hwnd uintptr) bool {
	var pid uint32
	_, _, _ = procGetWindowThreadPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return int(pid) == os.Getpid()
}

// ownMessageWindow returns this process's message-only window of the class,
// which for Wails' class is the tray's.
func ownMessageWindow(class string) uintptr {
	name, _ := syscall.UTF16PtrFromString(class)
	var hwnd uintptr
	for {
		hwnd, _, _ = procFindWindowExW.Call(hwndMessage, hwnd, uintptr(unsafe.Pointer(name)), 0)
		if hwnd == 0 || ownedByThisProcess(hwnd) {
			return hwnd
		}
	}
}

// ownMenuWindow returns this process's open popup menu, or 0.
func ownMenuWindow() uintptr {
	foundMenu = 0
	_, _, _ = procEnumWindows.Call(enumMenuCallback, 0)
	return foundMenu
}

func enumMenuWindow(hwnd, _ uintptr) uintptr {
	buf := make([]uint16, 64)
	n, _, _ := procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if syscall.UTF16ToString(buf[:n]) != menuClassName || !ownedByThisProcess(hwnd) {
		return 1
	}
	if visible, _, _ := procIsWindowVisible.Call(hwnd); visible == 0 {
		return 1
	}
	foundMenu = hwnd
	return 0
}

// menuItems maps each top-level item's label to its command ID.
func menuItems(hmenu uintptr) map[string]uintptr {
	items := map[string]uintptr{}
	count, _, _ := procGetMenuItemCount.Call(hmenu)
	for i := uintptr(0); i < count && int32(count) > 0; i++ {
		buf := make([]uint16, 256)
		n, _, _ := procGetMenuStringW.Call(hmenu, i, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), mfByPosition)
		label, _, _ := strings.Cut(syscall.UTF16ToString(buf[:n]), "\t")
		if label == "" {
			continue
		}
		id, _, _ := procGetMenuItemID.Call(hmenu, i)
		items[label] = id
	}
	return items
}

func hasKey(m map[string]uintptr, k string) bool {
	_, ok := m[k]
	return ok
}

func keys(m map[string]uintptr) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
