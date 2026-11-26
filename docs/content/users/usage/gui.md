# DDEV GUI (Experimental)

The DDEV GUI app manages DDEV projects in a window of its own, with a menu in the system tray or macOS menu bar. From the window or the tray you can start, stop, and restart projects, open their sites and Mailpit, and follow what each action prints.

!!!warning "Experimental"
    The GUI app is experimental. Its features and behavior may change, and it has no update check of its own; it updates when DDEV does.

[`ddev gui`](../usage/commands.md#gui) opens the app, and `ddev gui --browser` opens the same interface in your browser instead.

The app contains no DDEV code. It runs `ddev gui --desktop` from the `ddev` you already have, so the window, the tray, and your terminal always show the same projects and use the same version of DDEV.

## Installing

=== "macOS"

    ```bash
    brew install --cask ddev/ddev/ddev-gui
    ```

    This also installs the `ddev` formula. The app is `/Applications/DDEV.app`. You can instead download `ddev-gui_macos-universal.v<version>.zip` from a [DDEV release](https://github.com/ddev/ddev/releases).

=== "Windows"

    The [DDEV Windows installer](../install/ddev-installation.md#windows) installs the app for every installation type, traditional Windows and WSL2. It needs the WebView2 runtime, which Windows 11 includes.

=== "Linux"

    Install `ddev-gui` from DDEV's package repository, which also installs `ddev`:

    ```bash
    # Debian and Ubuntu
    sudo apt-get install ddev-gui
    # Fedora, Red Hat, and similar
    sudo dnf install ddev-gui
    ```

    The app needs GTK 4 and WebKitGTK 6, as in Ubuntu 24.04 or Debian 13 and later. The tray icon needs a StatusNotifierItem host: Ubuntu's GNOME and KDE Plasma have one, and GNOME elsewhere needs the AppIndicator extension.

## Starting and Quitting

Run `ddev gui` in a terminal, or start the app as you start any other: from the Applications folder, the Start Menu (**DDEV** → **DDEV GUI (experimental)**), or your desktop's launcher.

`ddev gui` selects the project in the current directory, and `ddev gui my-project` selects `my-project`. If the app is already running, its window comes forward with that project selected. In WSL2, `ddev gui` opens the Windows app with the projects of that distro. If `ddev gui` cannot find the app, or runs in an SSH session or a Linux session without a display, it opens the interface in your browser and says why.

`ddev gui` looks for the app next to `ddev`, then in `/Applications` and `~/Applications` on macOS, or on `PATH` on Linux and Windows. To use a copy elsewhere, set `DDEV_GUI_APP` to its path:

```bash
DDEV_GUI_APP=~/Downloads/DDEV.app ddev gui
```

Closing the window keeps the app running in the tray. To show the window again, choose **Open DDEV** in the tray menu, or on Windows click the tray icon. Launching the app again, or running `ddev gui`, also shows its window.

To quit, choose **Quit DDEV app** in the tray menu. If an action such as a project start is still running, the app asks first, since quitting stops that action.

When an action ends while the window is in the background, the app shows a notification. On macOS, builds from pull requests and `make gui` are not signed with DDEV's certificate, and macOS shows no notifications for them.

=== "macOS"

    The menu is in the menu bar, and the app has a Dock icon. Start it from the Applications folder or Spotlight, or from a terminal:

    ```bash
    open -a DDEV
    # Quit it
    osascript -e 'quit app "DDEV"'
    ```

    **Open folder** opens the project in Finder. The app finds `ddev` on your login shell's `PATH`, so a Homebrew `ddev` works even though apps started from the Finder get a minimal `PATH`.

=== "Windows"

    The tray icon is in the notification area of the taskbar, possibly under the **^** overflow. Click it to show the window, or right-click it for the menu. From PowerShell:

    ```powershell
    & "$env:LOCALAPPDATA\Programs\DDEV\ddev-gui.exe"
    # Quit it
    Stop-Process -Name ddev-gui
    ```

    **Open folder** opens the project in File Explorer. With DDEV in WSL2, see [Using DDEV in WSL2](#using-ddev-in-wsl2).

=== "Linux"

    Start the app from your desktop's launcher, or from a terminal:

    ```bash
    ddev-gui &
    # Quit it
    pkill -x ddev-gui
    ```

    **Open folder** opens the project with `xdg-open`, which needs `xdg-utils`. The tray icon needs a StatusNotifierItem host, and notifications need a desktop session bus.

    If a closed window does not come back, as has happened on Ubuntu on arm64, quit the app from the tray and start it again.

## Using the Window

The window opens on a start page with DDEV's tip of the day, the latest posts of the [DDEV blog](https://ddev.com/blog/), and links to these docs, unless `ddev gui` selected a project. The app reads the blog's feed from ddev.com, and leaves the posts out when it can't reach it. **Start page** at the top of the project list returns to it. Important notices from DDEV show under the top bar until you dismiss them.

Above a project's tabs, **Open site**, **Mailpit**, and **Open folder** open what they name. The editor button opens the project in VS Code, and its arrow switches it to PhpStorm, which the button then remembers. The **⋯** button holds the rarer actions: snapshots, updating add-ons, and **Delete**. Right-click a project in the list for a menu with the same actions. A **?** next to a heading links to the docs about it, and the project type links to its [quickstart](../quickstart.md).

To start or stop several projects at once, Ctrl+click or Shift+click them in the list, or tick the box that shows when you point at one. The page then offers **Start**, **Stop**, and **Restart** for all of them. **Start all**, **Stop all**, and **Power off** at the top act on every project.

The arrow keys move between the start page, the shared services, and the projects in the list, and `/` goes to the filter.

**Shared services**, or a click on the status at the top, shows the router and the SSH agent that all projects share: their health, the router's configuration problems, a link to the Traefik dashboard, and their logs.

The **Activity** panel at the bottom shows what each action prints, as it prints it. Click its bar to open or close it. The × next to a finished action removes it from the list, and **Clear history** removes every finished action. A running action stays until it ends.

## Using DDEV in WSL2

When DDEV runs in WSL2, the GUI app runs on Windows and uses `ddev` inside your WSL2 distro:

* Start it from the Start Menu: **DDEV** → **DDEV GUI (experimental)**. The installer's shortcut names the distro you chose during installation, which need not be your default distro. You can also run `ddev gui` inside WSL2. It opens the Windows app with that distro's projects, and the app runs the `ddev` on the distro's login `PATH`. It finds the app on the Windows `PATH` or in `%LOCALAPPDATA%\Programs\DDEV`, and otherwise opens your Windows browser.
* Your projects stay in WSL2, and the window shows their Linux paths, such as `/home/you/my-site`. To add a project, enter its Linux path.
* **Open site** and **Mailpit** open your Windows browser. **Open folder** opens File Explorer at the project's `\\wsl.localhost\` path, and **Open in VS Code** opens the project in a VS Code window connected to WSL.
* The window reaches `ddev gui --desktop` in WSL2 through localhost forwarding, which is on by default.
* Quitting the app also stops `ddev gui --desktop` inside WSL2.

If you start `ddev-gui.exe` without the Start Menu shortcut, it chooses which `ddev` to use:

1. A `ddev.exe` in the same folder as `ddev-gui.exe`, as a traditional Windows installation puts it.
2. Otherwise, `ddev` in your default WSL2 distro.

A `ddev.exe` left in `%LOCALAPPDATA%\Programs\DDEV` from an earlier traditional installation therefore takes priority over WSL2. Use the Start Menu shortcut, delete that `ddev.exe`, or name the distro yourself:

```powershell
& "$env:LOCALAPPDATA\Programs\DDEV\ddev-gui.exe" --wsl-distro Ubuntu
```

Run `wsl --list` to see your distros' names.

## Which `ddev` the App Uses

**About DDEV** in the tray menu shows the DDEV version and the `ddev` the app runs. When `ddev gui` starts the app outside WSL2, the app runs that same `ddev`. Otherwise, outside WSL2, the app looks for `ddev` in this order:

1. The `DDEV_BINARY` environment variable, when it is set.
2. On Windows, the `ddev.exe` next to the app, or `ddev` in WSL2, as described in [Using DDEV in WSL2](#using-ddev-in-wsl2).
3. The `ddev` in the same folder as the app, or next to `DDEV.app` on macOS.
4. The first `ddev` on your login shell's `PATH`.
5. The places DDEV's installers use: `/opt/homebrew/bin` and `/usr/local/bin` on macOS, and `/usr/bin`, `/usr/local/bin`, and Homebrew's directory on Linux.

On Windows, `DDEV_GUI_MODE=native` or `DDEV_GUI_MODE=wsl` overrides step 2.

## Troubleshooting

If the app cannot start `ddev gui --desktop`, it shows a "DDEV cannot start" dialog with what `ddev` printed:

* `ddev gui` is unknown: the `ddev` the app found is older than the app. [Upgrade DDEV](../install/ddev-upgrade.md).
* `command not found`, or WSL cannot run `bash`: the WSL2 distro the app used has no DDEV installed, or is not a Linux distro you use, such as `docker-desktop`. Use the Start Menu shortcut, or pass `--wsl-distro` with the right distro.
