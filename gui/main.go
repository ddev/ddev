// Command ddev-gui is an experimental GUI app for DDEV. It shows the
// `ddev gui` dashboard in a native window through Wails, and adds a tray
// menu and notifications.
//
// It holds no DDEV code: it runs `ddev gui --desktop` from the ddev it
// finds (see serve.go), so the window, the tray, and the terminal all use
// one ddev. It is a separate module so that Wails, and the CGO it needs on
// macOS and Linux, stay out of the ddev binary and its vendor directory.
package main

import (
	"context"
	_ "embed"
	"fmt"
	"html"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

//go:embed icons/appicon.png
var appIcon []byte

// appIconWindows is the mark without the macOS tile, at each size that
// Windows shows, since Windows shrinks a PNG without smoothing. GTK 4
// takes the window icon from the desktop file instead.
//
//go:embed icons/appicon.ico
var appIconWindows []byte

//go:embed icons/tray.png
var trayIcon []byte

//go:embed icons/tray-color.png
var trayIconColor []byte

// version is set by the Makefile.
var version = "dev"

func main() {
	if os.Geteuid() == 0 {
		log.Fatal("DDEV is not designed to run as root")
	}
	shareHiddenConsole()
	if err := checkWebKitSandbox(); err != nil {
		log.Fatal(err)
	}
	setAppName()
	useLoginShellPath()

	var st *selfTest
	if os.Getenv("DDEV_GUI_SELFTEST") == "true" {
		var err error
		if st, err = newSelfTest(); err != nil {
			log.Fatal(err)
		}
	}

	// startErr is shown in the window and a dialog when ddev gui cannot
	// run, such as when ddev is missing or too old to have `ddev gui`.
	var startErr error
	var srv *serveClient
	tgt, err := chooseTarget(argValue(os.Args[1:], "--wsl-distro"))
	if err == nil {
		log.Printf("Running ddev gui from %s", tgt)
		srv, err = startServe(tgt)
	}
	if err != nil {
		startErr = err
		log.Print(err)
		if st != nil {
			fmt.Printf("SELFTEST FAIL: %v\n", err)
			st.cleanup()
			os.Exit(1)
		}
	}

	// handler serves the window through the Wails asset server. On macOS
	// and Linux it passes the window's requests to ddev gui. WebView2 on
	// Windows cannot stream a response through it, so there the window
	// loads ddev gui's URL instead.
	handler := errorPage(startErr)
	windowURL := "/"
	if srv != nil {
		handler = srv.Proxy()
		if runtime.GOOS == "windows" {
			windowURL = srv.URL()
		}
	}
	if st != nil {
		handler = st.wrap(handler)
		if runtime.GOOS == "windows" {
			// The self-test must add its script to the page, so on Windows
			// the window loads the proxy from a port of its own.
			if windowURL, err = serveLoopback(handler); err != nil {
				log.Fatal(err)
			}
		}
	}
	// `ddev gui <project>` starts the app with --project.
	if p := argValue(os.Args[1:], "--project"); p != "" && srv != nil {
		windowURL += "#project=" + url.QueryEscape(p)
	}
	logLevel := slog.LevelInfo
	if os.Getenv("DDEV_GUI_DEBUG") == "true" {
		handler = logRequests(handler)
		logLevel = slog.LevelDebug
	}

	notifier := notifications.New()
	// notifyOn is set once the notifier has started; jobs may end before.
	var notifyOn atomic.Bool
	var window *application.WebviewWindow
	if srv != nil {
		srv.OnJobDone = func(title, project, errText string) {
			// A notification is only useful when the window is not in front.
			if !notifyOn.Load() || (window.IsVisible() && window.IsFocused()) {
				return
			}
			n := notifications.NotificationOptions{ID: fmt.Sprintf("job-%d", time.Now().UnixNano()), Title: title, Body: "Finished"}
			if errText != "" {
				n.Body = "Failed: " + errText
			}
			if project != "" {
				n.Subtitle = project
			}
			_ = notifier.SendNotification(n)
		}
		go srv.Follow()
	}
	shutdown := func() {
		if srv != nil {
			srv.Close()
		}
	}

	showWindow := func() {
		if window.IsMinimised() {
			window.UnMinimise()
		}
		window.Show()
		window.Focus()
	}
	// lostFocus is when the window last lost the focus, and clickShown when
	// a tray click last showed it, both in Unix nanoseconds.
	var lostFocus, clickShown atomic.Int64
	// toggleWindow hides the window when it is in front, and else shows it.
	// Where the system can't say, the focus stands in: a click on the tray
	// may take it, and on Wayland a window a click showed may get none.
	toggleWindow := func() {
		front, known := inFront(window)
		if !known {
			lost := lostFocus.Load()
			front = window.IsFocused() || time.Since(time.Unix(0, lost)) < 500*time.Millisecond || clickShown.Load() > lost
		}
		if window.IsVisible() && !window.IsMinimised() && front {
			clickShown.Store(0)
			window.Hide()
			return
		}
		clickShown.Store(time.Now().UnixNano())
		showWindow()
	}
	// reveal shows the window with a project selected.
	var reveal func(name string)
	singleInstance := &application.SingleInstanceOptions{
		UniqueID: "com.ddev.gui",
		// Launching the app again is how a closed window comes back where
		// there is no tray or Dock icon, such as GNOME without AppIndicator.
		// `ddev gui <project>` with the app running passes --project here.
		OnSecondInstanceLaunch: func(d application.SecondInstanceData) {
			application.InvokeAsync(func() {
				if p := argValue(d.Args, "--project"); p != "" && reveal != nil {
					reveal(p)
					return
				}
				showWindow()
			})
		},
	}
	if st != nil {
		// A self-test runs beside the user's own copy of the app.
		singleInstance = nil
	}

	// quitter asks before a quit would kill running jobs.
	q := &quitter{srv: srv}
	icon := appIcon
	if runtime.GOOS == "windows" {
		icon = appIconWindows
	}
	app := application.New(application.Options{
		Name:        "DDEV",
		Description: "Experimental DDEV GUI app " + version,
		LogLevel:    logLevel,
		Icon:        icon,
		Assets: application.AssetOptions{
			Handler:        handler,
			DisableLogging: true,
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		// The Wayland app ID, which must name the launcher,
		// com.ddev.gui.desktop, for the desktop to show its icon.
		Linux:          application.LinuxOptions{ApplicationID: "com.ddev.gui"},
		SingleInstance: singleInstance,
		// Wails clears WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS, so a debugging
		// port for the window's DevTools must come through here.
		Windows:    application.WindowsOptions{AdditionalBrowserArgs: strings.Fields(os.Getenv("DDEV_GUI_WEBVIEW2_ARGS"))},
		OnShutdown: shutdown,
		ShouldQuit: q.shouldQuit,
		// Wails' own handler reads only the first signal, so a quit that
		// ShouldQuit refused would leave the app deaf to the next one.
		DisableDefaultSignalHandler: true,
	})
	q.app, q.show = app, showWindow
	q.handleSignals()
	followDarkMode(app)

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     "DDEV",
		Width:     1280,
		Height:    820,
		MinWidth:  720,
		MinHeight: 480,
		URL:       windowURL,
	})
	q.window = window
	window.OnWindowEvent(events.Common.WindowLostFocus, func(*application.WindowEvent) {
		lostFocus.Store(time.Now().UnixNano())
	})
	// Closing the window keeps the app in the tray, as Docker Desktop does.
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})
	app.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(*application.ApplicationEvent) {
		showWindow()
	})
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		fitWindow(app.Screen.GetPrimary(), window)
		if st != nil {
			go runSelfTest(srv, st, func() error { return checkTray(window) })
		}
		if startErr != nil {
			app.Dialog.Error().SetTitle("DDEV cannot start").SetMessage(startErr.Error()).Show()
			return
		}
		go watchExit(app, srv, q)
		notifyOn.Store(startNotifications(notifier, st == nil))
	})

	// pageURL is the dashboard page once loaded, without the launch token.
	pageURL := "/"
	if u, err := url.Parse(windowURL); err == nil && u.Host != "" {
		pageURL = u.Scheme + "://" + u.Host + "/"
	}
	// navigate shows the window and target, such as "about", which ddev gui
	// passes to the page: a new URL leaves WebView2 black on the next show.
	// A ddev without that route falls back to the URL's hash.
	navigate := func(target string) {
		showWindow()
		if srv != nil && srv.Post("/api/show?target="+url.QueryEscape(target)) == nil {
			return
		}
		window.SetURL(pageURL + "#" + target)
	}
	reveal = func(name string) { navigate("project=" + url.QueryEscape(name)) }
	if runtime.GOOS == "darwin" && srv != nil {
		// The app menu's About shows the page's, with the versions of both.
		menu := application.DefaultApplicationMenu()
		if about := menu.FindByRole(application.About); about != nil {
			about.OnClick(func(*application.Context) { navigate("about") })
		}
		app.Menu.Set(menu)
	}
	startTray(&trayMenu{app: app, srv: srv, startErr: startErr, show: showWindow, navigate: navigate}, toggleWindow)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// startNotifications starts the notifier and, unless ask is false, asks for
