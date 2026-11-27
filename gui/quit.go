package main

import (
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/wailsapp/wails/v3/pkg/application"
)

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
