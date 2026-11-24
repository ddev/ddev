# Experimental DDEV desktop app

This is an experimental desktop app for DDEV, built with
[Wails v3](https://v3.wails.io/). It shows the dashboard from `ddev serve`
(`pkg/webui`) in a native window, and adds what a browser tab cannot:

- A tray menu, also called a menu bar extra on macOS. It lists each project
  with its status, starts, stops, or restarts it, and opens its site. It also
  has Start all, Stop all, and Power off; the last two ask first, as the
  page does. Its last line names the DDEV version and the `ddev` it runs.
- A question before quitting while an action runs, since quitting stops the
  action and could leave a project half started. Ctrl+C or SIGTERM quits
  without asking.
- A Dock or taskbar icon. Closing the window keeps the app in the tray.
  Launching the app again shows the window, which is the way back on a
  desktop without a tray, such as GNOME without the AppIndicator extension.
- A native notification when an action ends while the window is in the
  background. Notifications need the app bundle on macOS and a session bus on
  Linux; without them the app runs without notifications.

## One ddev

The app holds no DDEV code and carries no `ddev` binary. It runs
`ddev serve --desktop` from the `ddev` you already have, shows that
dashboard, and drives the tray through the same HTTP API. The window, the
tray, and your terminal therefore read and change projects with one `ddev`,
of one version. Upgrading `ddev` upgrades what the app shows.

The app runs, in this order:

1. `$DDEV_BINARY`, when it is set.
2. On Windows, with no `--wsl-distro` argument, a `ddev.exe` next to
   `ddev-desktop.exe`, as a traditional Windows install puts it. Otherwise
   `ddev` in WSL2, through `wsl.exe -e bash -lc`, as the installer's WSL2
   choices put it. The installer's Start Menu shortcut passes
   `--wsl-distro <distro>` for the distro it installed DDEV in.
3. The `ddev` next to the app's executable, or next to `DDEV.app` on macOS.
   Packages install the app next to `ddev`.
4. The first `ddev` on the `PATH` of your login shell. An app started from
   the Finder or a launcher gets a minimal `PATH`, so the app reads `PATH`
   from the login shell.
5. Where DDEV's installers put it: `/opt/homebrew/bin` and `/usr/local/bin`
   on macOS, and `/usr/bin`, `/usr/local/bin`, and Homebrew's directory on
   Linux.

`DDEV_DESKTOP_MODE=native` or `wsl` overrides step 2.

`ddev serve --desktop` listens on `127.0.0.1` with a token, opens links in
the default browser, and stops when its standard input closes, so it stops
with the app, inside WSL2 too. On macOS and Linux the window reaches it
through the Wails asset server, which passes each request to `ddev serve`
with the token, so the window needs no token or cookie. A state change must
carry the `X-DDEV-Dashboard` header, which the page always sends and a page
from another origin cannot add without a CORS preflight. On Windows, Wails'
WebView2 response writer cannot stream, so the window loads the URL of
`ddev serve` instead, as a browser would.

The app needs a `ddev` with `ddev serve`. With an older one, the window and
a dialog say that `ddev serve --desktop` stopped, with what it printed.

This is a separate Go module, so that Wails stays out of the `ddev` binary
and `vendor/`. Wails needs CGO on macOS and Linux, but not on Windows, so the
Windows app cross-builds anywhere.

## Install

| OS | How | Where |
| --- | --- | --- |
| macOS | `brew install --cask ddev/ddev/ddev-desktop`, which also installs the `ddev` formula. Or download `ddev-desktop_macos-universal.v<version>.zip` from a release. | `/Applications/DDEV.app`, one universal app for Apple silicon and Intel |
| Windows | The DDEV installer, for every install type. Start it from the Start Menu: DDEV, then DDEV Desktop (experimental). | `%LOCALAPPDATA%\Programs\DDEV\ddev-desktop.exe` |
| Linux | `sudo apt install ddev-desktop` or `sudo dnf install ddev-desktop` from DDEV's package repository, which depends on `ddev`. The Linux release archives also hold `ddev-desktop`. | `/usr/bin/ddev-desktop`, with a launcher entry |

The Linux app needs glibc 2.38 or later, GTK 4, and WebKitGTK 6, as in
Ubuntu 24.04 or Debian 13. The tray icon needs a StatusNotifierItem host.
Ubuntu's GNOME and KDE Plasma have one; GNOME elsewhere needs the
AppIndicator extension.

On Windows the app needs the WebView2 runtime, which Windows 11 includes.

## Build and run

`make desktop` builds `ddev` for this OS, then the app:

- On macOS, a universal `DDEV.app` in `.gotmp/bin/darwin_universal/`, with
  `DDEV.app.zip` next to it, zipped by `ditto` to keep its signature. Building
  needs the Xcode command-line tools.
- On Linux, `ddev-desktop` next to `ddev` in `.gotmp/bin/linux_<arch>/`.
  Building needs the GTK 4 and WebKitGTK 6 development packages, such as
  `libgtk-4-dev` and `libwebkitgtk-6.0-dev` on Debian and Ubuntu.
- For Windows, from any OS, `make windows_amd64_desktop` puts
  `ddev-desktop.exe` next to `ddev.exe` in `.gotmp/bin/windows_amd64/`.
  `make windows_amd64_install` builds it into the installer.

The app runs the `ddev` it finds as described above, which for a build of
your own is often not the `ddev` you just built. Set `DDEV_BINARY`:

```bash
# macOS; open passes none of your shell's variables, --env passes one
open --env DDEV_BINARY="$PWD/.gotmp/bin/darwin_arm64/ddev" .gotmp/bin/darwin_universal/DDEV.app
# Or run it in the foreground, with its log in the terminal
DDEV_BINARY="$PWD/.gotmp/bin/darwin_arm64/ddev" .gotmp/bin/darwin_universal/DDEV.app/Contents/MacOS/ddev-desktop
# Quit it
osascript -e 'quit app "DDEV"'

# Linux; the app runs the ddev next to it
.gotmp/bin/linux_amd64/ddev-desktop &
pkill -x ddev-desktop
```

`open --env` has no effect when the app is already running.

From PowerShell on Windows:

```powershell
# Start it; it returns at once, since it is a GUI app
.\ddev-desktop.exe
# Use ddev in a particular WSL2 distro
.\ddev-desktop.exe --wsl-distro Ubuntu-24.04
# Quit it, or use Quit in the tray menu
Stop-Process -Name ddev-desktop
```

A GUI app prints nothing to the Windows console. To see its log, pipe it, as
`./ddev-desktop.exe | cat` in Git Bash.

To try a PR's build, `ddev utility download-ddev --pr <number>` downloads its
`ddev` and its desktop app into one directory, where the app finds that
`ddev`. On macOS that is `DDEV.app` next to `ddev`.

`DDEV_DESKTOP_DEBUG=true` logs each request the window makes.

## Tests

- `make -C desktop test` runs the Go tests with CGO on, which Wails needs
  on macOS and Linux. They cover how the app finds `ddev`, reads the status
  stream, notices jobs that end, and proxies the window's requests. With
  `DDEV_BINARY` set to a `ddev` with `ddev serve`, they also start a real
  `ddev serve --desktop` and drive it.
- `DDEV_DESKTOP_SELFTEST=true` runs `selftest.js` inside the app's own
  window, then exits with its result. The script waits for the status stream,
  submits a missing directory to "Add project" and expects an error, then
  runs `ddev config` in an empty directory and waits for the job to finish.
  The app keeps that project in a temporary global configuration, which it
  deletes at the end. It needs no Docker. On Linux, run it under `xvfb-run`
  when there is no display. On Windows the self-test's window loads the
  dashboard from a port of the app's own, since it must add its script to
  the page.
- `make webui-e2e` runs the dashboard's Playwright tests in Chromium and in
  WebKit, the engine of the app's window on macOS and Linux.

```bash
DDEV_DESKTOP_SELFTEST=true DDEV_BINARY="$PWD/.gotmp/bin/darwin_arm64/ddev" .gotmp/bin/darwin_universal/DDEV.app/Contents/MacOS/ddev-desktop
# Expect "SELFTEST PASS" and exit status 0
```

The tray menu, notifications, and hiding the window to the tray have no
automated test.

## CI, signing, and releases

| Workflow | macOS | Linux | Windows |
| --- | --- | --- | --- |
| `pr-build.yml` | `DDEV.app.zip`, unsigned, added to the `ddev-macos-arm64` and `ddev-macos-amd64` artifacts | `ddev-desktop` added to the `ddev-linux-*` artifacts | `ddev-desktop.exe` in the `ddev-windows-*` artifacts and the installers |
| `main-build.yml` | Signed with DDEV's Developer ID, notarized, and stapled, by `make -C desktop app sign notarize` | Built on native amd64 and arm64 runners | Signed by `signtool` with the other Windows binaries, before NSIS packs it |
| Release, by goreleaser | `ddev-desktop_macos-universal.v<version>.zip` on the release, and the `ddev-desktop` cask in `ddev/homebrew-ddev`, from `packaging/ddev-desktop.rb.tmpl` by `.ci-scripts/update-desktop-cask.sh` | In the Linux archives, and the `ddev-desktop` deb and rpm | In the Windows zips and the installers |

The PR jobs run the Go tests and the self-test on each OS. They may fail
without failing the workflow, so the download comment still appears. In
`main-build.yml` the desktop jobs are required, since the release ships
their output.

## Known limits

- **Windows:** nobody has used the tray menu or seen a notification there.
  The app has no icon resource, so a notification has no icon. The installer
  does not check for the WebView2 runtime.
- **Linux:** no AUR or Homebrew package holds the app yet.
- **Updates:** the app has no update check of its own. Homebrew, the Linux
  packages, and the Windows installer update it with `ddev`.
