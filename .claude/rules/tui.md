---
paths:
  - "pkg/tui/**"
---

# Check TUI changes in tmux

For interactive behavior or layout changes, run the built binary's `ddev tui`
command in tmux (`send-keys`, `capture-pane`) against a test project, with
`DDEV_XDG_CONFIG_HOME` pointing to an isolated directory. Register only the
test project there. Report if tmux or Docker is unavailable.

For actions that launch a browser, put a logging fake of the platform's browser
opener (`xdg-open` on Linux, `open` on macOS) first on PATH inside the tmux
command, and check its log. Avoid launching the user's browser.
