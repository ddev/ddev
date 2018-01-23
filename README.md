# ddev
[![CircleCI](https://circleci.com/gh/drud/ddev.svg?style=shield)](https://circleci.com/gh/drud/ddev) [![Go Report Card](https://goreportcard.com/badge/github.com/drud/ddev)](https://goreportcard.com/report/github.com/drud/ddev) ![project is maintained](https://img.shields.io/maintenance/yes/2018.svg)

![ddev logo](images/ddev_logo.png)

ddev is an open source tool that makes it simple to get local PHP development environments up and running in minutes. It's powerful and flexible as a result of its per-project environment configurations, which can be extended, version controlled, and shared. In short, ddev aims to allow development teams to use Docker in their workflow without the complexities of bespoke configuration.

## Getting Started

1. **Check System Requirements:** We support recent versions of macOS, Windows 10, and select Linux distributions that will run docker (ddev requires Docker and docker-compose). ([more info here](https://ddev.readthedocs.io/en/latest/#system-requirements)). 
2. **Install ddev:** Options include [macOS homebrew](https://ddev.readthedocs.io/en/latest/#homebrew-macos) (recommended), an [install script](https://ddev.readthedocs.io/en/latest/#installation-script-linux-and-macos), or [a manually download](https://ddev.readthedocs.io/en/latest/#manual-installation-linux-and-macos).
3. **Choose a CMS Quick Start Guide:** 
  - [WordPress](https://ddev.readthedocs.io/en/latest/users/cli-usage#wordpress-quickstart)
  - [Drupal 6 and 7](https://ddev.readthedocs.io/en/latest/users/cli-usage#drupal-6/7-quickstart)
  - [Drupal 8](https://ddev.readthedocs.io/en/latest/users/cli-usage#drupal-8-quickstart)
  - [TYPO3](https://ddev.readthedocs.io/en/latest/users/cli-usage#typo3-quickstart)

Having trouble? See our [support options below](#support). Additionally, you may have trouble if [another local development tool is already using port 80 or 443](https://ddev.readthedocs.io/en/latest/#using-ddev-with-other-development-environments).

## Current Feature List

* Quickly create multiple local web development environments based on a code repositories.
* Import database for a project you're working on.
* Import upload files to match the project (Drupal's sites/default/files or WordPress's wp-content/uploads).
* Pantheon integration - grab a Pantheon archive and work locally with the database and files.
* Run commands within the docker environment using `ddev exec`.
* View logs from the web and db containers.
* Use `ddev ssh` to explore the linux environment inside the container.
* List running projects.

Just running `ddev` will show you all the commands.

## Support
If you're having trouble using ddev, please use these resources to get help:

1. Please review the [ddev Documentation](https://ddev.readthedocs.io) to ensure your question isn't answered there.
2. Review the [ddev issue queue](https://github.com/drud/ddev/issues) to see if an issue similar to yours already exists.
3. If you've exhausted these options and still need help, please [file an issue](https://github.com/drud/ddev/issues/new) following the pre-populated guidelines and our [Contributing Guidelines](https://github.com/drud/ddev/blob/master/CONTRIBUTING.md) as best as possible.
4. We also have a channel (#ddev) in the [Drupal Slack](https://www.drupal.org/slack) account. We try to be very responsive, but replies may lag outside normal business hours in the US Mountain Time zone.

## Contributing
Interested in contributing to ddev? We would love your suggestions, contributions, and help! Please review our [Guidelines for Contributing](https://github.com/drud/ddev/blob/master/CONTRIBUTING.md), then [create an issue](https://github.com/drud/ddev/issues/new) or open a pull request!

## Addititional Information
* **Roadmap:** The [ddev roadmap](https://github.com/drud/ddev/wiki/roadmap) is managed by @rickmanelius. We love your input! Make requests in the [ddev issue queue](https://github.com/drud/ddev/issues).
