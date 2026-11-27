package main

import (
	"net/url"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// trayMenu keeps the tray menu in step with the projects.
type trayMenu struct {
	app  *application.App
	tray *application.SystemTray
	// srv is nil when ddev gui could not start, and startErr says why.
	srv      *serveClient
	startErr error
	show     func()
	icons    *trayIcons
	// navigate shows the window and a target, such as "about".
	navigate func(target string)
	last     []project
	err      string
}

// startTray adds the tray icon and its menu. toggle runs on a click.
func startTray(t *trayMenu, toggle func()) {
	t.tray = t.app.SystemTray.New()
	t.icons = newTrayIcons(t.tray, runtime.GOOS)
	if runtime.GOOS == "linux" {
		// Without a label, Wails names the item "Wails". The hover text shows
		// the label, so a tooltip would repeat it. macOS shows a label as text.
		t.tray.SetLabel("DDEV")
	} else {
		t.tray.SetTooltip("DDEV")
	}
	// Wails opens the menu on a right-click only if the tray has a menu when
	// the app starts, and follow's first menu arrives later.
	firstErr := "Connecting to ddev gui…"
	if t.startErr != nil {
		firstErr = t.startErr.Error()
	}
	t.tray.SetMenu(t.build(nil, firstErr))
	if runtime.GOOS != "darwin" {
		// As other Windows and Linux tray apps do, a click shows or hides the
		// window. A macOS menu bar extra opens its menu.
		t.tray.OnClick(toggle)
	}
	go t.follow()
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
		sub.Add("Show in DDEV").OnClick(t.open("project=" + url.QueryEscape(name)))
	}
	m.AddSeparator()
	m.Add("Start all").SetEnabled(len(projects) > running).OnClick(t.global("start-all"))
	// The window asks before these.
	m.Add("Stop all").SetEnabled(running > 0).OnClick(t.open("stop-all"))
	m.Add("Power off").OnClick(t.open("poweroff"))
	m.AddSeparator()
	if t.srv != nil {
		m.Add("About DDEV").OnClick(t.open("about"))
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

func (t *trayMenu) open(target string) func(*application.Context) {
	return func(*application.Context) { t.navigate(target) }
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
