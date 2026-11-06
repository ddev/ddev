package webui

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ddev/ddev/pkg/nodeps"
)

// openTarget hands a directory or URL to the OS opener, the way `ddev tui`
// does. The browser cannot open local folders or editors itself.
func openTarget(target string) error {
	switch {
	case nodeps.IsMacOS():
		return exec.Command("open", target).Run()
	case nodeps.IsWindows():
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Run()
	case nodeps.IsWSL2():
		// xdg-open before xdg-utils 1.2 has no WSL support.
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
	if _, err := exec.LookPath("xdg-open"); err != nil {
		return fmt.Errorf("no opener found (install xdg-utils)")
	}
	return exec.Command("xdg-open", target).Run()
}

// OpenBrowser opens u in the default browser.
func OpenBrowser(u string) error {
	return openTarget(u)
}

// editorURL is the URL that opens dir in VS Code or PhpStorm, reaching into
// the WSL2 distro when there is one.
func editorURL(editor, dir string) string {
	p := filepath.ToSlash(dir)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	distro := os.Getenv("WSL_DISTRO_NAME")
	if editor == "vscode" {
		u := &url.URL{Scheme: "vscode", Host: "file", Path: p, RawQuery: "windowId=_blank"}
		if distro != "" {
			u.Host, u.Path = "vscode-remote", "/wsl+"+distro+p
		}
		return u.String()
	}
	if distro != "" {
		p = "//wsl.localhost/" + distro + p
	}
	// EscapedPath leaves & and + alone, but they would end the file= value
	return "phpstorm://open?file=" + strings.NewReplacer("&", "%26", "+", "%2B").Replace((&url.URL{Path: p}).EscapedPath())
}
