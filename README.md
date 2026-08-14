<div align="center">

<a href="https://ddev.com">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://ddev.com/logos/dark-ddev.svg">
    <img alt="DDEV" src="https://ddev.com/logos/ddev.svg" width="320">
  </picture>
</a>

### Docker-based local development environments, ready in minutes

**Get the power of Docker without needing to know Docker.** Start PHP and Node.js projects in minutes, run many projects at the same time, and spend less time on setup.

[![Website](https://img.shields.io/badge/website-ddev.com-blue)](https://ddev.com)
[![Docs](https://img.shields.io/badge/docs-docs.ddev.com-blue)](https://docs.ddev.com)
[![Add-on registry](https://img.shields.io/badge/add--ons-addons.ddev.com-blue)](https://addons.ddev.com)
[![Discord](https://img.shields.io/discord/664580571770388500?logo=discord&logoColor=%23fff&label=Discord&link=https%3A%2F%2Fddev.com%2Fs%2Fdiscord)](https://ddev.com/s/discord)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)

[**Get Started**](https://ddev.com/get-started/) · [**Documentation**](https://docs.ddev.com) · [**Quickstarts**](https://docs.ddev.com/en/stable/users/quickstart/) · [**Add-on Registry**](https://addons.ddev.com) · [**Contributing**](CONTRIBUTING.md) · [**Sponsor**](https://ddev.com/sponsor)

</div>

---

## Get started

The quickest path is the [Get Started guide](https://ddev.com/get-started/): pick your operating system and it gives you the install commands and your first project, on one page.

If you'd rather read the full instructions:

1. 💻 **Check that DDEV runs where you work:** macOS, Windows 11, WSL2, Linux, and GitHub Codespaces. See the [system requirements](https://docs.ddev.com/en/stable/#system-requirements).
2. 📥 **Install a Docker provider and DDEV:** follow the [install instructions](https://docs.ddev.com/en/stable/users/install/).
3. 🏁 **Start your first project:** pick a [quickstart guide](https://docs.ddev.com/en/stable/users/quickstart/) for your CMS or framework, then run `ddev start`.

Want to try DDEV first? Run it in your browser, nothing to install: <a href="https://github.com/codespaces/new/ddev/ddev"><img src="https://github.com/codespaces/badge.svg" alt="Open in GitHub Codespaces" style="max-width: 100%; height: 20px;"></a>

## Why DDEV?

DDEV takes care of Docker for you, so you and your team can focus on your work. It comes with good defaults for everyday use, and you can change them when you need more.

- 🚀 **Ready in minutes:** good defaults and little setup. Just run `ddev start`.
- 📦 **One setup per project:** keep it in Git and share the same environment with your whole team.
- 🔧 **Your stack, your choice:** PHP [5.6 through 8.5](https://docs.ddev.com/en/stable/users/configuration/config/#php_version), [Nginx or Apache](https://docs.ddev.com/en/stable/users/configuration/config/#webserver_type), and [MariaDB, MySQL, or PostgreSQL](https://docs.ddev.com/en/stable/users/extend/database-types/), per project. And any version of Node.js you need.
- 🧩 **Easy to extend:** a growing set of [add-ons](https://addons.ddev.com) for extra services and integrations, and custom commands are ordinary shell scripts.
- 🖥️ **Runs everywhere:** macOS, Windows, WSL2, Linux, and GitHub Codespaces, on both ARM64 and AMD64.
- 🔒 **Included out of the box:** trusted HTTPS, Xdebug, database snapshots, and hosting integrations with [Upsun (formerly Platform.sh)](https://upsun.com), [Pantheon](https://pantheon.io), [Acquia](https://www.acquia.com), and others.
- 💙 **Run by the community:** free and open source, cared for by the nonprofit [DDEV Foundation](https://ddev.com/foundation/).

## Everyday commands

- `ddev start` and `ddev stop`: run or pause a project's containers
- `ddev import-db` and `ddev import-files`: load a database dump, or user-upload files such as Drupal `sites/default/files` and WordPress `wp-content/uploads`
- `ddev snapshot`: save and restore database state
- `ddev exec` and `ddev ssh`: run a command inside the web container, or open a shell in it
- `ddev logs` and `ddev list`: read container logs, or see every project on the machine
- `ddev share`: hand someone a temporary public URL for your local site

Run `ddev` for the full [command reference](https://docs.ddev.com/en/stable/users/usage/cli/).

## Community and support

- 💬 [Discord](https://ddev.com/s/discord): the fastest way to get help and talk to the maintainers
- 🧭 [Support](https://docs.ddev.com/en/stable/users/support/): every place the friendly community answers questions
- 🐛 [Issues](https://github.com/ddev/ddev/issues): report bugs and ask for new features
- 📚 [Documentation](https://docs.ddev.com): guides, references, and how-tos
- 📣 [Blog](https://ddev.com/blog/): news, releases, and community updates

## Contributing

- 🙋 “How can I contribute to DDEV?” in the [FAQ](https://docs.ddev.com/en/stable/users/usage/faq/): all the ways to help, code or not
- 📄 [Contributing](CONTRIBUTING.md): issues, pull requests, and Stack Overflow
- 🛠️ [Building, Testing, and Contributing](docs/content/developers/building-contributing.md): build the binary and run the tests

## Sponsor DDEV

DDEV is free and open source, maintained by the nonprofit [DDEV Foundation](https://ddev.com/foundation/) and paid for by its community. If DDEV saves you time, please consider [sponsoring the project](https://ddev.com/sponsor). Sponsorships fund the maintainers and infrastructure, and keep DDEV independent.

<div align="center">

<a href="https://ddev.com/#supporters">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://ddev.com/resources/featured-sponsors-darkmode.svg">
    <img alt="DDEV featured sponsors" src="https://ddev.com/resources/featured-sponsors.svg">
  </picture>
</a>

[![Sponsor DDEV](https://img.shields.io/badge/Sponsor-DDEV-blue?style=for-the-badge&logo=githubsponsors&logoColor=white)](https://ddev.com/sponsor)

</div>

## In-kind sponsors

DDEV depends on products and services provided free of charge by the generous companies below. Thanks!

| Sponsor | Contribution |
| ------- | ------------ |
| <a href="https://cloudsmith.com"><picture><source media="(prefers-color-scheme: dark)" srcset="https://ddev.com/logos/cloudsmith-dark.svg"><img alt="Cloudsmith" src="https://ddev.com/logos/cloudsmith.svg" height="40"></picture></a> | Linux package repository hosting is graciously provided by [Cloudsmith](https://cloudsmith.com). Cloudsmith is the only fully hosted, cloud-native, universal package management solution, that enables your organization to create, store and share packages in any format, to any place, with total confidence. |
| <a href="https://www.jetbrains.com"><picture><source media="(prefers-color-scheme: dark)" srcset="https://ddev.com/logos/jetbrains-dark.svg"><img alt="JetBrains" src="https://ddev.com/logos/jetbrains.svg" height="40"></picture></a> | [JetBrains](https://www.jetbrains.com) provides DDEV maintainers with licenses for its IDEs. |
| <a href="https://www.macstadium.com"><picture><source media="(prefers-color-scheme: dark)" srcset="https://ddev.com/logos/dark-mac-stadium.svg"><img alt="MacStadium" src="https://ddev.com/logos/mac-stadium.svg" height="40"></picture></a> | [MacStadium](https://www.macstadium.com) provides a cloud-hosted Apple silicon machine that DDEV uses for testing. |

## License

DDEV is licensed under the Apache License 2.0. See [LICENSE](LICENSE).
