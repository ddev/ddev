# Experimental DDEV GUI app

This is an experimental GUI app for DDEV, built with
[Wails v3](https://v3.wails.io/). `ddev gui` opens it, and
`ddev gui --browser` opens the same web UI (`pkg/webui`) in a browser. The
app shows that web UI in a native window, and adds what a browser tab cannot:

- A tray menu, also called a menu bar extra on macOS. It lists each project
  with its status, starts, stops, or restarts it, and opens its site. It also
  has Start all, Stop all, and Power off; the last two ask first, as the
  page does. Its About DDEV shows the versions of the app and of `ddev`,
  the `ddev` it runs, and `ddev version`, as the page's own About does.
- A dot on the tray icon: yellow while an action runs or a project starts,
  green while a project runs, and none when all are stopped. macOS draws a
  menu bar icon in one color, so there the dot is beside the icon.
- A question before quitting while an action runs, since quitting stops the
  action and could leave a project half started. Ctrl+C or SIGTERM quits
  without asking.
- A Dock or taskbar icon. Closing the window keeps the app in the tray.
  Launching the app again shows the window, which is the way back on a
  desktop without a tray, such as GNOME without the AppIndicator extension.
  On Windows and Linux, a click on the tray icon shows or hides the window,
  and a right-click opens the menu. On macOS, a click opens the menu.
- A native notification when an action ends while the window is in the
  background. Notifications need the app bundle on macOS and a session bus on
  Linux; without them the app runs without notifications.

## One ddev

The app holds no DDEV code and carries no `ddev` binary. It runs
`ddev gui --desktop` from the `ddev` you already have, shows that web
UI, and drives the tray through the same HTTP API. The window, the
tray, and your terminal therefore read and change projects with one `ddev`,
of one version. Upgrading `ddev` upgrades what the app shows.

The app runs, in this order:

1. `$DDEV_BINARY`, when it is set.
2. On Windows, with no `--wsl-distro` argument, a `ddev.exe` next to
   `ddev-gui.exe`, as a traditional Windows install puts it. Otherwise
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

`DDEV_GUI_MODE=native` or `wsl` overrides step 2. With `wsl`, or with
`--wsl-distro`, the app ignores `DDEV_BINARY` and runs the `ddev` on the
distro's login `PATH`. A `ddev.exe` left by an earlier traditional install
wins over WSL2 when the app starts without the Start Menu shortcut.

With ddev in WSL2, `ddev gui --desktop` listens on `127.0.0.1` inside the
distro, and the window reaches it through WSL's localhost forwarding, which
is on by default. Its links and folders open on Windows, through
`rundll32.exe` and `explorer.exe`. User documentation is in
`docs/content/users/usage/gui.md`.

`ddev gui --desktop` listens on `127.0.0.1` with a token, opens links in
the default browser, and stops when its standard input closes, so it stops
with the app, inside WSL2 too. On macOS and Linux the window reaches it
through the Wails asset server, which passes each request to `ddev gui`
with the token, so the window needs no token or cookie. A state change must
carry the `X-DDEV-Dashboard` header, which the page always sends and a page
from another origin cannot add without a CORS preflight. On Windows, Wails'
WebView2 response writer cannot stream, so the window loads the URL of
`ddev gui` instead, as a browser would.

The app needs a `ddev` with `ddev gui`. With an older one, the window and
a dialog say that `ddev gui --desktop` stopped, with what it printed.

This is a separate Go module, so that Wails stays out of the `ddev` binary
and `vendor/`. Wails needs CGO on macOS and Linux, but not on Windows, so the
Windows app cross-builds anywhere.

## Install

| OS | How | Where |
| --- | --- | --- |
| macOS | `brew install --cask ddev/ddev/ddev-gui`, which also installs the `ddev` formula. Or download `ddev-gui_macos-universal.v<version>.zip` from a release. | `/Applications/DDEV.app`, one universal app for Apple silicon and Intel |
| Windows | The DDEV installer, for every install type. Start it from the Start Menu: DDEV, then DDEV GUI (experimental). | `%LOCALAPPDATA%\Programs\DDEV\ddev-gui.exe` |
| Linux | `sudo apt install ddev-gui` or `sudo dnf install ddev-gui` from DDEV's package repository, which depends on `ddev`. The Linux release archives also hold `ddev-gui`. | `/usr/bin/ddev-gui`, with a launcher entry |

The Linux app needs glibc 2.38 or later, GTK 4, and WebKitGTK 6, as in
Ubuntu 24.04 or Debian 13. The tray icon needs a StatusNotifierItem host.
Ubuntu's GNOME and KDE Plasma have one; GNOME elsewhere needs the
AppIndicator extension.

On Windows the app needs the WebView2 runtime, which Windows 11 includes.

## Build and run

`make gui` builds `ddev` for this OS, then the app:

- On macOS, a universal `DDEV.app` in `.gotmp/bin/darwin_universal/`, with
  `DDEV.app.zip` next to it, zipped by `ditto` to keep its signature. Building
  needs the Xcode command-line tools.
- On Linux, `ddev-gui` next to `ddev` in `.gotmp/bin/linux_<arch>/`.
  Building needs the GTK 4 and WebKitGTK 6 development packages, such as
  `libgtk-4-dev` and `libwebkitgtk-6.0-dev` on Debian and Ubuntu.
- For Windows, from any OS, `make windows_amd64_gui` puts
  `ddev-gui.exe` next to `ddev.exe` in `.gotmp/bin/windows_amd64/`.
  `make windows_amd64_install` builds it into the installer.

`ddev gui` runs the app with `DDEV_BINARY` set to itself, so a local build
of `ddev` opens a local build of the app if `DDEV_GUI_APP` names it:

```bash
DDEV_GUI_APP="$PWD/.gotmp/bin/darwin_universal/DDEV.app" .gotmp/bin/darwin_arm64/ddev gui
```

Started any other way, the app runs the `ddev` it finds as described above,
which for a build of your own is often not the `ddev` you just built. Set
`DDEV_BINARY`:

```bash
# macOS; open passes none of your shell's variables, --env passes one
open --env DDEV_BINARY="$PWD/.gotmp/bin/darwin_arm64/ddev" .gotmp/bin/darwin_universal/DDEV.app
# Or run it in the foreground, with its log in the terminal
DDEV_BINARY="$PWD/.gotmp/bin/darwin_arm64/ddev" .gotmp/bin/darwin_universal/DDEV.app/Contents/MacOS/ddev-gui
# Quit it
osascript -e 'quit app "DDEV"'

# Linux; the app runs the ddev next to it
.gotmp/bin/linux_amd64/ddev-gui &
pkill -x ddev-gui
```

`open --env` has no effect when the app is already running.

From PowerShell on Windows:

```powershell
# Start it; it returns at once, since it is a GUI app
.\ddev-gui.exe
# Use ddev in a particular WSL2 distro
.\ddev-gui.exe --wsl-distro Ubuntu-24.04
# Quit it, or use Quit in the tray menu
Stop-Process -Name ddev-gui
```

A GUI app prints nothing to the Windows console. To see its log, pipe it, as
`./ddev-gui.exe | cat` in Git Bash.

To try a PR's build, `ddev utility download-ddev --pr <number>` downloads its
`ddev` and its GUI app into one directory, where the app finds that
`ddev`. On macOS that is `DDEV.app` next to `ddev`.

`DDEV_GUI_DEBUG=true` logs each request the window makes, and Wails'
own debug messages, such as a tray menu that failed to open.
On Windows, `DDEV_GUI_WEBVIEW2_ARGS` passes arguments to WebView2. With
`--remote-debugging-port=9222`, `http://127.0.0.1:9222/json` lists the
window's page, which Edge's DevTools can then inspect.

The icons use DDEV's brand mark from
`docs/content/developers/logos/SVG/Logo.svg`. After a change to
`icons/appicon.svg`, `icons/tray.svg`, or `pkg/webui/assets/favicon.svg`,
render the PNGs and the ICO again:

```bash
cd gui/icons
rsvg-convert -w 1024 -h 1024 appicon.svg -o appicon.png
rsvg-convert -w 44 -h 44 tray.svg -o tray.png
rsvg-convert -w 64 -h 64 ../../pkg/webui/assets/favicon.svg -o tray-color.png
mkdir -p "$HOME"/tmp/ico
for s in 16 20 24 32 40 48 64 256; do
  rsvg-convert -w $s -h $s ../../pkg/webui/assets/favicon.svg -o "$HOME"/tmp/ico/$s.png
done
# Below 32 pixels the mark's lines fade, so they get more opaque.
for s in 16 20 24; do
  magick "$HOME"/tmp/ico/$s.png -channel A -fx "min(1,u*32/$s)" +channel PNG32:"$HOME"/tmp/ico/$s.png
done
# ImageMagick stores an ICO's images uncompressed; this stores the PNGs.
python3 - "$HOME"/tmp/ico/{16,20,24,32,40,48,64,256}.png <<'PY'
import struct, sys
datas = [open(p, "rb").read() for p in sys.argv[1:]]
offset, entries = 6 + 16 * len(datas), b""
for d in datas:
    w, h = struct.unpack(">II", d[16:24])
    entries += struct.pack("<BBBBHHII", w % 256, h % 256, 0, 0, 1, 32, len(d), offset)
    offset += len(d)
open("appicon.ico", "wb").write(struct.pack("<HHH", 0, 1, len(datas)) + entries + b"".join(datas))
PY
```

`appicon.png` has the macOS tile. Windows uses `appicon.ico` and Linux
packages install `favicon.svg`, both the mark alone, as icons there have no
tile.

## Tests

- `make -C gui test` runs the Go tests with CGO on, which Wails needs
  on macOS and Linux. They cover how the app finds `ddev`, reads the status
  stream, notices jobs that end, and proxies the window's requests. With
  `DDEV_BINARY` set to a `ddev` with `ddev gui`, they also start a real
  `ddev gui --desktop` and drive it.
- `DDEV_GUI_SELFTEST=true` runs `selftest.js` inside the app's own
  window, then exits with its result. The script waits for the status stream,
  submits a missing directory to "Add project" and expects an error, then
  runs `ddev config` in an empty directory, waits for the job to finish, and
  expects that project's detail to answer within 30 seconds. On Windows it
  then posts the shell's click messages to its own tray window: a click must
  show the hidden window, a right-click must open the menu, and the menu's
  Open DDEV must show the window again. The app keeps that project in a temporary global configuration, which it
  deletes at the end. It needs no Docker. On Linux, run it under `xvfb-run`
  when there is no display. On Windows the self-test's window loads the
  dashboard from a port of the app's own, since it must add its script to
  the page. Do not run it against ddev in WSL2: its temporary directories
  are Windows paths that ddev inside WSL2 never sees, so the test project
  would land in your real global configuration.
- Without Docker, a project's detail fails before it runs host commands
  such as `git`. `TestStopWithStdin` in `pkg/webui` therefore checks without
  Docker that `ddev gui --desktop` keeps its stdin pipe from such
  commands, which hang on Windows when they inherit it.
- `make webui-e2e` runs the dashboard's Playwright tests in Chromium and in
  WebKit, the engine of the app's window on macOS and Linux.

```bash
DDEV_GUI_SELFTEST=true DDEV_BINARY="$PWD/.gotmp/bin/darwin_arm64/ddev" .gotmp/bin/darwin_universal/DDEV.app/Contents/MacOS/ddev-gui
# Expect "SELFTEST PASS" and exit status 0
```

Notifications, the quit question, and the tray on macOS and Linux have no
automated test.

## CI, signing, and releases

| Workflow | macOS | Linux | Windows |
| --- | --- | --- | --- |
| `pr-build.yml` | `DDEV.app.zip`, unsigned, added to the `ddev-macos-arm64` and `ddev-macos-amd64` artifacts | `ddev-gui` added to the `ddev-linux-*` artifacts | `ddev-gui.exe` in the `ddev-windows-*` artifacts and the installers |
| `main-build.yml` | Signed with DDEV's Developer ID, notarized, and stapled, by `make -C gui app sign notarize` | Built on native amd64 and arm64 runners | Signed by `signtool` with the other Windows binaries, before NSIS packs it |
| Release, by goreleaser | `ddev-gui_macos-universal.v<version>.zip` on the release, and the `ddev-gui` cask in `ddev/homebrew-ddev`, from `packaging/ddev-gui.rb.tmpl` by `.ci-scripts/update-gui-cask.sh` | In the Linux archives, and the `ddev-gui` deb and rpm | In the Windows zips and the installers |

The PR jobs run the Go tests and the self-test on each OS. They may fail
without failing the workflow, so the download comment still appears. In
`main-build.yml` the GUI jobs are required, since the release ships
their output.

## Known limits

- **Windows:** nobody has seen a notification there.
  The app has no icon resource, so a notification has no icon. The installer
  does not check for the WebView2 runtime.
- **Linux:** no AUR or Homebrew package holds the app yet. On Ubuntu 26.04
  arm64, a closed window once did not come back, from the tray or by
  relaunching. On amd64 it comes back with GNOME and KDE Plasma, on Wayland
  and through XWayland.
- **Updates:** the app has no update check of its own. Homebrew, the Linux
  packages, and the Windows installer update it with `ddev`.
