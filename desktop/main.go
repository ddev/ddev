// Command ddev-desktop is an experimental desktop app for DDEV. It shows the
// `ddev serve` dashboard in a native window through Wails, and adds a tray
// menu and notifications.
//
// It holds no DDEV code: it runs `ddev serve --desktop` from the ddev it
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

//go:embed icons/tray.png
var trayIcon []byte

// version is set by the Makefile.
var version = "dev"

func main() {
	if os.Geteuid() == 0 {
		log.Fatal("DDEV is not designed to run as root")
	}
	shareHiddenConsole()
	useLoginShellPath()

	var st *selfTest
	if os.Getenv("DDEV_DESKTOP_SELFTEST") == "true" {
		var err error
		if st, err = newSelfTest(); err != nil {
			log.Fatal(err)
		}
	}

	// startErr is shown in the window and a dialog when ddev serve cannot
	// run, such as when ddev is missing or too old to have `ddev serve`.
	var startErr error
	var srv *serveClient
	tgt, err := chooseTarget(argValue(os.Args[1:], "--wsl-distro"))
	if err == nil {
		log.Printf("Running ddev serve from %s", tgt)
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
	// and Linux it passes the window's requests to ddev serve. WebView2 on
	// Windows cannot stream a response through it, so there the window
	// loads ddev serve's URL instead.
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
	if os.Getenv("DDEV_DESKTOP_DEBUG") == "true" {
		handler = logRequests(handler)
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
		window.Show()
		window.Focus()
	}
	singleInstance := &application.SingleInstanceOptions{
		UniqueID: "com.ddev.desktop",
		// Launching the app again is how a closed window comes back where
		// there is no tray or Dock icon, such as GNOME without AppIndicator.
		OnSecondInstanceLaunch: func(application.SecondInstanceData) {
			application.InvokeAsync(showWindow)
		},
	}
	if st != nil {
		// A self-test runs beside the user's own copy of the app.
		singleInstance = nil
	}

	// quitter asks before a quit would kill running jobs.
	q := &quitter{srv: srv}
	app := application.New(application.Options{
		Name:        "DDEV",
		Description: "Experimental DDEV desktop app " + version,
		Icon:        appIcon,
		Assets: application.AssetOptions{
			Handler:        handler,
			DisableLogging: true,
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		SingleInstance: singleInstance,
		OnShutdown:     shutdown,
		ShouldQuit:     q.shouldQuit,
		// Wails' own handler reads only the first signal, so a quit that
		// ShouldQuit refused would leave the app deaf to the next one.
		DisableDefaultSignalHandler: true,
	})
	q.app = app
	q.handleSignals()

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     "DDEV",
		Width:     1280,
		Height:    820,
		MinWidth:  720,
		MinHeight: 480,
		URL:       windowURL,
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
			go runSelfTest(srv, st)
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
				_, _ = notifier.RequestNotificationAuthorization()
			}
		}
		notifyOn.Store(true)
	})

	tray := app.SystemTray.New()
	tray.SetTemplateIcon(trayIcon)
	tray.SetTooltip("DDEV")
	// pageURL is the dashboard page once loaded, without the launch token.
	pageURL := "/"
	if u, err := url.Parse(windowURL); err == nil && u.Host != "" {
		pageURL = u.Scheme + "://" + u.Host + "/"
	}
	t := &trayMenu{app: app, tray: tray, srv: srv, startErr: startErr, show: showWindow, reveal: func(name string) {
		// The page selects the project that its location's hash names.
		// A change to the hash alone does not reload the page. ExecJS would
		// wait for the Wails runtime, which the page does not load.
		window.SetURL(pageURL + "#project=" + url.QueryEscape(name))
		showWindow()
	}}
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

// watchExit tells the user when ddev serve stops while the app runs, since
// the window and the tray need it.
func watchExit(app *application.App, srv *serveClient, q *quitter) {
	<-srv.Exited()
	if srv.Closing() {
		return
	}
	msg := "ddev serve stopped, so DDEV must quit."
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
func runSelfTest(srv *serveClient, st *selfTest) {
	code := 1
	select {
	case res := <-st.result:
		res = st.check(res)
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
	app  *application.App
	tray *application.SystemTray
	// srv is nil when ddev serve could not start, and startErr says why.
	srv      *serveClient
	startErr error
	show     func()
	// reveal shows the window with a project selected.
	reveal  func(name string)
	last    []project
	err     string
	version string
}

func (t *trayMenu) follow() {
	first := true
	for {
		var changed <-chan struct{}
		var projects []project
		var err error
		ver := ""
		if t.srv != nil {
			changed = t.srv.Changed()
			projects, err = t.srv.Projects()
			ver = t.srv.Version()
		} else {
			err = t.startErr
		}
		errText := ""
		if err != nil {
			errText = err.Error()
		}
		if first || errText != t.err || ver != t.version || !slices.Equal(projects, t.last) {
			first, t.last, t.err, t.version = false, projects, errText, ver
			menu := t.build(projects, errText, ver)
			application.InvokeAsync(func() { t.tray.SetMenu(menu) })
		}
		select {
		case <-changed:
		case <-time.After(5 * time.Second):
		}
	}
}

func (t *trayMenu) build(projects []project, errText, ver string) *application.Menu {
	m := t.app.NewMenu()
	m.Add("Open DDEV").OnClick(func(*application.Context) { t.show() })
	m.AddSeparator()
	if errText != "" {
		// A menu item shows one line.
		errText, _, _ = strings.Cut(errText, "\n")
		m.Add(errText).SetEnabled(false)
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
		about := t.srv.target.String()
		if ver != "" {
			about = "DDEV " + ver + " · " + about
		}
		m.Add(about).SetEnabled(false)
	}
	m.Add("Quit DDEV app").OnClick(func(*application.Context) { t.app.Quit() })
	return m
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
		d.Show()
	}
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
	app *application.App
	srv *serveClient
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
	d.Show()
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
