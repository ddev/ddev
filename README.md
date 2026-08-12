<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://ddev.com/logos/dark-ddev.svg">
  <img alt="DDEV logo with light and dark mode variants" src="https://ddev.com/logos/ddev.svg">
</picture>

[![ddev.com](https://img.shields.io/badge/DDEV-Website-blue)](https://ddev.com)
[![documentation](https://img.shields.io/badge/DDEV-Documentation-blue)](https://docs.ddev.com)
[![add-on registry](https://img.shields.io/badge/DDEV-Add--on_Registry-blue)](https://addons.ddev.com)
[![Discord](https://img.shields.io/discord/664580571770388500?logo=discord&logoColor=%23fff&label=Discord&link=https%3A%2F%2Fddev.com%2Fs%2Fdiscord)](https://ddev.com/s/discord)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/ddev/ddev)

DDEV is an open-source tool for running local web development environments for PHP and Node.js, ready in minutes. Its per-project configuration can be extended, version controlled, and shared, so a whole team gets the same workflow without anyone maintaining bespoke setup. It depends on a Docker provider under the hood, but you'll hardly ever know it.

## Get Started

1. **Check that DDEV runs where you work:** macOS, Windows 11, WSL2, Linux, and [GitHub Codespaces](https://github.com/codespaces). See [system requirements](https://docs.ddev.com/en/stable/users/install/ddev-installation/).
2. **Install a [Docker provider and DDEV](https://docs.ddev.com/en/stable/users/install/).**
3. **Follow a [CMS quick start guide](https://docs.ddev.com/en/stable/users/quickstart/).**

[https://ddev.com/get-started/](https://ddev.com/get-started/) is the up-to-date getting-started guide, and you can try DDEV without installing anything: <a href="https://github.com/codespaces/new/ddev/ddev"><img src="https://github.com/codespaces/badge.svg" alt="Open in GitHub Codespaces" style="max-width: 100%; height: 20px;"></a>

Each project chooses its own stack: PHP [5.6 through 8.5](https://docs.ddev.com/en/stable/users/configuration/config/#php_version), [Nginx or Apache](https://docs.ddev.com/en/stable/users/configuration/config/#webserver_type), and [MariaDB, MySQL, or PostgreSQL](https://docs.ddev.com/en/stable/users/extend/database-types/).

## What DDEV Gives You

### Environments

* Create a local environment from a code repository with minimal configuration.
* Get trusted HTTPS without setting anything up.
* Extend and customize as much (or as little!) as you need to.

### Data

* Import a database into any of your local environments.
* Import upload files to match the project, such as Drupal `sites/default/files` or WordPress `wp-content/uploads`.
* Snapshot databases with `ddev snapshot`.
* Integrate with hosting platforms like [Upsun (formerly Platform.sh)](https://upsun.com), [Pantheon](https://pantheon.io), [Acquia](https://www.acquia.com), and others.

### Workflow

* Run commands inside the Docker environment with `ddev exec`, or explore it with `ddev ssh`.
* Read web and database container logs, and list running projects with `ddev list`.
* Share your development site with others temporarily using `ddev share`.
* Add custom commands as ordinary shell scripts.

Run `ddev` to see all the [commands](https://docs.ddev.com/en/stable/users/usage/cli/).

## Documentation and Support

[docs.ddev.com](https://docs.ddev.com) has the full documentation, and [ddev.com](https://ddev.com) has live examples, contributor live training, and guides.

If you need help, our friendly community provides [great support](https://docs.ddev.com/en/stable/users/support/).

## Contributing

Start with “How can I contribute to DDEV?” in the [FAQ](https://docs.ddev.com/en/stable/users/usage/faq/) and the [Contributing](CONTRIBUTING.md) page. To build and test the code, see [Building, Testing, and Contributing](docs/content/developers/building-contributing.md).

## Sponsors

DDEV's ongoing development is made possible entirely by the support of these awesome backers. If you'd like to join them, please consider [sponsoring DDEV development](https://ddev.com/sponsor).

<a href="https://ddev.com/#supporters">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://ddev.com/resources/featured-sponsors-darkmode.svg">
    <img alt="DDEV featured sponsors" src="https://ddev.com/resources/featured-sponsors.svg">
  </picture>
</a>

## License

DDEV is licensed under the Apache License 2.0. See [LICENSE](LICENSE).
