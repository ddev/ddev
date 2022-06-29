# Chocolatey deployment

The Makefile does basically everything needed in [quickstart doc](https://docs.chocolatey.org/en-us/create/create-packages#quick-start-guide) through the "choco pack" stage.

The package is built into `.gotmp/bin/windows_amd64/chocolatey`. It also may be untarred from a CircleCI artifact build.

The final steps are:

* Test the package (`cd [dir]; choco install -s .`)
* Push

```
cd <packagedir>
docker run --rm -v $PWD:/tmp/chocolatey -w /tmp/chocolatey linuturk/mono-choco apikey -k [API_KEY_HERE] -source https://push.chocolatey.org/
docker run --rm -v $PWD:/tmp/chocolatey -w /tmp/chocolatey linuturk/mono-choco push -s https://push.chocolatey.org/ --api-key=[API_KEY_HERE]
```
