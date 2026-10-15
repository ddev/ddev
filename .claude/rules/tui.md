---
paths:
  - "pkg/tui/**"
---

# Check TUI changes in tmux

Tests don't show how a key behaves on a real screen, so run the built binary
in tmux (`send-keys`, `capture-pane`) against a test project, with
`DDEV_XDG_CONFIG_HOME` set so no real project is listed.

Never let it open the user's browser: start the session with
`env PATH=<dir with a logging fake xdg-open>:$PATH ddev`, since tmux ignores
`-e PATH=`, and check the fake's log.
