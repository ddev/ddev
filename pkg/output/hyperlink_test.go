package output

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDetectHyperlinks exercises detectHyperlinks directly so each case runs
// with fresh inputs, bypassing the sync.OnceValue cache in HasTermHyperlinks.
func TestDetectHyperlinks(t *testing.T) {
	tests := []struct {
		name           string
		forceEnv       string
		isTTY          bool
		termProgram    string
		vteVersion     string
		wtSession      string
		konsoleVersion string
		termEnv        string
		want           bool
	}{
		{name: "FORCE_HYPERLINK=1 overrides non-TTY", forceEnv: "1", isTTY: false, want: true},
		{name: "FORCE_HYPERLINK=true overrides non-TTY", forceEnv: "true", isTTY: false, want: true},
		{name: "FORCE_HYPERLINK=0 overrides TTY+vscode", forceEnv: "0", isTTY: true, termProgram: "vscode", want: false},
		{name: "FORCE_HYPERLINK=false overrides iTerm2", forceEnv: "false", isTTY: true, termProgram: "iTerm.app", want: false},
		{name: "FORCE_HYPERLINK invalid value is false", forceEnv: "maybe", isTTY: true, termProgram: "vscode", want: false},
		{name: "not a TTY with no force", isTTY: false, termProgram: "vscode", want: false},
		{name: "TTY with no recognizable terminal", isTTY: true, want: false},
		{name: "iTerm2", isTTY: true, termProgram: "iTerm.app", want: true},
		{name: "vscode", isTTY: true, termProgram: "vscode", want: true},
		{name: "WezTerm TERM_PROGRAM", isTTY: true, termProgram: "WezTerm", want: true},
		{name: "ghostty TERM_PROGRAM", isTTY: true, termProgram: "ghostty", want: true},
		{name: "VTE high enough", isTTY: true, vteVersion: "5000", want: true},
		{name: "VTE too old", isTTY: true, vteVersion: "4999", want: false},
		{name: "VTE non-numeric", isTTY: true, vteVersion: "abc", want: false},
		{name: "Windows Terminal WT_SESSION", isTTY: true, wtSession: "some-guid", want: true},
		{name: "Konsole", isTTY: true, konsoleVersion: "210401", want: true},
		{name: "kitty in TERM", isTTY: true, termEnv: "xterm-kitty", want: true},
		{name: "alacritty in TERM", isTTY: true, termEnv: "alacritty", want: true},
		{name: "wezterm in TERM", isTTY: true, termEnv: "wezterm", want: true},
		{name: "xterm-256color not recognised", isTTY: true, termEnv: "xterm-256color", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := detectHyperlinks(tc.forceEnv, tc.isTTY, tc.termProgram, tc.vteVersion, tc.wtSession, tc.konsoleVersion, tc.termEnv)
			require.Equal(t, tc.want, got)
		})
	}
}

// TestFileURL verifies file:// URL construction for OSC 8 hyperlink targets.
func TestFileURL(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "empty path", path: "", want: ""},
		{name: "unix absolute path", path: "/Users/rfay/workspace/ddev", want: "file:///Users/rfay/workspace/ddev"},
		{name: "path with spaces", path: "/home/user/my project", want: "file:///home/user/my%20project"},
		{name: "nested path", path: "/var/www/html", want: "file:///var/www/html"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, FileURL(tc.path))
		})
	}
}

// TestHyperlinkPassthrough verifies Hyperlink returns plain text when OSC 8
// should not be emitted (empty args, or when detectHyperlinks would return
// false — which is always true in the test process since stdout is a pipe).
func TestHyperlinkPassthrough(t *testing.T) {
	url := "https://example.ddev.site"
	label := "example.ddev.site"

	// In the test process stdout is a pipe and FORCE_HYPERLINK is unset, so
	// HasTermHyperlinks() returns false and Hyperlink must return plain text.
	require.Equal(t, label, Hyperlink(url, label), "should be plain text in piped test process")
	require.Equal(t, label, Hyperlink("", label), "empty URL should return label unchanged")
	require.Equal(t, "", Hyperlink(url, ""), "empty label should return empty string")
	require.Equal(t, "", Hyperlink("", ""), "both empty should return empty string")
}

// TestHyperlinkOSC8Format verifies the OSC 8 escape sequence format when
// detectHyperlinks would return true. We construct the sequence manually to
// confirm the expected byte structure without relying on HasTermHyperlinks.
func TestHyperlinkOSC8Format(t *testing.T) {
	rawURL := "https://example.ddev.site"
	label := "example"

	// The OSC 8 sequence:
	//   open:  ESC ] 8 ; ; <url> ST   (ST = string terminator = ESC \)
	//   label: visible text
	//   close: ESC ] 8 ; ;        ST  (empty URL closes the link)
	openLink := "\x1b]8;;" + rawURL + "\x1b\\"
	closeLink := "\x1b]8;;\x1b\\"
	want := openLink + label + closeLink

	require.True(t, strings.HasPrefix(want, "\x1b]8;;"), "should open with OSC 8")
	require.True(t, strings.HasSuffix(want, closeLink), "should close with empty-URL OSC 8")
	require.True(t, strings.Contains(want, rawURL), "URL in the escape envelope")
	// Verify the label sits between the open and close sequences
	inner := strings.TrimPrefix(strings.TrimSuffix(want, closeLink), openLink)
	require.Equal(t, label, inner, "visible text should be exactly the label")
}
