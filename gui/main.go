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
	"os/signal"
	"runtime"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"
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
		if st != nil {
			go runSelfTest(srv, st, func() error { return checkTray(window) })
		}
		if startErr != nil {
			app.Dialog.Error().SetTitle("DDEV cannot start").SetMessage(startErr.Error()).Show()
			return
		}
		go watchExit(app, srv, q)
		// Notifications need an app bundle on macOS and a session bus on
		// Linux; without them the app runs without notifications. Windows
		// registers a COM class, which needs the main thread.
		if err := application.InvokeSyncWithError(func() error {
			return notifier.ServiceStartup(context.Background(), application.ServiceOptions{})
		}); err != nil {
			log.Printf("Notifications are off: %v", err)
			return
		}
		if st == nil {
			if ok, _ := notifier.CheckNotificationAuthorization(); !ok {
				// macOS refuses an ad-hoc signed app here, such as a PR build.
				if granted, err := notifier.RequestNotificationAuthorization(); err != nil {
					log.Printf("Notifications are off: %v", err)
				} else if !granted {
					log.Printf("Notifications are off: they are not allowed for DDEV in the system settings")
				}
			}
		}
		notifyOn.Store(true)
	})

	tray := app.SystemTray.New()
	icons := newTrayIcons(tray, runtime.GOOS)
	if runtime.GOOS == "linux" {
		// Without a label, Wails names the item "Wails". The hover text shows
		// the label, so a tooltip would repeat it. macOS shows a label as text.
		tray.SetLabel("DDEV")
	} else {
		tray.SetTooltip("DDEV")
	}
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
	showAbout := func() { navigate("about") }
	if runtime.GOOS == "darwin" && srv != nil {
		// The app menu's About shows the page's, with the versions of both.
		menu := application.DefaultApplicationMenu()
		if about := menu.FindByRole(application.About); about != nil {
			about.OnClick(func(*application.Context) { showAbout() })
		}
		app.Menu.Set(menu)
	}
	t := &trayMenu{app: app, tray: tray, icons: icons, window: window, srv: srv, startErr: startErr, show: showWindow, reveal: reveal, about: showAbout}
	// Wails opens the menu on a right-click only if the tray has a menu when
	// the app starts, and follow's first menu arrives later.
	firstErr := "Connecting to ddev gui…"
	if startErr != nil {
		firstErr = startErr.Error()
	}
	tray.SetMenu(t.build(nil, firstErr))
	if runtime.GOOS != "darwin" {
		// As other Windows and Linux tray apps do, a click shows or hides the
		// window. A macOS menu bar extra opens its menu.
		tray.OnClick(toggleWindow)
	}
	go t.follow()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
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

// runSelfTest waits for selftest.js to report, prints the result, and exits
// with its status. Wails' Quit ends the process without returning from Run.
// tray checks the tray menu once the page has passed.
func runSelfTest(srv *serveClient, st *selfTest, tray func() error) {
	code := 1
	select {
	case res := <-st.result:
		res = st.check(res)
		if res.OK {
			if err := tray(); err != nil {
				res = selfTestResult{Step: "tray menu", Error: err.Error()}
			}
		}
		if res.OK {
			fmt.Println("SELFTEST PASS")
			code = 0
		} else {
			fmt.Printf("SELFTEST FAIL at %s: %s\n%s\n", res.Step, res.Error, res.Output)
		}
	case <-time.After(3 * time.Minute):
		fmt.Println("SELFTEST FAIL: no result from the window within 3 minutes")
	}
	srv.Close()
	st.cleanup()
	os.Exit(code)
}

// logRequests logs each request the webview makes, since the window has no
// developer tools in a production build.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s host=%q origin=%q", r.Method, r.URL.RequestURI(), r.Host, r.Header.Get("Origin"))
		next.ServeHTTP(w, r)
	})
}

// trayMenu keeps the tray menu in step with the projects.
type trayMenu struct {
	app    *application.App
	tray   *application.SystemTray
	window *application.WebviewWindow
	// srv is nil when ddev gui could not start, and startErr says why.
	srv      *serveClient
	startErr error
	show     func()
	icons    *trayIcons
	// reveal shows the window with a project selected, and about with
	// About DDEV.
	reveal func(name string)
	about  func()
	last   []project
	err    string
}

func (t *trayMenu) follow() {
	waitForTrayWindow()
	first := true
	for {
		var changed <-chan struct{}
		var projects []project
		var err error
		if t.srv != nil {
			changed = t.srv.Changed()
			projects, err = t.srv.Projects()
		} else {
			err = t.startErr
		}
		errText := ""
		if err != nil {
			errText = err.Error()
		}
		if first || errText != t.err || !slices.Equal(projects, t.last) {
			first, t.last, t.err = false, projects, errText
			menu := t.build(projects, errText)
			application.InvokeAsync(func() { t.tray.SetMenu(menu) })
		}
		running := 0
		if t.srv != nil {
			running = len(t.srv.RunningJobs())
		}
		t.icons.set(stateOf(projects, running))
		select {
		case <-changed:
		case <-time.After(5 * time.Second):
		}
	}
}

