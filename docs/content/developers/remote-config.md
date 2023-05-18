# Remote config

DDEV supports to download a remote config from the `ddev/remote-config` Github
repository. At the moment the remote config contains messages which will be
shown to the user see next chapter but can also be easily enhanced in the
future with more information and configuration.

Here is an example of a `remote-config.jsonc` file:

```jsonc
{
    // Update interval of the remote config in hours
    "update-interval": 6,

    // Messages shown to the user
    "messages": {
        // infos and warnings are shown on almost every ddev command, please
        // use with caution!
        "infos": [
            {
                "message": "Ensure maintenance and further development of DDEV, see https://ddev.com/support-ddev/.",
                "conditions": [],
                "versions": ""
            }
        ],
        "warnings": [
            {
                "message": "Please update your installation as soon as possible, there is a big security risk by using this version.",
                "versions": "<1.20"
            }
        ],

        // ticker messages are are shown once in an interval, once a day by
        // default, and are rotated once a message was shown.
        "ticker": {
            "interval": 24,
            "messages": [
                {
                    "message": "Did you know? You can restart your project at any time using `ddev restart`.",
                    "conditions": [],
                    "versions": ""
                },
                {
                    "message": "Did you know? You can open a browser heading to your project with `ddev launch`."
                }
            ]
        }
    }
}
```

## Messages

### Infos and Warnings

The defined messages are shown to the user on every usage of DDEV. Supported
message types are `infos` and `warnings`. Every message can optionally include
a version constraint to limit the message to matching DDEV versions only.

More information about the supported constraints can be found in the
[Masterminds SemVer](https://github.com/Masterminds/semver#readme) repository.

### Ticker

On every launch another message from the `ticker` section will be shown to the
user.

## Testing

While running tests a Github token maybe required to avoid rate limits and can
be provided with the `DDEV_GITHUB_TOKEN` environment variable.
