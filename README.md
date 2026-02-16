<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://ddev.com/logos/dark-ddev.svg">
  <img alt="DDEV logo with light and dark mode variants" src="https://ddev.com/logos/ddev.svg">
</picture>

---

[![ddev.com](https://img.shields.io/badge/DDEV-Website-blue)](https://ddev.com)
[![add-on registry](https://img.shields.io/badge/DDEV-Add--on_Registry-blue)](https://addons.ddev.com)
[![last commit](https://img.shields.io/github/last-commit/ddev/ddev)](https://github.com/ddev/ddev/commits)
[![Discord](https://img.shields.io/discord/664580571770388500?logo=discord&logoColor=%23fff&label=Discord&link=https%3A%2F%2Fddev.com%2Fs%2Fdiscord)](https://ddev.com/s/discord)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/ddev/ddev)
<a href="https://github.com/codespaces/new/ddev/ddev"><img src="https://github.com/codespaces/badge.svg" alt="Open in GitHub Codespaces" style="max-width: 100%; height: 20px;"></a>

[![Works with Mac | Windows | Linux | Cloud](https://img.shields.io/badge/works%20with-Mac%20%7C%20Windows%20%7C%20Linux%20%7C%20Cloud-blue.svg)](https://docs.ddev.com/en/stable/users/install/ddev-installation/)
[![Supported PHP 5.6 to 8.5](https://img.shields.io/badge/supported-PHP%208.5%20%7C%208.4%20%7C%208.3%20%7C%208.2%20%7C%208.1%20%7C%208.0%20%7C%207.4%20%7C%207.3%20%7C%207.2%20%7C%207.1%20%7C%207.0%20%7C%205.6-blue.svg)](https://docs.ddev.com/en/stable/users/configuration/config/#php_version)
[![Supported nginx & apache](https://img.shields.io/badge/supported-Nginx%20%7C%20Apache-blue)](https://docs.ddev.com/en/stable/users/configuration/config/#webserver_type)
[![Supported MariaDB, MySQL, PostgreSQL](https://img.shields.io/badge/supported-MariaDB%20%7C%20MySQL%20%7C%20PostgreSQL-blue)](https://docs.ddev.com/en/stable/users/extend/database-types/)

DDEV is an open-source tool for running local web development environments for PHP and Node.js, ready in minutes. Its powerful, flexible per-project environment configurations can be extended, version controlled, and shared. DDEV allows development teams to adopt a consistent Docker workflow without the complexities of bespoke configuration.

## Documentation

To check out live examples, docs, contributor live training, guides and more visit [ddev.com](https://ddev.com) and [docs.ddev.com](https://docs.ddev.com/en/stable/users/support)

## Questions

If you need help, our friendly community provides [great support](https://docs.ddev.com/en/stable/users/support/).

## Wonderful Sponsors

DDEV is an Apache License 2.0 open-source project with its ongoing development made possible entirely by the support of these awesome backers. If you'd like to join them, please consider [sponsoring DDEV development](https://github.com/sponsors/ddev).

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://ddev.com/resources/featured-sponsors-darkmode.svg">
  <img alt="DDEV Sponsor logos with light and dark mode variants" src="https://ddev.com/resources/featured-sponsors.svg">
</picture>

## Contributing

See “How can I contribute to DDEV?” in the [FAQ](https://docs.ddev.com/en/stable/users/usage/faq/), and the [Contributing](CONTRIBUTING.md) page.

![Overview of GitHub contributions](https://repobeats.axiom.co/api/embed/941b040a17921e974655fc01d7735aa350a53603.svg "Repobeats analytics image")

## Get Started

1. **Check [System Requirements](https://docs.ddev.com/):** macOS (Intel and Apple Silicon), Windows 10/11, WSL2, Linux, and [GitHub Codespaces](https://github.com/codespaces).
2. **Install a [Docker provider and DDEV](https://docs.ddev.com/en/stable/users/install/)**.
3. **Try a [CMS Quick Start Guide](https://docs.ddev.com/en/stable/users/quickstart/)**.

Additionally, [https://ddev.com/get-started/](https://ddev.com/get-started/) provides an up-to-date getting-started guide.

## Highlighted Features

* Quickly create local web development environments based on code repositories, with minimal configuration.
* Import a database to any of your local environments.
* Import upload files to match the project (e.g. Drupal sites/default/files or WordPress `wp-content/uploads`).
* Customizable integration with hosting platforms like [Upsun (formerly Platform.sh)](https://upsun.com), [Pantheon](https://pantheon.io), [Acquia](https://www.acquia.com) and others.
* Run commands within the Docker environment using `ddev exec`.
* View logs from the web and database containers.
* Use `ddev ssh` to explore the Linux environment inside the container.
* List running projects with `ddev list`.
* Snapshot databases with `ddev snapshot`.
* Temporarily share your development website with others using `ddev share`.
* Create custom commands as simple shell scripts.
* Enjoy effortless, trusted HTTPS support.
* Extend and customize environments as much (or as little!) as you need to.

Run `ddev` to see all the [commands](https://docs.ddev.com/en/stable/users/usage/cli/).
