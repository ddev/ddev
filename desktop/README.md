# Experimental DDEV desktop app

This is an experimental desktop app for DDEV, built with
[Wails v3](https://v3.wails.io/). It shows the dashboard from `ddev serve`
(`pkg/webui`) in a native window, and adds what a browser tab cannot:

- A tray menu, also called a menu bar extra on macOS. It lists each project
  with its status, starts, stops, or restarts it, and opens its site. It also
  has Start all, Stop all, and Power off.
- A Dock or taskbar icon. Closing the window keeps the app in the tray.
- A native notification when an action ends while the window is in the
  background. Notifications need the app bundle on macOS and a session bus on
  Linux; without them the app runs without notifications.

## How it works

The window does not load a URL from a server. Wails passes each request from
the webview to the `http.Handler` of `webui.NewEmbedded`, so the app opens no
network port and needs no access token. A state change must still carry the
`X-DDEV-Dashboard` header. A page from another origin cannot add that header
without a CORS preflight, which the handler never answers.

Actions run as `ddev` subcommands, as they do in `ddev serve`. The app runs
the `ddev` next to its own executable, which the macOS app bundle includes.
Without one, it uses `$DDEV_BINARY`, or the first `ddev` on the `PATH` of
your login shell.

This is a separate Go module, so that Wails stays out of the `ddev` binary
and `vendor/`. Wails needs CGO on macOS and Linux.

## Build and run

On macOS:

```bash
make desktop
open .gotmp/bin/darwin_arm64/DDEV.app
```

On Linux, install the GTK 4 and WebKitGTK 6 development packages first, such
as `libgtk-4-dev` and `libwebkitgtk-6.0-dev` on Debian and Ubuntu:

```bash
make desktop
.gotmp/bin/linux_amd64/ddev-desktop
```

Set `DDEV_DESKTOP_DEBUG=true` to log each request the window makes.

## Tests

- `go test ./...` in `desktop/` covers how the app finds `ddev`.
- `DDEV_DESKTOP_SELFTEST=true` runs `selftest.js` inside the app's own
  window, then exits with its result. The script waits for the status stream,
  submits a missing directory to "Add project" and expects an error, then
  runs `ddev config` in an empty directory and waits for the job to finish.
  The app keeps that project in a temporary global configuration, which it
  deletes at the end. It needs no Docker. On Linux, run it under `xvfb-run`
  when there is no display.
- `make webui-e2e` runs the dashboard's Playwright tests in Chromium and in
  WebKit, the engine of the app's window on macOS and Linux.

```bash
DDEV_DESKTOP_SELFTEST=true .gotmp/bin/darwin_arm64/DDEV.app/Contents/MacOS/ddev-desktop
# Expect "SELFTEST PASS" and exit status 0
```

The tray menu, notifications, and hiding the window to the tray have no
automated test.

## Known limits

- **Windows:** Wails' WebView2 response writer cannot stream, so the status
  stream and live logs would not update. On Windows the window should load a
  `ddev serve` URL instead, which also suits WSL2, where DDEV runs.
- **Signing:** the macOS bundle is signed ad hoc. A release would need a
  Developer ID signature, notarization, and per-OS packages.
- **Versions:** the bundle carries its own `ddev`. A copy of the app that runs
  a different `ddev` from `PATH` can read projects with one version and act
  on them with another.
