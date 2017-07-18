[![CircleCI](https://circleci.com/gh/drud/ddev.svg?style=shield)](https://circleci.com/gh/drud/ddev) [![Go Report Card](https://goreportcard.com/badge/github.com/drud/ddev)](https://goreportcard.com/report/github.com/drud/ddev) ![project is maintained](https://img.shields.io/maintenance/yes/2017.svg)


# ddev

ddev is an open source tool that makes it dead simple to get local PHP development environments up and running within minutes. It's powerful and flexible as a result of its per-project environment configurations, which can be extended, version controlled, and shared. In short, ddev aims to allow development teams to use Docker in their workflow without the complexities of bespoke configuration.

## Roadmap

Each DRUD product has a dedicated product owner, who serves as the primary advocate for customers and end-users when making decisions regarding the public roadmap. For the [ddev roadmap](https://github.com/drud/ddev/wiki/roadmap), @rickmanelius is currently serving as the product owner.

We use the longer-term roadmap to prioritize short-term sprints. Please review the [ddev roadmap](https://github.com/drud/ddev/wiki/roadmap) and [ddev issue queue](https://github.com/drud/ddev/issues) to see what's on the horizon.

## System Requirements

- [Docker](https://www.docker.com/community-edition) version 17.05 or greater
- OS Support
  - macOS Sierra
  - Linux
    * Ubuntu 16.04 LTS
    * Debian Jessie
    * Fedora 25
  - Windows 10 Pro (**experimental support!**)
    * See [Decisions and Plan for Linux & Windows Support](https://github.com/drud/ddev/issues/196#issuecomment-300178008) for more information and the various options on getting ddev operational on Windows 10.

We are open to expanding this list to include additional OSs as well as improve our existing support for the ones listed above. Please [let us know](https://github.com/drud/ddev/issues/new) if you hit an issue!

### Using ddev with other development environments
ddev requires ports 80 and 3306 to be available for use on your system when sites are running. If you are using another local development environment alongside ddev, please ensure the other environment is turned off or otherwise not using ports 80 and 3306.

If you need to use another environment after using ddev, simply ensure all of your ddev sites are stopped or removed. ddev only uses system ports when at least one site is running.

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
- Move ddev to /usr/local/bin: `mv ddev /usr/local/bin/` (may require sudo), or another directory in your `$PATH` as preferred.
- Run `ddev` to test your installation. You should see ddev's command usage output.

### Manual Installation - Windows
- Download and extract the latest [ddev release](https://github.com/drud/ddev/releases) for Windows.
- Copy `ddev.exe` into `%HOMEPATH%\AppData\Local\Microsoft\WindowsApps`, or otherwise add `ddev.exe` to a folder defined in your `PATH`
- Run `ddev` from a Command Prompt or PowerShell to test your installation. You should see ddev's command usage output.

---

## Usage
```
➜  ddev
This Command Line Interface (CLI) gives you the ability to interact with the ddev to create a local development environment.

Usage:
  ddev [command]

Available Commands:
  config       Create or modify a ddev application config in the current directory
  describe     Get a detailed description of a running ddev site.
  exec         Execute a Linux shell command in the webserver container.
  hostname     Manage your hostfile entries.
  import-db    Import the database of an existing site to the local dev environment.
  import-files Import the uploaded files directory of an existing site to the default public upload directory of your application.
  list         List applications that exist locally
  logs         Get the logs from your running services.
  restart      Restart the local development environment for a site.
  remove       Remove an application's local services.
  sequelpro    Easily connect local site to sequelpro
  ssh          SSH to an app container.
  start        Start the local development environment for a site.
  stop         Stop an application's local services.
  version      print ddev version and component versions

Use "ddev [command] --help" for more information about a command.
```

## Getting Started - Documentation
Once you've installed ddev, check out the [ddev Documentation Site](https://ddev.readthedocs.io) for information on how to get started and how to use ddev.

## Support
If you've encountered trouble using ddev, please use these resources to get help with your issue:

1. Please review the [ddev Documentation](https://ddev.readthedocs.io) to ensure your question isn't answered there.
2. Review the [ddev issue queue](https://github.com/drud/ddev/issues) to see if an issue similar to yours already exists.
3. If you've exhausted these options and still need help, please [file an issue](https://github.com/drud/ddev/issues/new) following the pre-populated guidelines and our [Contributing Guidelines](https://github.com/drud/ddev/blob/master/CONTRIBUTING.md) as best as possible.

## Contributing
Interested in contributing to ddev? We would love your suggestions, contributions, and help! Please review our [Guidelines for Contributing](https://github.com/drud/ddev/blob/master/CONTRIBUTING.md), then [create an issue](https://github.com/drud/ddev/issues/new) or open a pull request!
