# ddev Documentation

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