func (t *trayMenu) build(projects []project, errText string) *application.Menu {
	m := t.app.NewMenu()
	m.Add("Open DDEV").OnClick(func(*application.Context) { t.show() })
	m.AddSeparator()
	if errText != "" {
		// A menu item shows one line.
		errText, _, _ = strings.Cut(errText, "\n")
		m.Add(trayText(errText)).SetEnabled(false)
	} else if len(projects) == 0 {
		m.Add("No projects").SetEnabled(false)
	}
	running := 0
	for _, p := range projects {
		sub := m.AddSubmenu(statusMark(p.Status) + " " + p.Name)
		sub.Add(p.Status + " · " + p.Type).SetEnabled(false)
		sub.AddSeparator()
		// A project that is starting or unhealthy is up too, and needs Stop.
		if p.Up {
			running++
			if site := p.URL; site != "" {
				sub.Add("Open site").OnClick(func(*application.Context) { _ = t.app.Browser.OpenURL(site) })
			}
			if mailpit := p.Mailpit; mailpit != "" {
				sub.Add("Open Mailpit").OnClick(func(*application.Context) { _ = t.app.Browser.OpenURL(mailpit) })
			}
			sub.Add("Restart").OnClick(t.action(p.Name, "restart"))
			sub.Add("Stop").OnClick(t.action(p.Name, "stop"))
		} else {
			sub.Add("Start").OnClick(t.action(p.Name, "start"))
		}
		sub.AddSeparator()
		name := p.Name
		sub.Add("Show in DDEV").OnClick(func(*application.Context) { t.reveal(name) })
	}
	m.AddSeparator()
	m.Add("Start all").SetEnabled(len(projects) > running).OnClick(t.global("start-all"))
	// The page asks before these too.
	m.Add("Stop all").SetEnabled(running > 0).OnClick(t.confirm("Stop all projects?", "Every running DDEV project will be stopped.", "Stop all", t.global("stop-all")))
	m.Add("Power off").OnClick(t.confirm("Power off DDEV?", "This stops all projects and DDEV’s shared containers.", "Power off", t.global("poweroff")))
	m.AddSeparator()
	if t.srv != nil {
		m.Add("About DDEV").OnClick(func(*application.Context) { t.about() })
	}
	m.Add("Quit DDEV app").OnClick(func(*application.Context) { t.app.Quit() })
	return m
}

// trayText keeps the underscores of a Linux tray label, where a single one
// marks the next letter as a shortcut key and is not shown.
func trayText(s string) string {
	if runtime.GOOS != "linux" {
		return s
	}
	return strings.ReplaceAll(s, "_", "__")
}

func (t *trayMenu) action(project, action string) func(*application.Context) {
	return t.post("/api/projects/" + url.PathEscape(project) + "/actions/" + action)
}

func (t *trayMenu) global(action string) func(*application.Context) {
	return t.post("/api/actions/" + action)
}

// confirm asks before it runs then.
func (t *trayMenu) confirm(title, message, okLabel string, then func(*application.Context)) func(*application.Context) {
	return func(ctx *application.Context) {
		d := t.app.Dialog.Question().SetTitle(title).SetMessage(message)
		ok := d.AddButton(okLabel)
		cancel := d.AddButton("Cancel")
		d.SetDefaultButton(cancel)
		d.SetCancelButton(cancel)
		ok.OnClick(func() { go then(ctx) })
		showQuestion(d, t.window, t.show)
	}
}

// showQuestion shows d on the window, which comes forward first. On macOS a
// question on its own blocks the whole app, the tray included, even once it
// hides behind another app; on the window it blocks only the window.
func showQuestion(d *application.MessageDialog, window *application.WebviewWindow, show func()) {
	if runtime.GOOS == "darwin" && window != nil {
		show()
		d.AttachToWindow(window)
	}
	d.Show()
}

func (t *trayMenu) post(path string) func(*application.Context) {
	return func(*application.Context) {
		if t.srv == nil {
			return
		}
		if err := t.srv.Post(path); err != nil {
			t.app.Dialog.Error().SetTitle("DDEV").SetMessage(err.Error()).Show()
		}
	}
}

func statusMark(status string) string {
	switch status {
	case "running":
		return "●"
	case "paused", "stopped":
		return "○"
	}
	return "◌"
}

// quitter keeps a quit from killing running jobs unasked: Shutdown kills
// them, which can leave a project half started or a database half restored.
type quitter struct {
	app    *application.App
	srv    *serveClient
	window *application.WebviewWindow
	show   func()
	// force is set once the user has agreed, or a signal asked to quit.
	force atomic.Bool
	// asking is set while the question is shown.
	asking atomic.Bool
}

// shouldQuit is Wails' ShouldQuit. It runs on the main thread, so it shows
// its question from another goroutine and refuses this quit; the answer
// quits again.
func (q *quitter) shouldQuit() bool {
	var running []string
	if q.srv != nil && !q.force.Load() {
		running = q.srv.RunningJobs()
	}
	if len(running) == 0 {
		return true
	}
	if !q.asking.Swap(true) {
		go q.ask(running)
	}
	return false
}

func (q *quitter) ask(running []string) {
	// GTK shows nothing for a dialog built off the main thread, so this goes
	// back to it. shouldQuit has returned by the time the main loop runs it.
	application.InvokeAsync(func() {
		d := q.app.Dialog.Question().
			SetTitle("DDEV is still working").
			SetMessage("Quitting stops these, which can leave a project half done:\n\n" + strings.Join(running, "\n"))
		quit := d.AddButton("Quit anyway")
		wait := d.AddButton("Keep running")
		d.SetDefaultButton(wait)
		d.SetCancelButton(wait)
		quit.OnClick(func() {
			q.asking.Store(false)
			q.force.Store(true)
			go q.app.Quit()
		})
		wait.OnClick(func() { q.asking.Store(false) })
		showQuestion(d, q.window, q.show)
	})
}

// handleSignals quits without asking on SIGINT or SIGTERM, as from a
// terminal or the system, and exits at once on a second signal.
func (q *quitter) handleSignals() {
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		q.force.Store(true)
		go q.app.Quit()
		<-sigs
		os.Exit(1)
	}()
}
