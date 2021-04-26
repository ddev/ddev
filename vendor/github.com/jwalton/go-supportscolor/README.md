# supports-color

Go library to detect whether a terminal supports color, and enables ANSI color support in recent Windows 10 builds.

This is a port of the Node.js package [supports-color](https://github.com/chalk/supports-color) v8.1.1 by [Sindre Sorhus](https://github.com/sindresorhus) and [Josh Junon](https://github.com/qix-).

## Install

```sh
$ go get github.com/jwalton/go-supportscolor
```

## Usage

```go
import (
    "fmt"
    "github.com/jwalton/go-supportscolor"
)

if supportscolor.Stdout().SupportsColor {
    fmt.Println("Terminal stdout supports color")
}

if supportscolor.Stdout().Has256 {
    fmt.Println("Terminal stdout supports 256 colors")
}

if supportscolor.Stderr().Has16m {
    fmt.Println("Terminal stderr supports 16 million colors (true color)")
}
```

## Windows 10 Support

`supportscolor` is cross-platform, and will work on Linux and MacOS systems, but will also work on Windows 10.

Many ANSI color libraries for Go do a poor job of handling colors in Windows.  This is because historically, Windows has not supported ANSI color codes, so hacks like [ansicon](https://github.com/adoxa/ansicon) or [go-colorable](https://github.com/mattn/go-colorable) were required.  However, Windows 10 has supported ANSI escape codes since 2017 (build 10586 for 256 color support, and build 14931 for 16.7 million true color support).  In [Windows Terminal](https://github.com/Microsoft/Terminal) this is enabled by default, but in `CMD.EXE` or PowerShell, ANSI support must be enabled via [`ENABLE_VIRTUAL_TERMINAL_PROCESSING`](https://docs.microsoft.com/en-us/windows/console/console-virtual-terminal-sequences).

This library takes care of all of this for you, though - if you call `supportscolor.Stdout()` on a modern build of Windows 10, it will set the `ENABLE_VIRTUAL_TERMINAL_PROCESSING` console mode automatically if required, and return the correct color level, and then you can just write ANSI escape codes to stdout and not worry about it.  If someone uses your app on an old version of Windows, this will return `SupportsColor == false`, and you can write black and white to stdout.

## API

Returns a `supportscolor.Support` with a `Stdout()` and `Stderr()` function for testing either stream.  (There's one for stdout and one for stderr, because if you run `mycmd > foo.txt` then stdout would be redirected to a file, and since it would not be a TTY would not have color support, while stderr would still be going to the console and would have color support.)

The `Stdout()`/`Stderr()` objects specify a level of support for color through a `.Level` property and a corresponding flag:

- `.Level = None` and `.SupportsColor = false`: No color support
- `.Level = Basic` and `.SupportsColor = true`: Basic color support (16 colors)
- `.Level = Ansi256` and `.Has256 = true`: 256 color support
- `.Level = Ansi16m` and `.Has16m = true`: True color support (16 million colors)

### `supportscolor.SupportsColor(fd, ...options)`

Additionally, `supportscolor` exposes the `.SupportsColor()` function that takes an arbitrary file descriptor (e.g. `os.Stdout.Fd()`) and options and will (re-)evaluate color support for an arbitrary stream.

For example, `supportscolor.Stdout()` is the equivalent of `supportscolor.SupportsColor(os.Stdout.Fd())`.

Available options are:

- `supportscolor.IsTTYOption(isTTY bool)` - Force whether the given file should be considered a TTY or not. If this not specified, TTY status will be detected automatically via `term.IsTerminal()`.
- `supportscolor.SniffFlagsOption(sniffFlags bool)` - By default it is `true`, which instructs `SupportsColor()` to sniff `os.Args` for the multitude of `--color` flags (see _Info_ below). If `false`, then `os.Args` is not considered when determining color support.

## Info

By default, supportscolor checks `os.Args` for the `--color` and `--no-color` CLI flags.

For situations where using `--color` is not possible, use the environment variable `FORCE_COLOR=1` (level 1 - 16 colors), `FORCE_COLOR=2` (level 2 - 256 colors), or `FORCE_COLOR=3` (level 3 - true color) to forcefully enable color, or `FORCE_COLOR=0` to forcefully disable. The use of `FORCE_COLOR` overrides all other color support checks.

Explicit 256/True color mode can be enabled using the `--color=256` and `--color=16m` flags, respectively.
