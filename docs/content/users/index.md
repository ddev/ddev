# Get Started with DDEV

[DDEV](https://github.com/drud/ddev) is an open source tool that makes it dead simple to get local PHP development environments up and running within minutes. It's powerful and flexible as a result of its per-project environment configurations, which can be extended, version controlled, and shared. In short, DDEV aims to allow development teams to use Docker in their workflow without the complexities of bespoke configuration.

DDEV works great on macOS, Windows WSL2, traditional Windows, and Linux. It works perfectly on amd64 and arm64 architectures, meaning it works fine natively on mac M1 systems and on Linux with both amd64 and arm64. It also works great on [Gitpod](topics/gitpod), where you don't have to install anything at all.

## System Requirements

=== "macOS"
    * DDEV runs natively on arm64 (Mac M1) systems as well as amd64 machines.
    * RAM: 8GB
    * Storage: 256GB
    * Colima or Docker Desktop is required.
    * Docker Desktop requires macOS Catalina (macOS 10.15) or higher.
    * Colima can even run on older systems.
    * DDEV should run anywhere Docker Desktop or Colima runs.

=== "Windows WSL2"
    * RAM: 8GB
    * Storage: 256GB
    * Systems that can run Docker Desktop on the Windows side or docker-ce inside WSL2 do fine.
    * The preferred distro is Ubuntu or an Ubuntu-derived distro, but people use lots of different distros.

=== "Traditional Windows"
    * Any recent edition of Windows Home, Pro, and several others.
    * RAM: 8GB
    * Storage: 256GB
    * Docker Desktop using the WSL2 back-end

=== "Linux"
    * Most distros and most versions work fine.
    * RAM: 8GB
    * Storage: 256GB

=== "Gitpod.io"

    With gitpod.io all you need is a browser and an internet connection. You don't have to install anything at all. You can use any kind of computer that has a browser, or tablet, or whatever you like.

!!!note "Using DDEV alongside other development environments"

    DDEV by default uses ports 80 and 443 on your system when projects are running. If you are using another local development environment (like Lando or Docksal or a native setup) you can either stop the other environment or configure DDEV to use different ports. See [troubleshooting](troubleshooting.md#unable-listen) for more detailed problem-solving. It's easiest just to stop the other environment when you want to use DDEV, and stop DDEV when you want to use the other environment.

## Support and User-Contributed Documentation

We love to hear from our users and help them be successful with DDEV. Support options include:

* Lots of built-in help: `ddev help` and `ddev help <command>`. You'll find examples and explanations.
* [DDEV Documentation](faq.md)
* [DDEV Stack Overflow](https://stackoverflow.com/questions/tagged/ddev) for support and frequently asked questions. We respond quite quickly here and the results provide quite a library of user-curated solutions.
* [DDEV issue queue](https://github.com/drud/ddev/issues) for bugs and feature requests
* Interactive community support on [Discord](https://discord.gg/hCZFfAMc5k) for everybody, plus sub-channels for CMS-specific questions and answers.
* [ddev-contrib](https://github.com/drud/ddev-contrib) repo provides a number of vetted user-contributed recipes for extending and using DDEV. Your contributions are welcome.
* [awesome-ddev](https://github.com/drud/awesome-ddev) repo has loads of external resources, blog posts, recipes, screencasts, and the like. Your contributions are welcome.
* [Twitter with tag #ddev](https://twitter.com/search?q=%23ddev&src=typd&f=live) will get to us, but it's not as good for interactive support, but we'll answer anywhere.
