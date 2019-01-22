# ddev
[![CircleCI](https://circleci.com/gh/drud/ddev.svg?style=shield)](https://circleci.com/gh/drud/ddev) [![Go Report Card](https://goreportcard.com/badge/github.com/drud/ddev)](https://goreportcard.com/report/github.com/drud/ddev) ![project is maintained](https://img.shields.io/maintenance/yes/2019.svg)

![ddev logo](images/ddev_logo.png)

ddev is an open source tool that makes it simple to get local PHP development environments up and running in minutes. It's powerful and flexible as a result of its per-project environment configurations, which can be extended, version controlled, and shared. In short, ddev aims to allow development teams to use Docker in their workflow without the complexities of bespoke configuration.

## Getting Started

1. **Check System Requirements:** We support recent versions of macOS, Windows 10, and Linux distributions that will run docker-ce (ddev requires Docker and docker-compose). ([more info here](https://ddev.readthedocs.io/en/stable/#system-requirements)). 
2. **Install ddev:** [Options include](https://ddev.readthedocs.io/en/stable/#installation) macOS homebrew (recommended), an install script, or manual installation.
3. **Choose a CMS Quick Start Guide:** 
  - [WordPress](https://ddev.readthedocs.io/en/stable/users/cli-usage#wordpress-quickstart)
  - [Drupal 6 and 7](https://ddev.readthedocs.io/en/stable/users/cli-usage#drupal-6/7-quickstart)
  - [Drupal 8](https://ddev.readthedocs.io/en/stable/users/cli-usage#drupal-8-quickstart)
  - [Backdrop](https://ddev.readthedocs.io/en/stable/users/cli-usage/#backdrop-quickstart) 
  - [TYPO3](https://ddev.readthedocs.io/en/stable/users/cli-usage#typo3-quickstart)

Having trouble? See our [support options below](#support). You might have trouble if [another local development tool is already using port 80 or 443](https://ddev.readthedocs.io/en/stable/#using-ddev-with-other-development-environments).

## Current Feature List

* Quickly create multiple local web development environments based on a code repositories.
* Import database for a project you're working on.
* Import upload files to match the project (e.g. Drupal's sites/default/files or WordPress's wp-content/uploads).
* Pantheon integration - grab a Pantheon archive and work locally with the database and files.
* Run commands within the docker environment using `ddev exec`.
* View logs from the web and db containers.
* Use `ddev ssh` to explore the linux environment inside the container.
* List running projects.

Just running `ddev` will show you all the commands.

## Support
If you're having trouble using ddev, please use these resources to get help:

1. See the [ddev Documentation](https://ddev.readthedocs.io).
2. Review [Stack Overflow DDEV-Local questions and answers](https://stackoverflow.com/tags/ddev) (or ask a question there! We get notified when you ask.)
3. The [ddev issue queue](https://github.com/drud/ddev/issues) may have an issue related to your problem.
4. For suspected bugs or feature requests, [file an issue](https://github.com/drud/ddev/issues/new).
5. The `#ddev` channel in [Drupal Slack](https://drupal.slack.com/messages/C5TQRQZRR) and [TYPO3 Slack](https://typo3.slack.com/messages/C8TRNQ601) for interactive, immediate community support


## Contributing
Interested in contributing to ddev? We would love your suggestions, contributions, and help! Please review our [Guidelines for Contributing](https://github.com/drud/ddev/blob/master/CONTRIBUTING.md), then [create an issue](https://github.com/drud/ddev/issues/new) or open a pull request!

## Addititional Information
* **Roadmap:** See the [ddev roadmap](https://github.com/drud/ddev/wiki/roadmap). We love your input! Make requests in the [ddev issue queue](https://github.com/drud/ddev/issues).
