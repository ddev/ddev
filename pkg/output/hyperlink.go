package output

import (
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mattn/go-isatty"
)

// HasTermHyperlinks reports whether the terminal likely renders OSC 8
// hyperlinks. Terminals that don't support OSC 8 are expected to silently
// ignore the escape sequences, but detection avoids emitting them where
// they are known to misbehave (e.g. GNU screen) or are useless (pipes).
// FORCE_HYPERLINK=1/0 overrides detection, following the convention used
// by eza, lsd, and the chalk/supports-hyperlinks ecosystem.
var HasTermHyperlinks = sync.OnceValue(func() bool {
	if v := os.Getenv("FORCE_HYPERLINK"); v != "" {
		enabled, err := strconv.ParseBool(v)
		return err == nil && enabled
	}
	if JSONOutput || !isatty.IsTerminal(os.Stdout.Fd()) {
		return false
	}
	switch os.Getenv("TERM_PROGRAM") {
	case "Hyper", "ghostty", "iTerm.app", "mintty", "rio", "Tabby", "vscode", "WezTerm":
		return true
	}
	// VTE-based terminals (GNOME Terminal, Tilix, etc.) support OSC 8 since 0.50
	if v, err := strconv.Atoi(os.Getenv("VTE_VERSION")); err == nil && v >= 5000 {
		return true
	}
	// Windows Terminal and Konsole don't set TERM_PROGRAM
	if os.Getenv("WT_SESSION") != "" || os.Getenv("KONSOLE_VERSION") != "" {
		return true
	}
	term := os.Getenv("TERM")
	for _, t := range []string{"alacritty", "contour", "foot", "ghostty", "kitty", "wezterm"} {
		if strings.Contains(term, t) {
			return true
		}
	}
	return false
})

// Hyperlink returns linkText wrapped in an OSC 8 terminal hyperlink pointing
// at targetURL when the terminal supports it; otherwise it returns linkText
// unchanged. The link target is invisible, so linkText may later be truncated
// for display while the full targetURL remains clickable.
func Hyperlink(targetURL string, linkText string) string {
	if targetURL == "" || linkText == "" || !HasTermHyperlinks() {
		return linkText
	}
	return text.Hyperlink(targetURL, linkText)
}

// FileURL converts an absolute filesystem path into a file:// URL usable as
// an OSC 8 hyperlink target, e.g. to open a project directory.
func FileURL(path string) string {
	if path == "" {
		return ""
	}
	p := filepath.ToSlash(path)
	// Windows paths like C:/Users/... need a leading slash in the URL path
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	u := url.URL{Scheme: "file", Path: p}
	return u.String()
}
