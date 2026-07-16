<a name="opt-in-usage-information"></a>

# Opt-In Usage Information

When you start DDEV for the first time or install a new release, you’ll be asked whether to send usage and error information to DDEV’s developers.

Regardless of your choice, you can change this at any time using one of these approaches:

- `ddev config global --instrumentation-opt-in=false` (or `=true`), which sets `instrumentation_opt_in` in `$HOME/.ddev/global_config.yaml` (see [global configuration directory](../usage/architecture.md#global-files))
- `export DDEV_NO_INSTRUMENTATION=true` to disable it regardless of the config file setting

Usage information is also never collected in automated workflows where `CI=true` is set in the environment.

!!!tip "See the aggregated stats"
    The anonymized, aggregated usage statistics gathered from opted-in users are published at [DDEV Live Usage Statistics](https://ddev.com/usage-stats/).

If you choose to share diagnostics, it helps us tremendously in our effort to improve the tool. DDEV tracks three kinds of events: `Identify` (about your machine), `Command` (which command you ran), and `Project` (your project's configuration). Here’s an example of what we might see:

```go
// Sent once per machine, to identify the environment DDEV runs in.
client.Track(&analytics.Track{
  DeviceId:   "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
  AppVersion: "v1.25.3",
  Platform:   "arm64",
  OSName:     "darwin",
  Language:   "en_US.UTF-8",
  ProductID:  "ddev cli",
  Event:      "Identify",
  Properties: map[string]interface{}{
    "DDEV Environment": "darwin",
    "Docker Platform":  "docker-desktop",
    "Docker Version":   "27.3.1",
    "Timezone":         "PST",
    // "User" is only sent if you've set `instrumentation_user` yourself
    "User": "jane@example.com",
    // "WSL Distro" is only sent when DDEV runs inside WSL2
    "WSL Distro": "Ubuntu",
  },
})

// Sent on every command run.
client.Track(&analytics.Track{
  DeviceId:   "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
  AppVersion: "v1.25.3",
  Platform:   "arm64",
  OSName:     "darwin",
  Language:   "en_US.UTF-8",
  ProductID:  "ddev cli",
  Event:      "Command",
  Properties: map[string]interface{}{
    // Positional arguments are sent as-is for built-in commands.
    // For your own custom commands the command name is replaced
    // with "custom-command" and the arguments are dropped entirely.
    "Arguments":    []string{},
    "Called As":    "start",
    "Command Name": "start",
    "Command Path": "ddev start",
  },
})

// Sent when a project starts, describing its configuration.
client.Track(&analytics.Track{
  DeviceId:   "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
  AppVersion: "v1.25.3",
  Platform:   "arm64",
  OSName:     "darwin",
  Language:   "en_US.UTF-8",
  ProductID:  "ddev cli",
  Event:      "Project",
  Properties: map[string]interface{}{
    "ID":                          "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
    "Performance Mode":            "mutagen",
    "PHP Version":                 "8.5",
    "Project Type":                "php",
    "Webserver Type":              "nginx-fpm",
    "XHProf Mode":                 "xhgui",
    "Add-ons":                     []string{"redis"},
    "Bind All Interfaces":         false,
    "CI":                          false,
    "Containers":                  []string{"db", "redis", "web"},
    "Containers Omitted":          []string{},
    "Corepack Enable":             true,
    "Database Type":               "mariadb",
    "Database Version":            "11.8",
    "DB Image Extra Packages":     []string{"netcat"},
    "DDEV Version Constraint":     ">= 1.25.3",
    "Disable Settings Management": false,
    "Fail On Hook Fail":           false,
    "No Project Mount":            false,
    "Nodejs Version":              "24",
    "Router":                      "traefik",
    "Router Disabled":             false,
    "WebExtraDaemonsDetails":      []string{"{\"Name\":\"vite\",\"Command\":\"npm run dev\",\"Directory\":\"/var/www/html\"}"},
    "WebExtraDaemonsNames":        []string{"vite"},
    "WebExtraExposedPortsDetails": []string{"{\"Name\":\"vite\",\"WebContainerPort\":5173,\"HTTPPort\":5172,\"HTTPSPort\":5173}"},
    "WebExtraExposedPortsNames":   []string{"vite"},
    "Webimage Extra Packages":     []string{"php${DDEV_PHP_VERSION}-tidy"},
  },
})
```

If you have any issues or concerns with it, we’d like to know.
