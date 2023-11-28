---
search:
  boost: .2
---
# Buildkite Test Agent Setup

We are using [Buildkite](https://buildkite.com/ddev) for Windows and macOS testing. The build machines and `buildkite-agent` must be set up before use.

## Windows Test Agent Setup

1. Create the user “testbot” on the machine. Use the password for `ddevtestbot@gmail.com`, available in 1Password.
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
    set DOCKERHUB_PULL_PASSWORD=xxx_readonly_token
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
        export DOCKERHUB_PULL_PASSWORD=xxx_readonly_token
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
        export DOCKERHUB_PULL_PASSWORD=xxx_readonly_token
        set -e
    ```

5. Run `.buildkite/sanetestbot.sh`

## macOS Test Agent Setup (Intel and Apple Silicon)

1. Create the user “testbot” on the machine. Use the password for `ddevtestbot@gmail.com`, available in 1Password.
2. Change the name of the machine to something in keeping with current style, perhaps `testbot-macos-arm64-8`. This is done in **Settings** → **General** → **About** → **Name** and in **Sharing** → **Computer Name** and in **Sharing** → **Local Hostname**.
3. Download and install Chrome and log the browser into the account used for test runners. It will pick up the Chrome Remote Desktop setup as a result. Configure Chrome Remote Desktop to serve. When this is done, the machine will be available for remote access and most other tasks can be done using Chrome Remote Desktop.
4. The machine should be on the correct network and have a static IP handed out by DHCP. IP addresses are listed in /etc/hosts on `pi.ddev.site`, so this one should be added.
5. Power should be set up as in ![macos power settings](../images/macos_power_settings.png).
6. Auto login should be set up as in ![macos users and groups](../images/macos_users_and_groups.png), see [automatically log in on boot](https://support.apple.com/en-us/HT201476).
7. Remote login should be enabled as in ![macos remote login](../images/macos_remote_login.png).
8. Automatic updates should be set to mostly security only as in ![macos automatic_updatees](../images/macos_automatic_updates.png).
9. Set the time zone to US MT (nearest city: Denver, Colorado).
10. `sudo mkdir -p /usr/local/bin && sudo chown -R testbot /usr/local/bin`
11. Install [Homebrew](https://brew.sh/) `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`
12. After installing Homebrew follow the instructions it gives you at the end to add brew to your PATH.
13. Install everything you’ll need with `brew install buildkite/buildkite/buildkite-agent bats-core composer ddev/ddev/ddev git golang jq mariadb mkcert netcat p7zip  && brew install --cask docker iterm2 ngrok`.
14. Run `ngrok config add-authtoken <token>` with token for free account from 1Password.
15. Run `mkcert -install`.
16. If Docker Desktop will be deployed, run Docker manually and go through its configuration routine.
17. If OrbStack will be deployed, install it from [orbstack.dev](https://orbstack.dev).
    * Install with Docker only.
    * Click "Sign in" in the lower left to sign in with OrbStack credentials (normal test runner gmail address; it will receive an email with a login code).
    * Configure it to automatically start and download updates, see ![OrbStack configuration](../images/orbstack_configuration.png).
18. If Rancher Desktop will be deployed, install it.
    * Turn off kubernetes.
19. Run iTerm. You may need to allow full disk access permissions.
20. Run `mkdir ~/workspace && cd ~/workspace && git clone https://github.com/ddev/ddev`.
21. Set up `nfsd` by running `bash ~/workspace/ddev/scripts/macos_ddev_nfs_setup.sh`.
22. `git config --global --add safe.directory '*'`.
23. Edit `/usr/local/etc/buildkite-agent/buildkite-agent.cfg` or `/opt/homebrew/etc/buildkite-agent/buildkite-agent.cfg` to add
    * the agent `token` (from [agents tab](https://buildkite.com/organizations/ddev/agents), "Reveal Agent Token").
    * the agent `name` (the name of the machine).
    * `tags`, like `"os=macos,architecture=arm64,osvariant=sonoma,dockertype=dockerformac,rancher-desktop=true,orbstack=true,docker-desktop=true"`
    * `build-path="~/tmp/buildkite-agent/builds"`
24. The `buildkite-agent/hooks/environment` file must be created and set executable to contain the Docker pull credentials (found in `druddockerpullaccount` in 1Password):

    ```bash
    #!/bin/bash
    export DOCKERHUB_PULL_USERNAME=druddockerpullaccount
    export DOCKERHUB_PULL_PASSWORD=xxx_readonly_token
    set -e
    ```

25. Run `brew services start buildkite-agent`.
26. Run `bash ~/workspace/ddev/.buildkite/testbot_maintenance.sh`.
27. Run `bash ~/workspace/ddev/.buildkite/sanetestbot.sh` to check your work.
28. The `testbot` user's ssh account is used for monitoring, so `ssh-keygen` and then add the public key `id_testbot` from 1Password to `~/.ssh/authorized_keys` and `chmod 600 ~/.ssh/authorized_keys`.
29. Add the new machine to Icinga by copying an existing Icinga service to the new one. This is done in **Icinga Director** → **Services** → **Single Services** → **Select a Service** → **Clone** → **Deploy**. The new service has to have `by-ssh-address` set to the name of the test runner, and that address needs to be added to `pi.ddev.site`'s `/etc/hosts` file.
30. If `zsh` is the shell configured, add `/etc/zshenv` so that `/usr/local/bin/docker` will be picked up:

    ```bash
    PATH=$PATH:/usr/local/bin:/opt/homebrew/bin
    ```
