---
search:
  boost: .5
---
# Remote Config

DDEV supports downloading a [`remote config`](https://github.com/ddev/remote-config/blob/main/remote-config.jsonc)
from the [`ddev/remote-config`](https://github.com/ddev/remote-config)
GitHub repository with messages that will be shown to the user as a "Tip of the Day". This feature
may be enhanced later with more information and filtering.

## Messages

### Notifications

The defined messages are shown to the user every `interval` as long as not
disabled (interval=0). Supported message types are `infos` and `warnings` where
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

To test, create a pull request on your fork or the main repository.

1. Run `prettier -c remote-config.jsonc` to make sure prettier will not complain. Run `prettier -w remote-config.jsonc` to get it to update the file.
2. For the fork `rfay` and branch `20240215_note_about_key_exp`, add configuration to your `~/.ddev/global_config.yaml`:

    ```yaml
    remote_config:
      update_interval: 1
      remote:
        owner: rfay
        repo: remote-config
        ref: 20240215_note_about_key_exp
        filepath: remote-config.jsonc
    ```

3. `rm ~/.ddev/.state.yaml ~/.ddev/.remote-config`
4. `DDEV_VERBOSE=true ddev start <project>`

Watch for failure to download or failure to parse the remote configuration.
