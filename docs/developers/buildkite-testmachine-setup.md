# Buildkite Test Agent Setup

We are using [Buildkite](https://buildkite.com/drud) for Windows and macOS testing. The build machines and buildkite-agent must be set up before use.

## Windows Test Agent Setup

1. Create the user "testbot" on the machine. The password should be the password of testbot@drud.com (available in 1password)
2. Install [chocolatey](https://chocolatey.org/docs/installation) with an administrative cmd window `@"%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe" -NoProfile -InputFormat None -ExecutionPolicy Bypass -Command "iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))" && SET "PATH=%PATH%;%ALLUSERSPROFILE%\chocolatey\bin"`
3. Install golang/mysql-cli/make/git/docker-ce/nssm with `choco install -y git mysql-cli golang make docker-desktop nssm GoogleChrome zip jq composer cmder netcat ddev mkcert` (If docker-toolbox, use that instead; you may have to download the release separately to get correct version.)
4. `mkcert -install`
5. Enable gd, fileinfo, and curl extensions in /c/tools/php73/php.ini
6. Install bats: `git clone git://github.com/bats-core/bats-core; cd bats-core; git checkout v1.1.0; ./install.sh`
7. If a laptop, set the "lid closing" setting in settings to do nothing.
8. Set the "Sleep after time" setting in settings to never.
9. Install the buildkite-agent. Use the latest release from [github.com/buildkite/agent](https://github.com/buildkite/agent/releases). It should go in /c/buildkite-agent, with the buildkite-agent.exe in /c/buildkite-agent/bin and the config in /c/buildkite-agent.
10. Update the buildkite-agent.cfg
    * *token*
    * *tags*: Tags will probably be like `"os=windows,osvariant=windows10pro,dockertype=dockerforwindows"` or `"os=windows,osvariant=windows10pro,dockertype=toolbox"`
    * build-path: "C:\Users\testbot\tmp\buildkite" on Windows. (This is to get it to build in the home directory; by default docker toolbox won't mount anything outside the home directory.)

11. Set up the agent to [run as a service](https://buildkite.com/docs/agent/v3/windows#running-as-a-service):
    * on the "Log On" tab in the services widget it must be set up to log in as the primary user of the machine, so it inherits environment variables and home directory.
12. Set up the machine to [automatically log in on boot](https://www.cnet.com/how-to/automatically-log-in-to-your-windows-10-pc/).  Run netplwiz, provide the password for the main user, uncheck the "require a password to log in".
13. On Docker Toolbox systems, add a link to "Docker Quickstart Terminal" in C:\ProgramData\Microsoft\Windows\Start Menu\Programs\StartUp (see [link](http://www.thewindowsclub.com/make-programs-run-on-startup-windows)).
14. On Docker Desktop for Windows systems, launch Docker. It will offer to reconfigure Hyper-V and do a restart.
15. On Docker Desktop for Windows, configured the C: and other drives as shared to docker.
16. On Docker Toolbox systems, make sure that nested virtualization is enabled however you need to enable it.
17. Edit /c/ProgramData/git/config "core" section to `autocrlf: false` and `eol: lf` verify that `git config --list` shows only autocrlf: false.
18. Run `winpty docker run -it -p 80 busybox ls` to trigger the Windows Defender warning, and "allow access".
19. Try running .buildkite/sanetestbot.sh to check your work.
20. Install ddev using the ddev_windows_installer.exe from <https://github.com/drud/ddev/releases>
21. Setup up winnfsd by running `windows_ddev_nfs_setup.sh`.
22. Change the name of the machine to something in keeping with current style. Maybe `testbot-dell-toolbox-3`.
23. Reboot the machine and do a test run. (On windows the machine name only takes effect on reboot.)
24. Set the timezone properly (US MT)
25. Log into Chrome with the user testbot@drud.com and enable Chrome Remote Desktop.

### Docker Toolbox Extra Instructions

1. `docker-machine rm default`
2. `docker-machine create -d virtualbox --virtualbox-cpu-count=2 --virtualbox-memory=4096 --virtualbox-disk-size=50000 default`
3. Disable "Windows Defender Firewall", as it always blocks our Xdebug test.

## macOS Test Agent Setup

1. Create the user "testbot" on the machine. The password should be the password of testbot@drud.com.
2. Change the name of the machine to something in keeping with current style. Maybe `testbot-macstadium-macos-3`.
3. Install [homebrew](https://brew.sh/) `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"`
4. Install golang/git/docker with `brew cask install iterm2 google-chrome  docker nosleep && brew tap buildkite/buildkite && brew tap drud/ddev && brew install golang git buildkite-agent mariadb jq p7zip bats-core composer ddev netcat mkcert && brew cask install ngrok`
5. `mkcert -install`
6. Run docker manually and go through its configuration routine.
7. Run `iterm`. On Mojave it may prompt for requiring full disk access permissions, follow through with that.
8. Set up nfsd by running `macos_ddev_nfs_setup.sh`
9. Add the path `/private/var` or on Catalina `/System/Volumes/Data/private/var` to `/etc/exports` and `sudo nfsd restart`.
10. Edit the buildkite-agent.cfg in /usr/local/etc/buildkite-agent.cfg to add
    * the agent token
    * Tags, like `"os=macos,osvariant=highsierra,dockertype=dockerformac"`
    * `build-path="~/tmp/buildkite-agent/builds"`
11. `brew services start buildkite-agent`
12. Enable nosleep using its shortcut in the Mac status bar.
13. In nosleep Preferences, enable "Never sleep on AC Adapter", "Never sleep on Battery", and "Start nosleep utility on system startup".
14. Set up Mac to [automatically log in on boot](https://support.apple.com/en-us/HT201476).
15. Try checking out ddev and running .buildkite/sanetestbot.sh to check your work.
16. Log into Chrome with the user testbot@drud.com and enable Chrome Remote Desktop.
17. Set the timezone properly (US MT)
18. Start the agent with `brew services start buildkite-agent`
19. Reboot the machine and do a test run.
