package main

/*
#cgo pkg-config: gtk4
#include <gtk/gtk.h>

static void set_prefer_dark(gboolean dark) {
	g_object_set(gtk_settings_get_default(), "gtk-application-prefer-dark-theme", dark, NULL);
}
*/
import "C"

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// setAppName names the app for GTK, which titles Wails' dialogs with it
// instead of the app ID, com.ddev.gui.
func setAppName() {
	C.g_set_application_name(C.CString("DDEV"))
}

// followDarkMode passes the desktop's dark mode to GTK, which WebKitGTK
// reports to the page. GTK without libadwaita does not read it itself.
func followDarkMode(app *application.App) {
	set := func(dark bool) {
		var v C.gboolean
		if dark {
			v = 1
		}
		application.InvokeAsync(func() { C.set_prefer_dark(v) })
	}
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		set(app.Env.IsDarkMode())
	})
	app.Event.OnApplicationEvent(events.Linux.SystemThemeChanged, func(e *application.ApplicationEvent) {
		set(e.Context().IsDarkMode())
	})
}
