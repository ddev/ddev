# Buildkite Test Agent Setup

We are using [Buildkite](https://buildkite.com/drud) for Windows and macOS testing. The build machines and buildkite-agent must be set up before use.

## Windows Test Agent Setup

1. Create the user "testbot" on the machine. The password should be the password of testbot@drud.com (available in 1password)
2. Install [chocolatey](https://chocolatey.org/docs/installation) with an administrative PowerShell window `Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))`
3. Install git with `choco install -y git`
4. Rename the computer to `testbot-win10(home|pro)-<descriptor>-<number>`. for example, testbot-win10home-drud1-1, with `Rename-computer <name`
5. `Set-Timezone -Id "Mountain Standard Time"`
6. Install WSL2 and restart in the same administrative PS windows:  `dism.exe /online /enable-feature /featurename:Microsoft-Windows-Subsystem-Linux /all /norestart` and `dism.exe /online /enable-feature /featurename:VirtualMachinePlatform /all`
7. After restart, in administrative git-bash window, `export BUILDKITE_AGENT_TOKEN=<token>`
8. Now run [windows_buildkite-testmachine_setup.sh](scripts/windows_buildkite_setup.sh)
9. Enable gd, fileinfo, and curl extensions in /c/tools/php*/php.ini
10. If a laptop, set the "lid closing" setting in settings to do nothing.
11. Set the "Sleep after time" setting in settings to never.
12. Install [winaero tweaker](https://winaero.com/request.php?1796) and "Enable user autologin checkbox". Set up the machine to [automatically log in on boot](https://www.cnet.com/how-to/automatically-log-in-to-your-windows-10-pc/).  Then run netplwiz, provide the password for the main user, uncheck the "require a password to log in".
13. Launch Docker. It may require you to take further actions.
14. Run .buildkite/sanetestbot.sh to check your work.
15. Reboot the machine and do a test run. (On windows the machine name only takes effect on reboot.)
16. Log into Chrome with the user testbot@drud.com and enable Chrome Remote Desktop.

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
