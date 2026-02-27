# Clone

DDEV clone creates independent copies of your project environment using git worktrees and Docker volume duplication. Each clone has its own code worktree, database, and containers while sharing git history with the source project.

## Commands

### `ddev clone create`

Create a clone of the current DDEV project:

```shell
# Create a clone with a new branch
ddev clone create feature-x

# Clone using an existing branch
ddev clone create feature-x --branch existing-branch

# Create a clone without starting it
ddev clone create feature-x --no-start
```

This command:

1. Creates a git worktree at `../<project>-clone-<name>`
2. Copies all Docker volumes (database, config, snapshots)
3. Configures a new DDEV project
4. Starts the cloned project

The database container in the source project is temporarily stopped during volume copy to ensure data consistency.

### `ddev clone list`

List all clones of the current project:

```shell
ddev clone list
```

Displays a table of all clones with their name, path, branch, and status. The current clone (if running from a clone directory) is marked with `*`.

This command works from the source project or any of its clones.

JSON output is supported:

```shell
ddev clone list -j
```

### `ddev clone remove`

Remove a clone and clean up all its resources:

```shell
# Remove a clone
ddev clone remove feature-x

# Force removal (skip dirty worktree confirmation)
ddev clone remove feature-x --force
```

This stops containers, removes Docker volumes and networks, removes the git worktree, and unregisters the project. If the worktree has uncommitted changes, you'll be asked for confirmation unless `--force` is used.

### `ddev clone prune`

Clean up stale clone references:

```shell
# Show what would be cleaned up
ddev clone prune --dry-run

# Clean up stale clones
ddev clone prune
```

Detects clones whose worktree directories have been manually deleted and cleans up their Docker resources and project registrations.

## How It Works

Each clone consists of:

- **Git worktree**: A separate working directory sharing the same git repository. Changes in one worktree do not affect others.
- **Docker volumes**: Independent copies of all project volumes (database, config, snapshots). The clone has its own database with the same data as the source at the time of cloning.
- **DDEV project**: A fully independent DDEV project with its own containers, network, and hostname.

### Naming Convention

Clone projects follow the naming pattern `<source>-clone-<name>`:

- Source project: `mysite`
- Clone: `mysite-clone-feature-x`
- Clone URL: `https://mysite-clone-feature-x.ddev.site`

### Volume Cloning

Volumes are cloned using an ephemeral Docker container with tar:

- Database volume (MariaDB, MySQL, or PostgreSQL)
- Config volume
- Snapshot volume
- Mutagen volume (if enabled)

## Hooks

Clone operations support DDEV hooks in `.ddev/config.yaml`:

```yaml
hooks:
  pre-clone-create:
    - exec-host: echo "About to create a clone"
  post-clone-create:
    - exec: drush cr
  pre-clone-remove:
    - exec-host: echo "About to remove a clone"
  post-clone-remove:
    - exec-host: echo "Clone removed"
```

| Hook | Trigger |
|------|---------|
| `pre-clone-create` | Before clone creation (source project context) |
| `post-clone-create` | After clone is created and started (clone context) |
| `pre-clone-remove` | Before clone removal |
| `post-clone-remove` | After clone removal |

## CI Best Practices

For CI environments, consider the stopped source pattern:

```shell
# Stop the source project before cloning (faster, no DB restart needed)
ddev stop
ddev clone create ci-test --no-start
cd ../mysite-clone-ci-test
ddev start
# Run tests...
ddev clone remove ci-test --force
```

## Mutagen Considerations

If the source project uses Mutagen for file sync:

- The Mutagen volume is cloned along with other volumes
- When the clone starts, DDEV automatically creates a new Mutagen sync session for the clone's worktree
- Each clone has an independent Mutagen sync

## Configuration

The volume cloning strategy can be configured globally:

```yaml
# ~/.ddev/global_config.yaml
volume_clone_strategy: tar-copy  # default
```

Currently, `tar-copy` is the only available strategy. Future versions may add additional strategies for filesystems that support efficient cloning (btrfs, zfs).
