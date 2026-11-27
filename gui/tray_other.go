//go:build !windows

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// waitForTrayWindow does nothing: only Wails' Windows tray needs its window
// before the menu changes.
func waitForTrayWindow() {}

// traySize is 0: only the Windows tray needs icons of its own size.
func traySize() int { return 0 }

// inFront is unknown: only Windows tells which window is on top.
func inFront(*application.WebviewWindow) (front, known bool) { return false, false }

// checkTray does nothing: only on Windows can the app click its own tray
// icon, by posting the message the shell would send.
func checkTray(*application.WebviewWindow) error { return nil }