// permission to notify. It returns whether notifications work: they need an
// app bundle on macOS and a session bus on Linux.
func startNotifications(notifier *notifications.NotificationService, ask bool) bool {
	// Windows registers a COM class, which needs the main thread.
	if err := application.InvokeSyncWithError(func() error {
		return notifier.ServiceStartup(context.Background(), application.ServiceOptions{})
	}); err != nil {
		log.Printf("Notifications are off: %v", err)
		return false
	}
	if !ask {
		return true
	}
	if ok, _ := notifier.CheckNotificationAuthorization(); !ok {
		// macOS refuses an ad-hoc signed app here, such as a PR build.
		if granted, err := notifier.RequestNotificationAuthorization(); err != nil {
			log.Printf("Notifications are off: %v", err)
		} else if !granted {
			log.Printf("Notifications are off: they are not allowed for DDEV in the system settings")
		}
	}
	return true
}

// fitWindow shrinks the window to fit the screen, such as a 1366x768
// laptop's, where Windows would put its title bar above the screen's edge.
func fitWindow(s *application.Screen, window *application.WebviewWindow) {
	if s == nil {
		return
	}
	w, h := window.Size()
	maxW, maxH := s.WorkArea.Width*9/10, s.WorkArea.Height*9/10
	if w <= maxW && h <= maxH {
		return
	}
	window.SetSize(min(w, maxW), min(h, maxH))
	window.Center()
}

