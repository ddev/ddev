# Remote Config

DDEV supports downloading a [`remote config`](https://github.com/ddev/remote-config/blob/main/remote-config.jsonc)
from the [`ddev/remote-config`](https://github.com/ddev/remote-config)
GitHub repository with messages that will be shown to the user. This feature
could be enhanced later with more information and configuration.

## Messages

### Notifications

The defined messages are shown to the user every `interval` as long as not
disabled (interval=0). Supported message types are `infos` and `warnings` where
`infos` are printed in a yellow box and `warnings` in a red box.

Messages will be shown as configured in the `remote-config` repository and the
user cannot influence them.

### Ticker

Messages rotate, with one shown to the user every `interval` as long as it’s not
disabled (interval=0).

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

## Testing

While running tests a GitHub token maybe required to avoid rate limits and can
be provided with the `DDEV_GITHUB_TOKEN` environment variable.
