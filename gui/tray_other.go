//go:build !windows

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// waitForTrayWindow does nothing: only Wails' Windows tray needs its window
// before the menu changes.
func waitForTrayWindow() {}

// checkTray does nothing: only on Windows can the app click its own tray
// icon, by posting the message the shell would send.
func checkTray(*application.WebviewWindow) error { return nil }
