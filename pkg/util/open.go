package util

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ddev/ddev/pkg/nodeps"
)

// OpenTarget hands a directory or URL to the OS opener. Editors are reached
// through their URL schemes, from EditorURL, so no editor command has to be
// on PATH.
func OpenTarget(target string) error {
	switch {
	case nodeps.IsMacOS():
		return exec.Command("open", target).Run()
	case nodeps.IsWindows():
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Run()
	case nodeps.IsWSL2():
		return openOnWSL(target)
	}
	if !IsCommandAvailable("xdg-open") {
		return fmt.Errorf("no opener found (install xdg-utils)")
	}
	return exec.Command("xdg-open", target).Run()
}

// openOnWSL opens target with Windows tools directly, since xdg-open's WSL
// support is missing before xdg-utils 1.2 and reports failure whenever it
// opens a directory.
func openOnWSL(target string) error {
	if strings.Contains(target, "://") {
		return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", target).Run()
	}
	winPath, err := exec.Command("wslpath", "-w", target).Output()
	if err != nil {
		return fmt.Errorf("wslpath -w %s: %w", target, err)
	}
	// explorer.exe exits non-zero even on success
	_ = exec.Command("explorer.exe", strings.TrimSpace(string(winPath))).Run()
	return nil
}

// OpenInPhpStorm opens dir with a PhpStorm command-line launcher when one is
// on PATH, since PhpStorm does not register phpstorm:// on Windows.
func OpenInPhpStorm(dir string) error {
	var launcher string
	// Toolbox names it phpstorm on Linux and macOS, PhpStorm on Windows
	for _, name := range []string{"phpstorm", "PhpStorm", "phpstorm64.exe"} {
		if p, err := exec.LookPath(name); err == nil {
			launcher = p
			break
		}
	}
	if launcher == "" {
		return OpenTarget(EditorURL("phpstorm", dir))
	}
	if nodeps.IsWSL2() {
		winPath, err := exec.Command("wslpath", "-w", dir).Output()
		if err != nil {
			return fmt.Errorf("wslpath -w %s: %w", dir, err)
		}
		dir = strings.TrimSpace(string(winPath))
	}
	// The launcher may run as the IDE process itself, so don't wait for it
	c := exec.Command(launcher, dir)
	if err := c.Start(); err != nil {
		return err
	}
	go func() { _ = c.Wait() }()
	return nil
}

// EditorURL returns a URL that opens dir through an editor's registered URL
// scheme, reaching into the WSL2 distro when there is one, or "" for an
// unknown editor.
func EditorURL(editor, dir string) string {
	p := filepath.ToSlash(dir)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	distro := os.Getenv("WSL_DISTRO_NAME")
	switch editor {
	case "vscode":
		u := &url.URL{Scheme: "vscode", Host: "file", Path: p, RawQuery: "windowId=_blank"}
		if distro != "" {
			u.Host = "vscode-remote"
			u.Path = "/wsl+" + distro + p
		}
		return u.String()
	case "phpstorm":
		if distro != "" {
			p = "//wsl.localhost/" + distro + p
		}
		// EscapedPath leaves & and + alone, but they would end the file= value
		return "phpstorm://open?file=" + strings.NewReplacer("&", "%26", "+", "%2B").Replace((&url.URL{Path: p}).EscapedPath())
	}
	return ""
}