// argValue returns the value of a --name=value or --name value argument.
func argValue(args []string, name string) string {
	for i, a := range args {
		if v, ok := strings.CutPrefix(a, name+"="); ok {
			return v
		}
		if a == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// errorPage tells the window why there is no dashboard.
func errorPage(err error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		msg := "Starting…"
		if err != nil {
			msg = err.Error()
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, `<!doctype html><title>DDEV</title><body style="font-family:system-ui;margin:2em"><h1>DDEV cannot start</h1><pre style="white-space:pre-wrap">%s</pre></body>`, html.EscapeString(msg))
	})
}

// serveLoopback serves h on a free port of 127.0.0.1 and returns its URL.
func serveLoopback(h http.Handler) (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	go func() { _ = (&http.Server{Handler: h, ReadHeaderTimeout: 10 * time.Second}).Serve(l) }()
	return "http://" + l.Addr().String() + "/", nil
}

// watchExit tells the user when ddev gui stops while the app runs, since
// the window and the tray need it.
func watchExit(app *application.App, srv *serveClient, q *quitter) {
	<-srv.Exited()
	if srv.Closing() {
		return
	}
	msg := "ddev gui stopped, so DDEV must quit."
	if out := srv.Output(); out != "" {
		msg += "\n\n" + out
	}
	log.Print(msg)
	application.InvokeAsync(func() {
		d := app.Dialog.Error().SetTitle("DDEV stopped").SetMessage(msg)
		d.AddButton("Quit").OnClick(func() {
			q.force.Store(true)
			go app.Quit()
		})
		d.Show()
	})
}

// logRequests logs each request the webview makes, since the window has no
// developer tools in a production build.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s host=%q origin=%q", r.Method, r.URL.RequestURI(), r.Host, r.Header.Get("Origin"))
		next.ServeHTTP(w, r)
	})
}
