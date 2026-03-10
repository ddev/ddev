# Plan: Replace Command-Parsing Compose Layer with Direct API Calls

## Problem

The current `ComposeCmd` / `ComposeWithStreams` in `pkg/dockerutil/docker_compose.go` simulate the old `docker-compose` CLI by:

1. Accepting `Action []string` (e.g., `[]string{"--progress=quiet", "build", "--no-cache"}`)
2. Parsing that string array to extract global flags (`-p`, `--progress`, `--env-file`) and the subcommand
3. Parsing subcommand-specific flags (e.g., `--build` for `up`, `-T` for `exec`)
4. Dispatching to `svc.Up()`, `svc.Down()`, `svc.Build()`, etc.

This approach re-invents CLI argument parsing that compose already solved, is fragile (any new flag needs manual parser updates), and obscures what the callers actually want.

**The compose repo itself** (`/home/stas/code/pet/compose`) does this properly: each command (up, down, exec, build, pull, stop, config) is a separate function that directly calls `compose.NewComposeService()` and then calls the appropriate API method with typed options. There is no string-based command dispatch.

## Goal

Replace `ComposeCmd` / `ComposeWithStreams` with direct, typed wrapper functions that callers use instead of building string arrays. Each wrapper directly calls the compose SDK API method it needs.

## Current Call Sites Inventory

### Subcommands Used by Callers

| Subcommand | Callers | Notes |
|------------|---------|-------|
| `config` | `compose_yaml.go:75`, `router.go:281`, `ssh_auth.go:157` | Always uses `ComposeFiles`, some use `--env-file`, `--services`, `Profiles` |
| `up` | `ddevapp.go:1861`, `ddevapp.go:2161`, `router.go:162`, `ssh_auth.go:71` | Some with `--build`, `-d` (always detached), some with `Profiles`, some with `-p` |
| `down` | `utils.go:111`, `ssh_auth.go:57` | Always `ComposeFiles`, some with `Profiles` |
| `stop` | `ddevapp.go:2965` | `ComposeFiles` + `Profiles` |
| `build` | `ddevapp.go:1459-1474` | `ComposeFiles`, `ProjectName`, `Progress`, `Timeout`, `--progress=`, `--no-cache` |
| `exec` | `ddevapp.go:2552-2560`, `ddevapp.go:2625` | `ComposeFiles` + exec args (`-T`, `-u`, `-w`, service, command). Both `ComposeWithStreams` and `ComposeCmd` |
| `pull` | `docker_compose.go:643-653` (PullImages) | Uses `ComposeYaml` (in-memory project), `Progress` |

### How `ComposeYaml` (In-Memory Project) is Used

Only `PullImages` passes a pre-built `*types.Project` directly. All other callers use `ComposeFiles` (file paths). The `ComposeYaml` path bypasses file loading and just sets labels.

### How `Env` Field is Used

The old `Env []string` field was used for `COMPOSE_DISABLE_ENV_FILE=1` etc. In the current branch, `Env` is no longer used by any caller (the compose library handles env internally). This field can be removed from `ComposeCmdOpts`.

## Proposed New API

Replace the two generic functions with typed wrappers. Each wrapper:
- Accepts a typed options struct (no string parsing)
- Calls `getComposeService()` and the appropriate `svc.Method()` directly
- Handles timeout and context internally

### New Functions

```go
// ComposeUp starts services (always detached in DDEV's case).
func ComposeUp(opts ComposeUpOpts) error

type ComposeUpOpts struct {
    ComposeFiles []string
    Project      *types.Project // Alternative: pass pre-loaded project directly
    ProjectName  string
    Profiles     []string
    Build        bool           // --build flag
    Progress     bool           // Show progress UI
    Timeout      time.Duration
}

// ComposeDown stops and removes containers/networks.
func ComposeDown(opts ComposeDownOpts) error

type ComposeDownOpts struct {
    ComposeFiles    []string
    Project         *types.Project
    ProjectName     string
    Profiles        []string
    RemoveOrphans   bool
    Timeout         time.Duration
}

// ComposeStop stops services without removing them.
func ComposeStop(opts ComposeStopOpts) error

type ComposeStopOpts struct {
    ComposeFiles []string
    Project      *types.Project
    ProjectName  string
    Profiles     []string
    Timeout      time.Duration
}

// ComposeBuild builds service images.
func ComposeBuild(opts ComposeBuildOpts) (string, error)

type ComposeBuildOpts struct {
    ComposeFiles []string
    Project      *types.Project
    ProjectName  string
    NoCache      bool
    Progress     string   // "auto", "plain", "quiet", "tty"
    ShowProgress bool     // Attach EventProcessor
    Timeout      time.Duration
    // For streaming mode:
    Stdout       io.Writer // If set, stream to these instead of capturing
    Stderr       io.Writer
}

// ComposeExec runs a command in a running service container.
func ComposeExec(opts ComposeExecOpts) (string, string, error)

type ComposeExecOpts struct {
    ComposeFiles []string
    Project      *types.Project
    ProjectName  string
    Service      string
    Command      []string
    Tty          bool
    Interactive  bool
    Detach       bool
    User         string
    WorkDir      string
    Env          []string
    // For streaming mode:
    Stdin        io.Reader
    Stdout       io.Writer
    Stderr       io.Writer
}

// ComposeExecStreams runs exec with direct I/O streaming (for interactive commands).
func ComposeExecStreams(opts ComposeExecOpts, stdin io.Reader, stdout, stderr io.Writer) error

// ComposeConfig loads and merges compose files, returns the rendered YAML.
func ComposeConfig(opts ComposeConfigOpts) (string, error)

type ComposeConfigOpts struct {
    ComposeFiles []string
    ProjectName  string
    Profiles     []string
    EnvFiles     []string
    Services     bool     // Return only service names
}

// ComposePull pulls images for services.
func ComposePull(opts ComposePullOpts) error

type ComposePullOpts struct {
    ComposeFiles []string
    Project      *types.Project
    ProjectName  string
    Progress     bool
    // For streaming mode:
    Stdout       io.Writer
    Stderr       io.Writer
}
```

