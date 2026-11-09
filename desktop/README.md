# Experimental DDEV desktop app

This is an experimental desktop app for DDEV, built with
[Wails v3](https://v3.wails.io/). It shows the dashboard from `ddev serve`
(`pkg/webui`) in a native window, and adds what a browser tab cannot:

- A tray menu, also called a menu bar extra on macOS. It lists each project
  with its status, starts, stops, or restarts it, and opens its site. It also
  has Start all, Stop all, and Power off.
- A Dock or taskbar icon. Closing the window keeps the app in the tray.
  Launching the app again shows the window, which is the way back on a
  desktop without a tray, such as GNOME without the AppIndicator extension.
- A native notification when an action ends while the window is in the
  background. Notifications need the app bundle on macOS and a session bus on
  Linux; without them the app runs without notifications.

## How it works

The window reaches the dashboard in one of three modes. The app picks one,
and `DDEV_DESKTOP_MODE` overrides it:

| Mode | Default on | How the window reaches the dashboard |
| --- | --- | --- |
| `embedded` | macOS, Linux | Wails passes each request from the webview to the handler of `webui.NewEmbedded`. No network port is open, so no token is needed. |
| `wsl` | Windows, when `ddev` is installed in WSL2 | The app runs `ddev serve --desktop` in WSL2 and loads its URL, which WSL2 forwards to Windows. The tray reaches it with the token. |
| `loopback` | Windows otherwise | The app serves the dashboard on `127.0.0.1` with a token, as `ddev serve` does, and loads its URL |

Windows needs a URL because Wails' WebView2 response writer cannot stream,
so the status stream and live logs would not update in `embedded` mode.
`ddev serve --desktop` stops when its standard input closes, so it does not
outlive the app inside WSL2.

In `embedded` mode a state change must still carry the `X-DDEV-Dashboard`
header. A page from another origin cannot add that header without a CORS
preflight, which the handler never answers. The other modes use the token
checks of `ddev serve`.

Actions run as `ddev` subcommands, as they do in `ddev serve`. The app runs
the `ddev` next to its own executable, which the macOS app bundle includes.
Without one, it uses `$DDEV_BINARY`, or the first `ddev` on the `PATH` of
your login shell.

This is a separate Go module, so that Wails stays out of the `ddev` binary
and `vendor/`. Wails needs CGO on macOS and Linux, but not on Windows, so the
Windows app cross-builds anywhere.

## Build and run

`make desktop` builds `ddev` for this OS, then the app next to it in
`.gotmp/bin/<os>_<arch>/`. The app runs the `ddev` next to it, so the window
and its actions use one version. It warns at startup when the `ddev` it runs,
or the first `ddev` on `PATH`, is another version.

### macOS

Building needs the Xcode command-line tools.

```bash
make desktop
open .gotmp/bin/darwin_arm64/DDEV.app
```

`DDEV.app` carries its own `ddev` and `ddev-hostname`, and `DDEV.app.zip`
next to it is the same bundle zipped. For an Intel Mac, run
`make darwin_amd64` and `make -C desktop GOARCH=amd64`.

The app shows a window and a DDEV icon in the menu bar. Closing the window
keeps the app running; click the Dock icon or launch the app again to get it
back. Quit from the menu bar icon.

### Linux

Building needs the GTK 4 and WebKitGTK 6 development packages, such as
`libgtk-4-dev` and `libwebkitgtk-6.0-dev` on Debian and Ubuntu:

```bash
make desktop
.gotmp/bin/linux_amd64/ddev-desktop &
```

Running needs glibc 2.38 or later, GTK 4, and WebKitGTK 6, as in Ubuntu
24.04 or Debian 13. On Ubuntu that is `libgtk-4-1` and `libwebkitgtk-6.0-4`.
The app is a single binary with no desktop launcher; start it from a
terminal.

The tray icon needs a StatusNotifierItem host. Ubuntu's GNOME and KDE Plasma
have one; GNOME elsewhere needs the AppIndicator extension.
Without a tray, launch the app again to show a closed window.

### Windows

The app needs no CGO on Windows, so it cross-builds from any OS:

```bash
make -C desktop GOOS=windows GOARCH=amd64 binary
# .gotmp/bin/windows_amd64/ddev-desktop.exe
```

Running needs the WebView2 runtime, which Windows 11 includes. The app picks
one of two modes:

- With `ddev` installed in WSL2, it runs `ddev serve --desktop` there through
  `wsl.exe -e bash -lc`, and the window shows your WSL2 projects. That `ddev`
  must be on the PATH of a bash login shell, and it needs `ddev serve`, which
  no release has yet.
- Otherwise it serves the dashboard itself on `127.0.0.1`, and runs the
  `ddev.exe` next to `ddev-desktop.exe`, or the first one on `PATH`.

Set `DDEV_BINARY` to choose the `ddev` the app runs, and
`DDEV_DESKTOP_DEBUG=true` to log each request the window makes.

## Tests

- `go test ./...` in `desktop/` covers how the app finds `ddev` and picks a
  mode. With `DDEV_BINARY` set, it also drives the code path of `wsl` mode
  against a local `ddev serve`.
- `DDEV_DESKTOP_SELFTEST=true` runs `selftest.js` inside the app's own
  window, then exits with its result. The script waits for the status stream,
  submits a missing directory to "Add project" and expects an error, then
  runs `ddev config` in an empty directory and waits for the job to finish.
  The app keeps that project in a temporary global configuration, which it
  deletes at the end. It needs no Docker. It runs in `embedded` and
  `loopback` modes. On Linux, run it under `xvfb-run` when there is no
  display.
- `make webui-e2e` runs the dashboard's Playwright tests in Chromium and in
  WebKit, the engine of the app's window on macOS and Linux.

```bash
DDEV_DESKTOP_SELFTEST=true .gotmp/bin/darwin_arm64/DDEV.app/Contents/MacOS/ddev-desktop
# Expect "SELFTEST PASS" and exit status 0
```

The tray menu, notifications, and hiding the window to the tray have no
automated test.

## CI and signing

| Workflow | Builds | Signing |
| --- | --- | --- |
| `pr-build.yml` | `DDEV.app` for macOS arm64 and amd64, Linux amd64 and arm64, and Windows amd64 and arm64, each as a `ddev-desktop-*` artifact listed in the PR's download comment. It runs the Go tests and the self-test on macOS, Linux, and Windows. | None. macOS blocks an unsigned download, so run `xattr -dr com.apple.quarantine DDEV.app` after unzipping. |
| `main-build.yml` | The same apps, as workflow artifacts only | `make -C desktop sign notarize` signs the bundled `ddev` and the bundle with DDEV's Developer ID, then notarizes and staples it. `signtool` on the Windows signing runner signs the Windows apps. |

These jobs may fail without failing their workflow, so the app cannot hold
up a DDEV release or the PR download comment.

## Known limits

- **Windows:** nobody has run the app on Windows by hand. The tray and
  notifications in `wsl` mode are untested, and `wsl` mode sends no
  notifications.
- **Releases:** the app is not part of any release or package.
- **Versions:** the Linux and Windows downloads hold the app alone. Put it
  next to a `ddev` of the same version, such as the one from
  `ddev utility download-ddev --pr N`. Otherwise it runs the `ddev` on `PATH`
  and warns when that is another version.
