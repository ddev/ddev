# Manifest

DDEV supports to download a `manifest.json` from the Github repository. At the
moment the manifest file contains messages which will be shown to the user see
next chapter but can also be easily enhanced in the future with more
information.

Here is an example of the `manifest.json` file:

```json
{
    "messages": {
        "infos": [
            {
                "message": "Ensure maintenance and further development of DDEV see https://github.com/sponsors/rfay."
            }
        ],
        "warnings": [
            {
                "message": "Please update your installation as soon as possible, there is a big security risk by using this version.",
                "versions": "<1.20"
            }
        ],
        "tips": {
            "messages": [
                "Did you know? You can restart your project at any time using `ddev restart`.",
                "Did you know? You can open a browser heading to your project with `ddev launch`."
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

### Tips

On every launch another message from the `tips` section will be shown to the
user.

## Testing

While running tests a Github token maybe required to avoid rate limits and can
be provided with the `DDEV_GITHUB_TOKEN` environment variable.

## Thanks

This feature was highly inspired by Composer.
