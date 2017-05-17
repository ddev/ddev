<h1>ddev Documentation</h1>

[ddev](https://github.com/drud/ddev) is a local web development environment management system powered by Docker and Docker Compose. It provides rapid, repeatable, and destructable environments geared for Drupal and WordPress development.

## System Requirements

- [docker](https://www.docker.com/community-edition)
- OS Support
  - macOS Sierra (fully supported with automated tests)
  - Linux
    * Ubuntu 16.04 LTS (fully supported with automated tests)
    * Debian Jessie (tested manually with automated tests planned)
    * Fedora 25 (tested manually with automated tests planned)
  - Windows 10 Pro (**experimental support!**)
    * See [Decisions and Plan for Linux & Windows Support](https://github.com/drud/ddev/issues/196#issuecomment-300178008) for more information and the various options on getting ddev operational on Windows 10.

We are open to expanding this list to include additional OSs as well as improve our existing support for the ones listed above. Please [let us know](https://github.com/drud/ddev/issues/new) if you hit an issue!

### Using ddev with other development environments
ddev requires ports 80 and 3306 to be available for use on your system when sites are running. If you are using another local development environment along side ddev, please ensure the other environment is turned off or otherwise not occupying ports 80 and 3306.

If you need to use another environment after using ddev, simply ensure all of your ddev sites are stopped or removed. ddev only occupies system ports when at least one site is running.

## Installation
### Installation Script - Linux and macOS

Linux and macOS end-users can use this line of code to your terminal to download, verify, and install ddev using our [install script](https://github.com/drud/ddev/blob/master/install_ddev.sh):
```
curl https://raw.githubusercontent.com/drud/ddev/master/install_ddev.sh | bash
```

### Manual Installation - Linux and macOS
You can also easily perform the installation manually if preferred:

- Download and extract the latest [ddev release](https://github.com/drud/ddev/releases) for your architecture.
- Make ddev executable: `chmod ugo+x ddev`
- Move ddev to /usr/local/bin: `mv ddev /usr/local/bin/` (may require sudo)
- Run `ddev` to test your installation. You should see usage output similar to below.

### Manual Installation - Windows

- Download and extract the latest [ddev release](https://github.com/drud/ddev/releases) for Windows.
- Copy `ddev.exe` into `%HOMEPATH%\AppData\Local\Microsoft\WindowsApps`, or otherwise add `ddev.exe` to a folder defined in your `PATH`
- Run `ddev` from a Command Prompt or PowerShell to test your installation. You should see usage output similar to below.

## Support
If you've encountered trouble using ddev, please use these resources to get help with your issue:

1. Please review the [ddev Documentation](https://ddev.readthedocs.io) to ensure your question isn't answered there.
2. Review the [ddev issue queue](https://github.com/drud/ddev/issues) to see if an issue similar to yours already exists.
3. If you've exhausted these options and still need help, please [file an issue](https://github.com/drud/ddev/issues/new) following the pre-populated guidelines and our [Contributing Guidelines](https://github.com/drud/ddev/blob/master/CONTRIBUTING.md) as best as possible.
