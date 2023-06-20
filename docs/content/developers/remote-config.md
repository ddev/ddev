# Remote Config

DDEV supports downloading a remote config from the [`ddev/remote-config`](https://github.com/ddev/remote-config) GitHub repository with messages that will be shown to the user. This feature could be enhanced later with more information and configuration.

Here is an example of a `remote-config.jsonc` file:

```jsonc
{
    // Update interval of the remote config in hours
    "update-interval": 10,
    "remote": {
        "owner": "ddev",
        "repo": "remote-config",
        "ref": "live",
        "filepath": "remote-config",
    },

    // Messages shown to the user
    "messages": {
        // All notifications are shown once in an interval
        "notifications": {
            "disabled": false,
            "interval": 20,
            "infos": [
                {
                    "message": "Ensure maintenance and further development of DDEV, see https://ddev.com/support-ddev/.",
                    "conditions": [],
                    "versions": ""
                },
            ],
            "warnings": [
                {
                    "message": "Please update your installation as soon as possible, there is a big security risk by using this version.",
                    "conditions": [],
                    "versions": "<1.20"
                },
            ],
        },

        // One ticker messages is shown once in an interval and rotated to the
        // next afterwards
        "ticker": {
            "disabled": false,
            "interval": 5,
            "messages": [
                {
                    "message": "Did you know? You can restart your project at any time using `ddev restart`.",
                    "conditions": [],
                    "versions": ""
                },
                {
                    "message": "Did you know? You can open a browser heading to your project with `ddev launch`."
                },
            ]
        }
    }
}
```

## Messages

### Notifications

The defined messages are shown to the user every `interval` as long as not
`disabled`. Supported message types are `infos` and `warnings` where `infos`
are printed in a yellow box and `warnings` in a red box.

Messages will be shown as configured in the `remote-config` repository and the user cannot influence them.

### Ticker

Messages rotate, with one shown to the user every `interval` as long as it’s not `disabled`.

The user can disable the ticker or change the interval in the global config.

### Conditions and Versions

Every message can optionally include a condition and version constraint to limit the message to matching conditions and DDEV versions.

Each element in the `conditions` array may contain a condition listed by `ddev debug message-conditions`. It may be prefixed by a `!` to negate the condition. All conditions must be met in order for a message to be displayed. Unknown conditions are always met.

The field `versions` may contain a version constraint which must be met by the
current version of DDEV. More information about the supported constraints can
be found in the [Masterminds SemVer repository](https://github.com/Masterminds/semver#readme).

## Testing

While running tests a Github token maybe required to avoid rate limits and can
be provided with the `DDEV_GITHUB_TOKEN` environment variable.
