# ddev
[![CircleCI](https://circleci.com/gh/drud/ddev.svg?style=shield)](https://circleci.com/gh/drud/ddev) [![Go Report Card](https://goreportcard.com/badge/github.com/drud/ddev)](https://goreportcard.com/report/github.com/drud/ddev) ![project is maintained](https://img.shields.io/maintenance/yes/2017.svg)

![ddev logo](images/ddev_logo.png)

ddev is an open source tool that makes it dead simple to get local PHP development environments up and running within minutes. It's powerful and flexible as a result of its per-project environment configurations, which can be extended, version controlled, and shared. In short, ddev aims to allow development teams to use Docker in their workflow without the complexities of bespoke configuration.

## Getting Started

1. **Check System Requirements:** We support macOS High Sierra, Windows 10, and Linux ([more info here](https://ddev.readthedocs.io/en/latest/#system-requirements)).
2. **Install ddev:** Options include [macOS homebrew](https://ddev.readthedocs.io/en/latest/#homebrew-macos) (recommended), an [install script](https://ddev.readthedocs.io/en/latest/#installation-script-linux-and-macos), or [a manually download](https://ddev.readthedocs.io/en/latest/#manual-installation-linux-and-macos).
3. **Choose a CMS Quick Start Guide:** 
  - [Wordpress](https://ddev.readthedocs.io/en/latest/users/cli-usage#wordpress-quickstart)
  - [Drupal 7](https://ddev.readthedocs.io/en/latest/users/cli-usage#drupal-7-quickstart)
  - [Drupal 8](https://ddev.readthedocs.io/en/latest/users/cli-usage#drupal-8-quickstart)

Having trouble? See our [support options below](#support). Additionally, you may have trouble if another local development tool is already using port 80 or 3306. See our troubleshooting docs for more info.

## Current Feature List

```
➜  ddev
This Command Line Interface (CLI) gives you the ability to interact with the ddev to create a local development environment.

Usage:
  ddev [command]

Available Commands:
  auth-pantheon Provide a machine token for the global pantheon auth.
  config        Create or modify a ddev application config in the current directory
  describe      Get a detailed description of a running ddev site.
  exec          Execute a shell command in the container for a service. Uses the web service by default.
  help          Help about any command
  hostname      Manage your hostfile entries.
  import-db     Import the database of an existing site to the local dev environment.
  import-files  Import the uploaded files directory of an existing site to the default public upload directory of your application.
  list          List applications that exist locally
  logs          Get the logs from your running services.
  pull          Import files and database using a configured provider plugin.
  remove        Remove the local development environment for a site.
  restart       Restart the local development environment for a site.
  sequelpro     Easily connect local site to sequelpro
  ssh           Starts a shell session in the container for a service. Uses web service by default.
  start         Start the local development environment for a site.
  stop          Stop the local development environment for a site.
  version       print ddev version and component versions

Flags:
  -h, --help          help for ddev
  -j, --json-output   If true, user-oriented output will be in JSON format.

Use "ddev [command] --help" for more information about a command.
```

## Support
If you've encountered trouble using ddev, please use these resources to get help with your issue:

1. Please review the [ddev Documentation](https://ddev.readthedocs.io) to ensure your question isn't answered there.
2. Review the [ddev issue queue](https://github.com/drud/ddev/issues) to see if an issue similar to yours already exists.
3. If you've exhausted these options and still need help, please [file an issue](https://github.com/drud/ddev/issues/new) following the pre-populated guidelines and our [Contributing Guidelines](https://github.com/drud/ddev/blob/master/CONTRIBUTING.md) as best as possible.
4. We also have a channel (#ddev) in the Drupal Slack account. We're fairly responsive to questions and requests during normal business hours (US Mountain Time).

## Contributing
Interested in contributing to ddev? We would love your suggestions, contributions, and help! Please review our [Guidelines for Contributing](https://github.com/drud/ddev/blob/master/CONTRIBUTING.md), then [create an issue](https://github.com/drud/ddev/issues/new) or open a pull request!

## Addititional Information
* **Roadmap:** The [ddev roadmap is publically available](https://github.com/drud/ddev/wiki/roadmap) and managed by @rickmanelius. Additional requests should be added to the [ddev issue queue](https://github.com/drud/ddev/issues).