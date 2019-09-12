<h1>Frequently-Asked Questions (FAQ)</h1>

* **What operating systems does DDEV-Local work on?** DDEV-Local works nearly anywhere Docker can be installed, including macOS, Windows 10 Pro/Enterprise,  Windows 10 Home, and Linux variant we've ever tried. It also runs in many Linux-like environments, for example ChromeOS (in Linux machine) and Windows 10's WSL2. In general, ddev works the same on each of these platforms, as all the important work is done inside identical Docker containers.
* **How can I troubleshoot what's going wrong?** See the [troubleshooting](troubleshooting.md) and [Docker troubleshooting](docker_installation.md#troubleshooting) sections of the docs.
* **Do I need to install PHP or Composer or Nginx or MySQL on my computer?** Absolutely *not*. All of these tools live inside ddev's docker containers, so you need only Docker and ddev. This is especially handy for Windows users where there's a bit more friction installing those tools.
* **How do I get support?** See the (many) [support options](../index.md#support), including Slack, Gitter, Stack Overflow and others.
* **How can I get the best performance?** Docker's normal mounting can be slow, especially on macOS. See the [Performance](performance.md) section for speed-up options including NFS mounting.
* **How can I check that Docker is working?** The Docker Installation docs have a [full Docker troubleshooting section](docker_installation.md#troubleshooting), including a single `docker run` command that will verify whether everything is set up.
* **Can I run ddev and also other Docker or non-Docker development environments at the same time?** Yes, you can, as long as they're configured with different ports. But it's easiest to shut down one before using the other. For example, if you use Lando for one project, do a `lando poweroff` before using ddev, and then do a `ddev poweroff` before using Lando again. If you run nginx or apache locally, just stop them before using ddev. More information is in the [troubleshooting](troubleshooting.md) section.
* **How can I contribute to DDEV-Local?** We love contributions of knowledge, support, docs, and code, and invite you to all of them. Make an issue or PR to the [main repo](https://github.com/drud/ddev). Add your external resource to [awesome-ddev](https://github.com/drud/awesome-ddev). Add your recipe or HOWTO to [ddev-contrib]](https://github.com/drud/ddev-contrib). Help others in [Stack Overflow](https://stackoverflow.com/tags/ddev) or [Slack](../index.md#support) or [gitter](https://gitter.im/drud/ddev). 



