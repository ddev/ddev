<h1>ddev Documentation</h1>

[ddev](https://github.com/drud/ddev) is an open source tool that makes it dead simple to get local PHP development environments up and running within minutes. It's powerful and flexible as a result of its per-project environment configurations, which can be extended, version controlled, and shared. In short, ddev aims to allow development teams to use Docker in their workflow without the complexities of bespoke configuration.



## System Requirements

- [Docker](https://www.docker.com/community-edition) version 17.05 or higher. Linux users make sure you do the [post-install steps](https://docs.docker.com/install/linux/linux-postinstall/#manage-docker-as-a-non-root-user)
- docker-compose 1.10.0 and higher (bundled with Docker in Docker for Mac and Docker for Windows)
- OS Support
  - macOS Sierra and higher (macOS 10.12 and higher but it probably runs anywhere docker runs)
  - Linux: Most recent Linux distributions which can run Docker are fine. This includes at least Ubuntu 14.04+, Debian Jessie+, Fedora 25+. Make sure to follow the docker-ce [post-install steps](https://docs.docker.com/install/linux/linux-postinstall/#manage-docker-as-a-non-root-user)
  - Windows 10 Pro

We are open to expanding this list to include additional OSs as well as improve our existing support for the ones listed above. Please [let us know](https://github.com/drud/ddev/issues/new) if you hit an issue!

### Using ddev with other development environments
ddev by default uses ports 80 and 443 on your system when projects are running. If you are using another local development environment you can either stop the other environment or configure ddev to use different ports. See [troubleshooting](https://ddev.readthedocs.io/en/latest/users/troubleshooting/#webserver-ports-are-already-occupied-by-another-webserver) for more detailed problemsolving.

## Installation

_When upgrading, please check the [release notes](https://github.com/drud/ddev/releases) for actions you might need to take on each project._

### Homebrew - macOS

For macOS users, we recommend downloading, installing, and upgrading via [homebrew](https://brew.sh/):
```
brew tap drud/ddev && brew install ddev
```
Later, to upgrade to a newer version of ddev, run:
```
brew upgrade ddev
```

### Installation/Upgrade Script - Linux and macOS

Linux and macOS end-users can use this line of code to your terminal to download, verify, and install (or upgrade) ddev using our [install script](https://github.com/drud/ddev/blob/master/install_ddev.sh):

```
curl https://raw.githubusercontent.com/drud/ddev/master/install_ddev.sh | bash
```

Later, to upgrade ddev to the latest version, just run this again.

### Manual Installation or Upgrade - Linux and macOS

You can also easily perform the installation or upgrade manually if preferred. ddev is just a single executable, no special installation is actually required, so for all operating systems, the installation is just copying ddev into place where it's in the system path.

- Download and extract the latest [ddev release](https://github.com/drud/ddev/releases) for your architecture.
- Move ddev to /usr/local/bin: `mv ddev /usr/local/bin/` (may require sudo), or another directory in your `$PATH` as preferred.
- Run `ddev` to test your installation. You should see ddev's command usage output.

### Installation via package managers - Linux

Some Linux distributions may package ddev in a way that's convenient for your distro. Right now, we are aware of packages for the following distros:

	* [Arch Linux (AUR)](https://aur.archlinux.org/packages/ddev-bin/)

Note that third party packaging is encouraged, but only supported on a best-effort basis.

### Installation or Upgrade - Windows

- A windows installer is provided in each [ddev release](https://github.com/drud/ddev/releases) (`ddev_windows_installer.<version>.exe`). Run that and it will do the full installation for you. If you get a Windows Defender Smartscreen warning "Windows protected your PC", click "More info" and then "Run anyway". Open a new terminal or cmd window and start using ddev.

### Versioning

The DDEV project is committed to supporting [Semantic Version 2.0.0](https://semver.org/). Additional context on this decision can be read in [Ensure ddev is properly utilizing Semantic Versioning](https://github.com/drud/ddev/issues/352).

## Support

- [ddev Documentation](https://ddev.readthedocs.io)
- [ddev StackOverflow](https://stackoverflow.com/questions/tagged/ddev) for support and frequently asked questions
- [ddev issue queue](https://github.com/drud/ddev/issues) for bugs and feature requests
- The `#ddev` channel in [Drupal Slack](https://drupal.slack.com/messages/C5TQRQZRR) and [TYPO3 Slack](https://typo3.slack.com/messages/C8TRNQ601) for interactive, immediate community support
