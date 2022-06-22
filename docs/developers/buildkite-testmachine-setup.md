# Buildkite Test Agent Setup

We are using [Buildkite](https://buildkite.com/drud) for Windows and macOS testing. The build machines and buildkite-agent must be set up before use.

## Windows Test Agent Setup

1. Create the user "testbot" on the machine. The password should be the password of ddevtestbot@gmail.com (available in lastpass)
2. In admin PowerShell, `wsl --install`
3. In admin PowerShell, `Set-ExecutionPolicy -Scope "CurrentUser" -ExecutionPolicy "RemoteSigned"`
4. In admin PowerShell, download and run [windows_buildkite_start.ps1](scripts/windows_buildkite_start.ps1) (Use `curl <url> -O windows_buildkite_start.ps1`)
5. After restart, in **administrative** git-bash window, `Rename-Computer <testbot-win10(home|pro)-<description>-1` and then `export BUILDKITE_AGENT_TOKEN=<token>`
6. Now download and run [windows_buildkite-testmachine_setup.sh](scripts/windows_buildkite_setup.sh)
7. Download and run [windows_postinstall.sh](scripts/windows_postinstall.sh)
8. Launch Docker. It may require you to take further actions.
9. Log into Chrome with the user ddevtestbot@gmail.com and enable Chrome Remote Desktop.
10. Enable gd, fileinfo, and curl extensions in /c/tools/php*/php.ini
11. If a laptop, set the "lid closing" setting in settings to do nothing.
12. Set the "Sleep after time" setting in settings to never.
13. Install [winaero tweaker](https://winaero.com/request.php?1796) and "Enable user autologin checkbox". Set up the machine to [automatically log in on boot](https://www.cnet.com/how-to/automatically-log-in-to-your-windows-10-pc/).  Then run netplwiz, provide the password for the main user, uncheck the "require a password to log in".
14. The buildkite/hooks/environment.bat file must be updated to contain the docker pull credentials:
```bash
@echo off
set DOCKERHUB_PULL_USERNAME=druddockerpullaccount
set DOCKERHUB_PULL_PASSWORD=
```
15. Set the buildkite-agent service to run as the testbot user and use delayed start: Choose "Automatic, delayed start" and on the "Log On" tab in the services widget it must be set up to log in as the testbot user, so it inherits environment variables and home directory (and can access NFS, has testbot git config, etc).
16. `git config --global --add safe.directory '*'`
17. Manually run `testbot_maintenance.sh`, `curl -sL -O https://raw.githubusercontent.com/drud/ddev/master/.buildkite/testbot_maintenance.sh && bash testbot_maintenance.sh`
18. Run .buildkite/sanetestbot.sh to check your work.
19. Reboot the machine and do a test run. (On windows the machine name only takes effect on reboot.
20. Verify that go, ddev, git-bash are in the path
21. In "Advanced Windows Update Settings" enable "Receive updates for other Microsoft products" to make sure you get WSL2 kernel upgrades. Make sure to run Windows update to get latest kernel.

## Additional Windows setup for WSL2 testing

1. Do not set up buildkite-agent on the Windows side, or disable it.
2. Open WSL2 and check out ddev
3. [Install buildkite-agent in WSL2](https://buildkite.com/docs/agent/v3/ubuntu) and configure it. It needs the same changes as macOS, but tags `tags="os=wsl2,architecture=amd64,dockertype=dockerforwindows"` and build-path should be in `~/tmp/buildkite-agent`
4. As root user, run `.github/workflows/linux-setup.sh`
5. As root user, add sudo capability with `echo "ALL ALL=NOPASSWD: ALL" >/etc/sudoers.d/all && chmod 440 /etc/sudoers.d/all`
6. Test from PowerShell that `wsl -d Ubuntu buildkite-agent start` succeeds and starts listening.
Set up Windows to automatically start WSL2 buildkite-agent: Use task scheduler to create a simple task that runs `C:\Windows\System32\wsl.exe -d Ubuntu buildkite-agent start` at login.
7. Install homebrew, `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`
8. Manually run `testbot_maintenance.sh`, `curl -sL -O https://raw.githubusercontent.com/drud/ddev/master/.buildkite/testbot_maintenance.sh && bash testbot_maintenance.sh`
9. 16. `git config --global --add safe.directory '*'`
10. The buildkite/hooks/environment file must be updated to contain the docker pull credentials:
```bash
   #!/bin/bash
   export DOCKERHUB_PULL_USERNAME=druddockerpullaccount
   export DOCKERHUB_PULL_PASSWORD=xxx
   set -e
```

## macOS Test Agent Setup (works for M1 as well)

1. Create the user "testbot" on the machine. The password should be the password of ddevtestbot@gmail.com.
2. Change the name of the machine to something in keeping with current style. Maybe `testbot-macstadium-macos-3`.
3. Install [Homebrew](https://brew.sh/) `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"`
4. Install golang/git/docker with `brew install buildkite/buildkite/buildkite-agent  bats-core colima composer drud/ddev/ddev git golang jq mariadb mkcert netcat p7zip  && brew install --cask docker iterm2 google-chrome nosleep ngrok`
5. Run `ngrok config add-authtoken <token>` with token for free account.
6. `mkcert -install`
7. Run Docker manually and go through its configuration routine.
8. Run iTerm. On Mojave and higher it may prompt for requiring full disk access permissions, follow through with that.
9. Set up nfsd by running `macos_ddev_nfs_setup.sh`
10. `git config --global --add safe.directory '*'`
11. Edit the buildkite-agent.cfg in `/usr/local/etc/buildkite-agent/buildkite-agent.cfg` or `/opt/homebrew/etc/buildkite-agent/buildkite-agent.cfg` to add
    * the agent token
    * Tags, like `"os=macos,architecture=arm64,osvariant=monterrey,dockertype=dockerformac"`
    * `build-path="~/tmp/buildkite-agent/builds"`
12. The buildkite/hooks/environment file must be updated to contain the docker pull credentials:
```bash
   #!/bin/bash
   export DOCKERHUB_PULL_USERNAME=druddockerpullaccount
   export DOCKERHUB_PULL_PASSWORD=xxx
   set -e
```
11. `brew services start buildkite-agent`
12. Manually run `testbot_maintenance.sh`, `curl -sL -O https://raw.githubusercontent.com/drud/ddev/master/.buildkite/testbot_maintenance.sh && bash testbot_maintenance.sh`
13. Enable nosleep using its shortcut in the Mac status bar.
14. In nosleep Preferences, enable "Never sleep on AC Adapter", "Never sleep on Battery", and "Start nosleep utility on system startup".
15. `sudo chown testbot /usr/local/bin`
16. Set up Mac to [automatically log in on boot](https://support.apple.com/en-us/HT201476).
17. Try checking out ddev and running .buildkite/sanetestbot.sh to check your work.
18. Log into Chrome with the user ddevtestbot@gmail.com and enable Chrome Remote Desktop.
19. Set the timezone properly (US MT)
20. Start the agent with `brew services start buildkite-agent`
