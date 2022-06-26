# Github Self-Hosted Agent Setup

We are using GitHub Self-Hosted Agents for Windows and macOS testing. The build machines and agents must be set up before use.

## Windows Agent Setup

1. Create the user "testbot" on the machine. The password should be the password of testbot@drud.com (available in 1password)
2. In admin PowerShell, `Set-ExecutionPolicy -Scope "CurrentUser" -ExecutionPolicy "RemoteSigned"`
3. In admin Powershell, download and run [windows_buildkite_start.ps1](scripts/windows_buildkite_start.ps1) (Use `curl <url> -O windows_buildkite_start.ps1`)
4. After restart, in administrative git-bash window, `Rename-Computer <testbot-win10(home|pro)-<description>-1`.
5. Now download and run [windows_github_agent_setup.sh](scripts/windows_github_agent_setup.sh)
6. Launch Docker. It may require you to take further actions.
7. Log into Chrome with the user testbot@drud.com and enable Chrome Remote Desktop.
8. Enable gd, fileinfo, and curl extensions in /c/tools/php*/php.ini
9. If a laptop, set the "lid closing" setting in settings to do nothing.
10. Set the "Sleep after time" setting in settings to never.
11. Install [winaero tweaker](https://winaero.com/request.php?1796) and "Enable user autologin checkbox". Set up the machine to [automatically log in on boot](https://www.cnet.com/how-to/automatically-log-in-to-your-windows-10-pc/).  Then run netplwiz, provide the password for the main user, uncheck the "require a password to log in".
12. Add the path `C:\Program Files\git\bin` to the very front of the *system* environment variables. Otherwise Windows will try to use its own bash.exe or will try to use PowerShell.
13. Install the github self-hosted runner software using the "Add New" instructions on <https://github.com/organizations/drud/settings/actions>. When it asks if you want it as a service... you do.
14. Run .buildkite/sanetestbot.sh to check your work.
15. Reboot the machine and do a test run. (On windows the machine name only takes effect on reboot.)

## macOS GitHub Self-Hosted Runner Setup

1. Create the user "testbot" on the machine. The password should be the password of testbot@drud.com.
2. Change the name of the machine to something in keeping with current style. Maybe `testbot-macstadium-macos-3`.
3. Install [Homebrew](https://brew.sh/) `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"`
4. Install golang/git/docker with `brew install homebrew/cask/iterm2 homebrew/cask/google-chrome homebrew/cask/docker drud/ddev/ddev homebrew/cask/nosleep buildkite/buildkite/buildkite-agent golang git mariadb jq p7zip bats-core composer netcat mkcert homebrew/cask/ngrok`
5. `mkcert -install`
6. Run Docker manually and go through its configuration routine.
7. Run iTerm. On Mojave and higher it may prompt for requiring full disk access permissions, follow through with that.
8. Set up nfsd by running `macos_ddev_nfs_setup.sh`
9. Enable nosleep using its shortcut in the Mac status bar.
10. In nosleep Preferences, enable "Never sleep on AC Adapter", "Never sleep on Battery", and "Start nosleep utility on system startup".
11. Set up Mac to [automatically log in on boot](https://support.apple.com/en-us/HT201476).
12. Install the github self-hosted runner software using the "Add New" instructions on <https://github.com/organizations/drud/settings/actions>
13. Set the runner to run as a service per [docs](https://docs.github.com/en/free-pro-team@latest/actions/hosting-your-own-runners/configuring-the-self-hosted-runner-application-as-a-service) with `./svc.sh install && ./svc.sh start`
14. Try checking out ddev and running .buildkite/sanetestbot.sh to check your work.
15. Log into Chrome with the user testbot@drud.com and enable Chrome Remote Desktop.
16. Set the timezone properly (US MT)
