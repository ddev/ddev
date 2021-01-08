# Buildkite Test Agent Setup

We are using [Buildkite](https://buildkite.com/drud) for Windows and macOS testing. The build machines and buildkite-agent must be set up before use.

## Windows Test Agent Setup

1. Create the user "testbot" on the machine. The password should be the password of testbot@drud.com (available in 1password)
2. In admin PowerShell, `Set-ExecutionPolicy -Scope "CurrentUser" -ExecutionPolicy "RemoteSigned"`
3. In admin PowerShell, download and run [windows_buildkite_start.ps1](scripts/windows_buildkite_start.ps1) (Use `curl <url> -O windows_buildkite_start.ps1`)
4. After restart, in administrative git-bash window, `Rename-Computer <testbot-win10(home|pro)-<description>-1` and then `export BUILDKITE_AGENT_TOKEN=<token>`
5. Now download and run [windows_buildkite-testmachine_setup.sh](scripts/windows_buildkite_setup.sh)
6. Launch Docker. It may require you to take further actions.
7. Log into Chrome with the user testbot@drud.com and enable Chrome Remote Desktop.
8. Enable gd, fileinfo, and curl extensions in /c/tools/php*/php.ini
9. If a laptop, set the "lid closing" setting in settings to do nothing.
10. Set the "Sleep after time" setting in settings to never.
11. Install [winaero tweaker](https://winaero.com/request.php?1796) and "Enable user autologin checkbox". Set up the machine to [automatically log in on boot](https://www.cnet.com/how-to/automatically-log-in-to-your-windows-10-pc/).  Then run netplwiz, provide the password for the main user, uncheck the "require a password to log in".
12. Set the buildkite-agent service to run as the testbot user and use delayed start: Choose "Automatic, delayed start" and on the "Log On" tab in the services widget it must be set up to log in as the testbot user, so it inherits environment variables and home directory (and can access NFS, has testbot git config, etc).
13. Run .buildkite/sanetestbot.sh to check your work.
14. Reboot the machine and do a test run. (On windows the machine name only takes effect on reboot.)

## macOS Test Agent Setup

1. Create the user "testbot" on the machine. The password should be the password of testbot@drud.com.
2. Change the name of the machine to something in keeping with current style. Maybe `testbot-macstadium-macos-3`.
3. Install [Homebrew](https://brew.sh/) `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"`
4. Install golang/git/docker with `brew cask install iterm2 google-chrome  docker nosleep && brew tap buildkite/buildkite && brew tap drud/ddev && brew install golang git buildkite-agent mariadb jq p7zip bats-core composer ddev netcat mkcert && brew cask install ngrok`
5. `mkcert -install`
6. Run Docker manually and go through its configuration routine.
7. Run iTerm. On Mojave and higher it may prompt for requiring full disk access permissions, follow through with that.
8. Set up nfsd by running `macos_ddev_nfs_setup.sh`
9. Edit the buildkite-agent.cfg in /usr/local/etc/buildkite-agent.cfg to add
    * the agent token
    * Tags, like `"os=macos,osvariant=catalina,dockertype=dockerformac"`
    * `build-path="~/tmp/buildkite-agent/builds"`
10. `brew services start buildkite-agent`
11. Enable nosleep using its shortcut in the Mac status bar.
12. In nosleep Preferences, enable "Never sleep on AC Adapter", "Never sleep on Battery", and "Start nosleep utility on system startup".
13. Set up Mac to [automatically log in on boot](https://support.apple.com/en-us/HT201476).
14. Try checking out ddev and running .buildkite/sanetestbot.sh to check your work.
15. Log into Chrome with the user testbot@drud.com and enable Chrome Remote Desktop.
16. Set the timezone properly (US MT)
17. Start the agent with `brew services start buildkite-agent`