### Project Loading

Move project loading into each wrapper. The compose library's `LoadProject` API (via `compose.NewComposeService().LoadProject()`) handles custom labels, env file loading, profiles, etc. properly. Use it instead of `loadProjectFromCmd` + `setCustomLabels`.

Alternatively, keep using `loader.LoadWithContext` for simple cases but use `api.ProjectLoadOptions` to match how compose does it internally. The key is that callers should not need to know about project loading at all - they pass files and options, the wrapper handles loading.

### Progress Display

Keep `getComposeService()` for creating the compose service with appropriate progress display. The TTY detection logic there is correct. Progress mode should be a simple enum/string passed in opts, not embedded in `Action` strings.

## Implementation Steps

### Step 1: Create New Typed Functions (Additive)

Add the new typed functions alongside the existing `ComposeCmd`/`ComposeWithStreams`. Each function:
1. Creates a context (with timeout if specified)
2. Loads the project (from files or from the passed `*types.Project`)
3. Creates the compose service via `getComposeService()`
4. Calls the appropriate API method with typed options
5. Returns results

Keep internal helpers: `getComposeService()`, `loadProjectFromCmd()`, `setCustomLabels()`.

### Step 2: Migrate Callers One by One

Migrate each call site from `ComposeCmd`/`ComposeWithStreams` to the new typed function:

1. **`config` callers** (3 sites) -> `ComposeConfig()`
   - `compose_yaml.go:75` - uses env-files, profiles
   - `router.go:281` - simple, files only
   - `ssh_auth.go:157` - simple, files only

2. **`up` callers** (4 sites) -> `ComposeUp()`
   - `ddevapp.go:1861` - simple up -d
   - `ddevapp.go:2161` - up -d with profiles
   - `router.go:162` - up --build -d with -p flag
   - `ssh_auth.go:71` - up --build -d with -p flag

3. **`down` callers** (2 sites) -> `ComposeDown()`
   - `utils.go:111` - with profiles
   - `ssh_auth.go:57` - simple

4. **`stop` caller** (1 site) -> `ComposeStop()`
   - `ddevapp.go:2965` - with profiles

5. **`build` caller** (1 site) -> `ComposeBuild()`
   - `ddevapp.go:1459-1474` - with progress, timeout, no-cache, retry logic

6. **`exec` callers** (2 sites) -> `ComposeExec()` / `ComposeExecStreams()`
   - `ddevapp.go:2552-2560` - both streaming and capturing modes
   - `ddevapp.go:2625` - always streaming

7. **`pull` caller** (1 site in PullImages) -> `ComposePull()`
   - `docker_compose.go:643-653` - uses in-memory project

### Step 3: Remove Old Interface

Once all callers are migrated:
1. Delete `ComposeCmd()` and `ComposeWithStreams()`
2. Delete `ComposeCmdOpts` struct
3. Delete `parseComposeAction()`, `parseExecSubArgs()`, `containsFlag()`
4. Delete `parsedAction` and `execSubArgs` types

### Step 4: Clean Up Project Loading

Consider switching from `loader.LoadWithContext()` to `svc.LoadProject()` (the compose service's built-in loader) which handles:
- Custom labels automatically (no need for `setCustomLabels()`)
- Env file resolution
- Profile handling
- Service selection

This would let us delete `setCustomLabels()` and simplify `loadProjectFromCmd()`.

### Step 5: Update Tests

Update `docker_compose_test.go` to test the new typed functions instead of the generic `ComposeCmd`/`ComposeWithStreams`.

## What NOT to Change

- `getComposeService()` - this is already well-structured
- `dockerManager` singleton and initialization - works fine
- `CreateComposeProject()` - utility for creating in-memory projects
- `PullImages()` / `Pull()` - just update to use `ComposePull()` internally
- `DownloadDockerBuildx*` functions - separate concern, already done
- `sanitizeServiceName()` - utility function, keep as-is

## Key Differences from Compose CLI

The compose CLI (`cmd/compose/compose.go`) uses cobra commands and `ProjectOptions.WithServices()` which loads the project inside cobra's `RunE`. DDEV doesn't use cobra for compose operations, so our wrappers serve the same purpose - they're the bridge between "I have file paths and options" and "call the compose SDK".

The compose CLI creates `NewComposeService(dockerCli)` fresh for each command. DDEV can do the same or reuse via `getComposeService()`. The important thing is that `backendOptions.Options` (event processor, dry-run, etc.) are passed at service creation time, which DDEV already handles correctly.

## Risk Assessment

- **Low risk**: All changes are in `pkg/dockerutil/docker_compose.go` and its callers
- **Medium risk**: Project loading differences between `loader.LoadWithContext` and `svc.LoadProject()` - the latter does more (service selection, env resolution). Need to verify DDEV's compose files work correctly with both
- **Testing**: Each step can be tested independently by running the affected integration tests