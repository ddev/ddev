<h1>ddev Documentation</h1>

[ddev](https://github.com/drud/ddev) is an open source tool that makes it dead simple to get local PHP development environments up and running within minutes. It's powerful and flexible as a result of its per-project environment configurations, which can be extended, version controlled, and shared. In short, ddev aims to allow development teams to use Docker in their workflow without the complexities of bespoke configuration.



## System Requirements

- [Docker](https://www.docker.com/community-edition) version 17.05 or higher
- docker-compose 1.10.0 and higher (bundled with Docker in Docker for Mac and Docker for Windows)
- OS Support 
  - macOS Sierra and higher (macOS 10.12 and higher)
  - Linux (See [Linux notes](users/linux_notes.md)): Most recent Linux distributions which can run Docker are fine. This includes at least Ubuntu 14.04+, Debian Jessie+, Fedora 25+
  - Windows 10 Pro

We are open to expanding this list to include additional OSs as well as improve our existing support for the ones listed above. Please [let us know](https://github.com/drud/ddev/issues/new) if you hit an issue!

### Using ddev with other development environments
ddev requires ports 80 and 443 to be available for use on your system when projects are running. If you are using another local development environment alongside ddev, please ensure the other environment is turned off or otherwise not occupying ports 80 and 443.

If you need to use another environment after using ddev, just stop or remove all of your ddev projects. ddev only occupies system ports when at least one project is running.

## Installation/Upgrade
### Homebrew - macOS

For macOS users, we recommend downloading, installing, and upgrading via [homebrew](https://brew.sh/):
```
brew tap drud/ddev && brew install ddev
```
Later, to upgrade to a newer version of ddev, simply run:
```
brew upgrade ddev
```

### Installation Script - Linux and macOS

Linux and macOS end-users can use this line of code to your terminal to download, verify, and install ddev using our [install script](https://github.com/drud/ddev/blob/master/install_ddev.sh):
```
curl https://raw.githubusercontent.com/drud/ddev/master/install_ddev.sh | bash
```

Later, to upgrade ddev to the newest version, just run `install_ddev.sh` again.


### Manual Installation - Linux and macOS
You can also easily perform the installation manually if preferred:

- Download and extract the latest [ddev release](https://github.com/drud/ddev/releases) for your architecture.
- Make ddev executable: `chmod ugo+x ddev`
- Move ddev to /usr/local/bin: `mv ddev /usr/local/bin/` (may require sudo), or another directory in your `$PATH` as preferred.
- Run `ddev` to test your installation. You should see ddev's command usage output.

### Manual Installation - Windows

- Download and extract the latest [ddev release](https://github.com/drud/ddev/releases) for Windows.
- Copy `ddev.exe` into `%HOMEPATH%\AppData\Local\Microsoft\WindowsApps`, or otherwise add `ddev.exe` to a folder defined in your `PATH`
- Run `ddev` from a Command Prompt or PowerShell to test your installation. You should see ddev's command usage output.

### Versioning

The DDEV project is committed to supporting [Semantic Version 2.0.0](https://semver.org/). Additional context on this decision can be read in [Ensure ddev is properly utilizing Semantic Versioning](https://github.com/drud/ddev/issues/352).

## Support
If you've encountered trouble using ddev, please use these resources to get help with your issue:

1. Please review the [ddev Documentation](https://ddev.readthedocs.io) to ensure your question isn't answered there.
2. Review the [ddev issue queue](https://github.com/drud/ddev/issues) to see if an issue similar to yours already exists.
3. If you've exhausted these options and still need help, please [file an issue](https://github.com/drud/ddev/issues/new) following the pre-populated guidelines and our [Contributing Guidelines](https://github.com/drud/ddev/blob/master/CONTRIBUTING.md) as best as possible.
