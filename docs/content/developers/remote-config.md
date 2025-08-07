---
search:
  boost: .5
---
# Remote Config

DDEV supports downloading a [`remote config`](https://github.com/ddev/remote-config/blob/main/remote-config.jsonc)
from the [`ddev/remote-config`](https://github.com/ddev/remote-config)
GitHub repository with messages that will be shown to the user as a "Tip of the Day". This feature
may be enhanced later with more information and filtering.

## Overview

DDEV's remote configuration system provides a way to deliver dynamic content to users, including:

- **Ticker Messages**: Rotating "tip of the day" messages shown during various DDEV operations
- **Notifications**: Important announcements and warnings displayed to users
- **Sponsorship Data**: Information about DDEV's financial sponsors (separate repository)

The system is designed to be:

- **Non-intrusive**: Users can disable or configure intervals
- **Version-aware**: Messages can target specific DDEV versions
- **Condition-based**: Messages can be shown based on user environment (Docker provider, OS, etc.)
- **Cached locally**: Configuration is cached to minimize network requests

## Data Types

### Remote Configuration

The main configuration includes:

- Update intervals for automatic refreshing
- Notification messages (info and warning types)
- Ticker messages with optional titles and conditions
- Version constraints for message targeting

### Sponsorship Information  

Separate from the main config, DDEV downloads sponsorship data from a JSON endpoint including:

- Monthly and annual sponsor information
- Sponsor counts and income totals
- GitHub sponsorship details
- Update timestamps

## Storage Format

DDEV stores downloaded data locally using Go's `gob` binary encoding format in the user's global DDEV directory:

- `~/.ddev/.remote-config`: Main remote configuration cache
- `~/.ddev/.sponsorship-data`: Sponsorship information cache  
- `~/.ddev/.amplitude.cache`: Analytics event cache (if enabled)

## Debugging Tools

DDEV provides several debugging commands for working with remote configuration:

### View Cached Data

```bash
# Decode and view cached remote config
ddev debug gob-decode ~/.ddev/.remote-config

# Decode sponsorship data
ddev debug gob-decode ~/.ddev/.sponsorship-data

# Decode analytics cache  
ddev debug gob-decode ~/.ddev/.amplitude.cache
```

### Download Fresh Data

```bash
# Download latest remote config (updates cache by default)
ddev debug remote-data --type=remote-config

# Download sponsorship data without updating cache
ddev debug remote-data --type=sponsorship-data --update-storage=false
```

### View Available Conditions

```bash
# List all available message conditions
ddev debug message-conditions
```

## Messages

### Notifications

The defined messages are shown to the user every `interval` as long as not
disabled (interval=-1). Supported message types are `infos` and `warnings` where
`infos` are printed in a yellow box and `warnings` in a red box.

Messages will be shown as configured in the `remote-config` repository and the
user cannot influence them.

### Infos

`infos` and `warnings` (yellow and red) can be specified like this:

```json
{
  "messages": {
    "notifications": {
      "interval": 20,
      "infos": [
        {
          "message": "This is a message to users of DDEV before v1.22.7",
          "versions": "<=v1.22.6"
        }
      ],
      "warnings": []
    }
  }
}
```

### Ticker

Messages rotate, with one shown to the user every `interval` as long as it's not
disabled (interval=-1).

The user can disable the ticker or change the interval in the global config.

### Conditions and Versions

Every message can optionally include a condition and version constraint to limit
the message to matching conditions and DDEV versions.

Each element in the `conditions` array may contain a condition listed by
`ddev debug message-conditions`. It may be prefixed by a `!` to negate the
condition. All conditions must be met in order for a message to be displayed.
Unknown conditions are always met.

The field `versions` may contain a version constraint which must be met by the
current version of DDEV. More information about the supported constraints can
be found in the [Masterminds SemVer repository](https://github.com/Masterminds/semver#readme).

## Configuration

### Global Configuration

Users can configure remote config behavior in `~/.ddev/global_config.yaml`:

```yaml
remote_config:
  update_interval: 24  # Hours between updates
  remote_config_url: "https://raw.githubusercontent.com/ddev/remote-config/main/remote-config.jsonc"
  sponsorship_data_url: "https://ddev.com/s/sponsorship-data.json"
```

### Per-User Control

Users can disable features entirely:

```yaml
# Disable ticker messages (set in messages section)
messages:
  ticker_interval: -1

# Disable notifications and remote config updates
remote_config:
  update_interval: -1
```

## Architecture

### File Structure

The remote configuration system consists of several Go packages:

- `pkg/config/remoteconfig/`: Main package with interfaces and logic
- `pkg/config/remoteconfig/types/`: Public type definitions
- `pkg/config/remoteconfig/storage/`: File storage implementations
- `pkg/config/remoteconfig/downloader/`: JSONC download functionality

### Key Components

- **RemoteConfig Interface**: Main interface for displaying messages
- **Storage Interfaces**: Abstract file storage
- **Message Processing**: Condition and version constraint evaluation
- **State Management**: Tracking update times and message rotations

## Testing and Development

### Testing Remote Config Changes

To test changes to remote configuration:

1. **Change the upstream configuration** in [ddev/remote-config](https://github.com/ddev/remote-config) or use a fork/branch.

2. **Set up test configuration** in `~/.ddev/global_config.yaml`:

   ```yaml
   remote_config:
     update_interval: 1  # Update every hour for testing
     remote_config_url: "https://raw.githubusercontent.com/your-username/remote-config/your-test-branch/remote-config.jsonc"
     sponsorship_data_url: "https://ddev.com/s/sponsorship-data.json"  # Or your test URL
   ```

3. **Clear or edit cached data**:

   ```bash
   rm ~/.ddev/.state.yaml ~/.ddev/.remote-config
   ```

4. **Test with verbose output**:

   ```bash
   DDEV_VERBOSE=true ddev start <project>
   ```

### Testing with Debug Commands

Use the debug commands to validate your configuration:

```bash
# Download and validate remote config (tests your configured URL from global config)
ddev debug remote-data --type=remote-config --update-storage=false

# View the current cached config
ddev debug gob-decode ~/.ddev/.remote-config

# Verify message conditions work
ddev debug message-conditions
```
