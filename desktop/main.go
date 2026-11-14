// Command ddev-desktop is an experimental desktop app for DDEV. It shows the
// `ddev serve` dashboard in a native window through Wails, and adds a tray
// menu and notifications. On macOS and Linux it opens no network port; see
// modes.go for Windows.
//
// It is a separate module so that Wails, and the CGO it needs on macOS and
// Linux, stay out of the ddev binary and its vendor directory.
package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/ddev/ddev/pkg/webui"
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
	ddevBinary, findErr := findDdev()
	if findErr != nil {
		log.Print(findErr)
	}

	m, err := chooseMode()
	if err != nil {
		log.Fatal(err)
	}
	var st *selfTest
	if os.Getenv("DDEV_DESKTOP_SELFTEST") == "true" {
		if m == modeWSL {
			log.Fatal("The self-test runs in embedded or loopback mode")
		}
		if st, err = newSelfTest(); err != nil {
			log.Fatal(err)
		}
	}

	globalconfig.EnsureGlobalConfig()
	_ = os.Setenv("DOCKER_CLI_HINTS", "false")
	_ = globalconfig.GetMutagenDataDirectory()
	// Hooks would print to nowhere; the subcommands still run theirs.
	ddevapp.SkipHooks = true

	var wrap func(http.Handler) http.Handler
	if st != nil || os.Getenv("DDEV_DESKTOP_DEBUG") == "true" {
		wrap = func(h http.Handler) http.Handler {
			if st != nil {
				h = st.wrap(h)
			}
			if os.Getenv("DDEV_DESKTOP_DEBUG") == "true" {
				h = logRequests(h)
			}
			return h
		}
	}

	// srv is the in-process dashboard; it is nil in WSL2 mode.
	var srv *webui.Server
	var be backend
	var remote *remoteServer
	handler := http.NotFoundHandler()
	windowURL := "/"
	switch m {
	case modeEmbedded:
		srv = webui.NewEmbedded(ddevBinary)
		handler = srv.Handler()
		if wrap != nil {
			handler = wrap(handler)
		}
	case modeLoopback:
		if srv, err = webui.New(0); err != nil {
			log.Fatal(err)
		}
		srv.SetDesktop(ddevBinary, wrap)
		windowURL = srv.URL()
	case modeWSL:
		if remote, err = startRemote(ddevBinary); err != nil {
			log.Fatal(err)
		}
		windowURL = remote.URL()
		be = remote
	}
	notifier := notifications.New()
	// notifyOn is set once the notifier has started; jobs may end before.
	var notifyOn atomic.Bool
	var window *application.WebviewWindow
	if srv != nil {
		be = srv
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
	}
	if m == modeLoopback {
		go func() {
			if err := srv.Serve(); err != nil {
				log.Print(err)
			}
		}()
	}
	shutdown := func() {
		if srv != nil {
			// Open status streams never go idle, so do not wait for them long.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = srv.Shutdown(ctx)
		}
		if remote != nil {
			remote.Close()
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
	})

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
		if ddevBinary == "" && m != modeWSL {
			app.Dialog.Error().SetTitle("ddev not found").SetMessage(findErr.Error()).Show()
		}
		// WSL2 mode reads and acts on projects with the one ddev inside WSL2.
		if ddevBinary != "" && m != modeWSL && st == nil {
			go warnVersionMismatch(app, ddevBinary)
		}
		if srv == nil {
			return
		}
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
	t := &trayMenu{app: app, tray: tray, srv: be, show: showWindow}
	go t.follow()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// runSelfTest waits for selftest.js to report, prints the result, and exits
// with its status. Wails' Quit ends the process without returning from Run.
func runSelfTest(srv *webui.Server, st *selfTest) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	st.cleanup()
	os.Exit(code)
}

// warnVersionMismatch warns when the ddev that actions run, or the ddev a
// terminal would run, is not the version this app was built from.
func warnVersionMismatch(app *application.App, ddevBinary string) {
	bins := []string{ddevBinary}
	if b, err := exec.LookPath(ddevName()); err == nil {
		bins = append(bins, b)
	}
	bad := versionMismatches(versionconstants.DdevVersion, bins)
	if len(bad) == 0 {
		return
	}
	msg := fmt.Sprintf("This app is DDEV %s, but:\n\n%s\n\nThe app can show projects wrongly, and two versions that both start a project keep rewriting its files. Use one DDEV version for the app and the terminal.", versionconstants.DdevVersion, strings.Join(bad, "\n"))
	log.Print(msg)
	application.InvokeAsync(func() {
		app.Dialog.Warning().SetTitle("DDEV versions differ").SetMessage(msg).Show()
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

// trayMenu keeps the tray menu in step with the projects.
type trayMenu struct {
	app  *application.App
	tray *application.SystemTray
	srv  backend
	show func()
	last []webui.ProjectState
	err  string
}

func (t *trayMenu) follow() {
	first := true
	for {
		changed := t.srv.Changed()
		projects, err := t.srv.Projects()
		errText := ""
		if err != nil {
			errText = err.Error()
		}
		if first || errText != t.err || !slices.Equal(projects, t.last) {
			first, t.last, t.err = false, projects, errText
			menu := t.build(projects, errText)
			application.InvokeAsync(func() { t.tray.SetMenu(menu) })
		}
		select {
		case <-changed:
		case <-time.After(5 * time.Second):
		}
	}
}

func (t *trayMenu) build(projects []webui.ProjectState, errText string) *application.Menu {
	m := t.app.NewMenu()
	m.Add("Open DDEV").OnClick(func(*application.Context) { t.show() })
	m.AddSeparator()
	if errText != "" {
		m.Add(errText).SetEnabled(false)
	} else if len(projects) == 0 {
		m.Add("No projects").SetEnabled(false)
	}
	running := 0
	for _, p := range projects {
		sub := m.AddSubmenu(statusMark(p.Status) + " " + p.Name)
		if p.Status == ddevapp.SiteRunning {
			running++
			site := p.URL
			sub.Add("Open site").OnClick(func(*application.Context) { _ = t.app.Browser.OpenURL(site) })
			sub.Add("Restart").OnClick(t.action(p.Name, "restart"))
			sub.Add("Stop").OnClick(t.action(p.Name, "stop"))
		} else {
			sub.Add("Start").OnClick(t.action(p.Name, "start"))
		}
	}
	m.AddSeparator()
	m.Add("Start all").SetEnabled(len(projects) > running).OnClick(t.global("start-all"))
	m.Add("Stop all").SetEnabled(running > 0).OnClick(t.global("stop-all"))
	m.Add("Power off").OnClick(t.global("poweroff"))
	m.AddSeparator()
	m.Add("Quit DDEV app").OnClick(func(*application.Context) { t.app.Quit() })
	return m
}

func (t *trayMenu) action(project, action string) func(*application.Context) {
	return t.post("/api/projects/" + url.PathEscape(project) + "/actions/" + action)
}

func (t *trayMenu) global(action string) func(*application.Context) {
	return t.post("/api/actions/" + action)
}

func (t *trayMenu) post(path string) func(*application.Context) {
	return func(*application.Context) {
		if err := t.srv.Post(path); err != nil {
			t.app.Dialog.Error().SetTitle("DDEV").SetMessage(err.Error()).Show()
		}
	}
}

func statusMark(status string) string {
	switch status {
	case ddevapp.SiteRunning:
		return "●"
	case ddevapp.SitePaused, ddevapp.SiteStopped:
		return "○"
	}
	return "◌"
}
