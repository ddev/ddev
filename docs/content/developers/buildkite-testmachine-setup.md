# Buildkite Test Agent Setup

We are using [Buildkite](https://buildkite.com/ddev) for Windows and macOS testing. The build machines and `buildkite-agent` must be set up before use.

## Windows Test Agent Setup

1. Create the user “testbot” on the machine. Use the password for `ddevtestbot@gmail.com`, available in LastPass.
2. In admin PowerShell, `wsl --install`.
3. In admin PowerShell, `Set-ExecutionPolicy -Scope "CurrentUser" -ExecutionPolicy "RemoteSigned"`.
4. In admin PowerShell, download and run [windows_buildkite_start.ps1](scripts/windows_buildkite_start.ps1) with `curl <url> -O windows_buildkite_start.ps1`.
5. After restart, in **administrative** Git Bash window, `Rename-Computer <testbot-win10(home|pro)-<description>-1` and then `export BUILDKITE_AGENT_TOKEN=<token>`.
6. Now download and run [`windows_buildkite-testmachine_setup.sh`](scripts/windows_buildkite_setup.sh).
7. Download and run [windows_postinstall.sh](scripts/windows_postinstall.sh).
8. Launch Docker. It may require you to take further actions.
9. Log into Chrome with the user `ddevtestbot@gmail.com` and enable Chrome Remote Desktop.
10. Enable gd, fileinfo, and curl extensions in `/c/tools/php*/php.ini`.
11. If a laptop, set the “lid closing” setting to do nothing.
12. Set the “Sleep after time” setting in settings to never.
13. Install [winaero tweaker](https://winaero.com/request.php?1796) and “Enable user autologin checkbox”. Set up the machine to [automatically log in on boot](https://www.cnet.com/how-to/automatically-log-in-to-your-windows-10-pc/).  Then run netplwiz, provide the password for the main user, uncheck “require a password to log in”.
14. The `buildkite/hooks/environment.bat` file must be updated to contain the Docker pull credentials:

    ```bash
    @echo off
    set DOCKERHUB_PULL_USERNAME=druddockerpullaccount
    set DOCKERHUB_PULL_PASSWORD=
    ```

15. Set the `buildkite-agent` service to run as the testbot user and use delayed start: Choose “Automatic, delayed start” and on the “Log On” tab in the services widget it must be set up to log in as the testbot user, so it inherits environment variables and home directory (and can access NFS, has testbot Git config, etc).
16. `git config --global --add safe.directory '*'`.
17. Manually run `testbot_maintenance.sh`, `curl -sL -O https://raw.githubusercontent.com/ddev/ddev/master/.buildkite/testbot_maintenance.sh && bash testbot_maintenance.sh`.
18. Run `.buildkite/sanetestbot.sh` to check your work.
19. Reboot the machine and do a test run. (On Windows, the machine name only takes effect on reboot.)
20. Verify that `go`, `ddev`, `git-bash` are in the path.
21. In “Advanced Windows Update Settings” enable “Receive updates for other Microsoft products” to make sure you get WSL2 kernel upgrades. Make sure to run Windows Update to get the latest kernel.

## Additional Windows Setup for WSL2+Docker Desktop Testing

1. Do not set up `buildkite-agent` on the Windows side, or disable it.
2. Edit Ubuntu's `/etc/wsl.conf` to contain:

    ```
    [boot]
    systemd=true
    ```

3. Update WSL2 to WSL2 Preview from Microsoft Store and `wsl --shutdown` and then restart.
4. `wsl --update`
5. Open WSL2 and check out [ddev/ddev](https://github.com/ddev/ddev).
6. As normal user, run `.github/workflows/linux-setup.sh`.
7. `export PATH=/home/linuxbrew/.linuxbrew/bin:$PATH
    echo "export PATH=/home/linuxbrew/.linuxbrew/bin:$PATH" >>~/.bashrc`

8. As root user, add sudo capability with `echo "ALL ALL=NOPASSWD: ALL" >/etc/sudoers.d/all && chmod 440 /etc/sudoers.d/all`.
9. Manually run `testbot_maintenance.sh`, `curl -sL -O https://raw.githubusercontent.com/ddev/ddev/master/.buildkite/testbot_maintenance.sh && bash testbot_maintenance.sh`.
10. `git config --global --add safe.directory '*'`
11. Install basics in WSL2:

    ```bash
    curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | sudo tee /etc/apt/keyrings/ddev.gpg > /dev/null
    echo "deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ * *" | sudo tee /etc/apt/sources.list.d/ddev.list >/dev/null
    # Update package information and install DDEV
    sudo apt update && sudo apt install -y ddev

    sudo mkdir -p /usr/sharekeyrings && curl -fsSL https://keys.openpgp.org/vks/v1/by-fingerprint/32A37959C2FA5C3C99EFBC32A79206696452D198 | sudo gpg --dearmor -o /usr/share/keyrings/buildkite-agent-archive-keyring.gpg
    echo "deb [signed-by=/usr/share/keyrings/buildkite-agent-archive-keyring.gpg] https://apt.buildkite.com/buildkite-agent stable main" | sudo tee /etc/apt/sources.list.d/buildkite-agent.list
    sudo apt update && sudo apt install -y build-essential buildkite-agent ca-certificates curl ddev gnupg lsb-release make mariadb-client
    sudo snap install ngrok
    ```

12. [Configure `buildkite-agent` in WSL2](https://buildkite.com/docs/agent/v3/ubuntu). It needs the same changes as macOS, but tags `tags="os=wsl2,architecture=amd64,dockertype=dockerforwindows"` and build-path should be in `~/tmp/buildkite-agent`.

13. The buildkite/hooks/environment file must be updated to contain the Docker pull credentials:

    ```bash
        #!/bin/bash
        export DOCKERHUB_PULL_USERNAME=druddockerpullaccount
        export DOCKERHUB_PULL_PASSWORD=xxx
        set -e
    ```

14. Verify that `buildkite-agent` is running.
15. In Task Scheduler, create a task that runs on User Logon and runs `C:\Windows\System32\wsl.exe` with arguments `-d Ubuntu`.
16. Add `buildkite-agent` to the `docker` and `testbot` groups in `/etc/group`
17. `echo "capath=/etc/ssl/certs/" >>~/.curlrc` And then do the same as `buildkite-agent` user
18. `sudo chmod -R ug+w /home/linuxbrew`
19. `nc.exe -l -p 9003` on Windows to trigger and allow Windows Defender.
20. Run `ngrok config add-authtoken <token>` with token for free account.
21. Copy ngrok config into `buildkite-agent` account, `sudo cp -r ~/.ngrok2 ~buildkite-agent/ && sudo chown -R buildkite-agent:buildkite--agent ~buildkite-agent/ngrok2`
22. Add `/home/linuxbrew/.linuxbrew/bin` to `PATH` in `/etc/environment`.
23. Copy ngrok config into `buildkite-agent` account, `sudo cp -r ~/.ngrok2 ~buildkite-agent/ && sudo chown -R buildkite-agent:buildkite--agent ~buildkite-agent/ngrok2`
24. Add `buildkite-agent` to `sudo` group in `/etc/groups`
25. Give `buildkite-agent` a password with `sudo passwd buildkite-agent`
26. As `buildkite-agent` user `mkcert -install`

## Additional Windows Setup for WSL2+Docker-Inside Testing

1. Uninstall Docker Desktop.
2. Remove all of the entries (especially `host.docker.internal`) that Docker Desktop has added in `C:\Windows\system32\drivers\etc\hosts`.
3. Install Docker and basics in WSL2:

    ```bash
    sudo mkdir -p /etc/apt/keyrings
    sudo mkdir -p /etc/apt/keyrings && sudo rm -f /etc/apt/keyrings/docker.gpg && curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
    sudo apt update && sudo apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
    sudo usermod -aG docker $USER
    ```

4. Configure buildkite agent in /etc/buildkite-agent:
    * tags="os=wsl2,architecture=amd64,dockertype=wsl2"
    * token="xxx"
    * Create `/etc/buildkite-agent/hooks/environment` and set to executable with contents:

    ```
        #!/bin/bash
        export DOCKERHUB_PULL_USERNAME=druddockerpullaccount
        export DOCKERHUB_PULL_PASSWORD=xxx
        set -e
    ```

5. Run `.buildkite/sanetestbot.sh`

## macOS Test Agent Setup (Intel and Apple Silicon)

1. Create the user “testbot” on the machine. Use the password for `ddevtestbot@gmail.com`, available in LastPass.
2. Change the name of the machine to something in keeping with current style. Maybe `testbot-macstadium-macos-3`.
3. Install [Homebrew](https://brew.sh/) `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"`
4. Install everything you’ll need with `brew install buildkite/buildkite/buildkite-agent  bats-core colima composer ddev/ddev/ddev git golang jq mariadb mkcert netcat p7zip  && brew install --cask docker iterm2 google-chrome nosleep ngrok`.
5. Run `ngrok config add-authtoken <token>` with token for free account.
6. Run `mkcert -install`.
7. Run Docker manually and go through its configuration routine.
8. Run iTerm. On Mojave and higher you may need to allow full disk access permissions.
9. Set up `nfsd` by running `macos_ddev_nfs_setup.sh`.
10. `git config --global --add safe.directory '*'`
11. Edit `/usr/local/etc/buildkite-agent/buildkite-agent.cfg` or `/opt/homebrew/etc/buildkite-agent/buildkite-agent.cfg` to add
    * the agent token
    * tags, like `"os=macos,architecture=arm64,osvariant=monterrey,dockertype=dockerformac"`
    * `build-path="~/tmp/buildkite-agent/builds"`
12. The buildkite/hooks/environment file must be updated to contain the Docker pull credentials:

    ```bash
        #!/bin/bash
        export DOCKERHUB_PULL_USERNAME=druddockerpullaccount
        export DOCKERHUB_PULL_PASSWORD=xxx
        set -e
    ```

13. Run `brew services start buildkite-agent`.
14. Manually run `testbot_maintenance.sh`, `curl -sL -O https://raw.githubusercontent.com/ddev/ddev/master/.buildkite/testbot_maintenance.sh && bash testbot_maintenance.sh`.
15. Enable nosleep using its shortcut in the Mac status bar.
16. In nosleep Preferences, enable “Never sleep on AC Adapter”, “Never sleep on Battery”, and “Start nosleep utility on system startup”.
17. `sudo chown testbot /usr/local/bin`
18. Set up Mac to [automatically log in on boot](https://support.apple.com/en-us/HT201476).
19. Try checking out [ddev/ddev](https://github.com/ddev/ddev) and running `.buildkite/sanetestbot.sh` to check your work.
20. Log into Chrome with the user `ddevtestbot@gmail.com` and enable Chrome Remote Desktop.
21. Set the timezone (US MT).
22. Start the agent with `brew services start buildkite-agent`.
