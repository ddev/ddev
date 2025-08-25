---
search:
  boost: .2
---
# Buildkite Test Agent Setup

We are using [Buildkite](https://buildkite.com/ddev) for Windows and macOS testing. The build machines and `buildkite-agent` must be set up before use.

## Windows Test Agent Setup

1. Create the user “testbot” on the machine. Use the password for `ddevtestbot@gmail.com`, available in 1Password.
2. In admin PowerShell, `wsl --install`.
3. (WSL2/Docker Desktop and traditional Windows only): Install either [Docker Desktop for Windows](https://docs.docker.com/desktop/release-notes/) (from the release notes page, which is best maintained) or [Rancher Desktop](https://rancherdesktop.io/). With Rancher Desktop, turn off Kubernetes. (Rancher Desktop on Windows appears to be too unreliable to use at this point in its history.)
4. In admin PowerShell, `Set-ExecutionPolicy -Scope "CurrentUser" -ExecutionPolicy "Unrestricted"`.
5. In admin PowerShell, download and run [windows_buildkite_start.ps1](scripts/windows_buildkite_start.ps1) with `curl <url> -O windows_buildkite_start.ps1`.
6. Install items as needed; `git`, `jq`, `mysql-cli`, `golang`, `make` are only required for a traditional Windows test machine. `choco install -y git jq  mysql-cli golang make mkcert netcat zip`.
7. After restart, in **administrative** Git Bash window, `Rename-Computer <testbot-win10(home|pro)-<description>-1` and then `export BUILDKITE_AGENT_TOKEN=<token>`.
8. (Traditional Windows test runner only): Download and run [windows_buildkite_setup.sh](scripts/windows_buildkite_setup.sh).
9. If using Rancher Desktop, adjust the /c/buildkite-agent/buildkite-agent.cfg file to set `rancher-desktop=true` in the tags instead of `docker-desktop`. If using Docker Desktop, set `docker-desktop=true`.
10. (Traditional Windows test runner only): Download and run [windows_postinstall.sh](scripts/windows_postinstall.sh).
11. (Traditional Windows or Docker Desktop WSL2 Only) Launch Docker. It may require you to take further actions.
    * Check "Start Docker Desktop when you sign in" or the equivalent with Rancher Desktop.
    * Check "Add the *.docker.internal names to the host's /etc/hosts file"
    * Uncheck "SBOM Indexing"
    * Under "Resources" uncheck "Resource Saver"
12. After starting Docker Desktop or Rancher Desktop, set the correct Docker context in the Git Bash window with `docker context use desktop-linux` (Docker Desktop) or `docker context use default` (Rancher Desktop).
13. Log into Chrome with the user `ddevtestbot@gmail.com` and enable Chrome Remote Desktop.
14. (Traditional Windows test runner only): Enable `gd`, `fileinfo`, and `curl` extensions in `/c/tools/php*/php.ini`.
15. Set the “Sleep after time” setting in settings to never.
16. Install [winaero tweaker](https://winaero.com/request.php?1796) and “Enable user autologin checkbox”. Set up the machine to [automatically log in on boot](https://www.cnet.com/how-to/automatically-log-in-to-your-windows-10-pc/).  Then run netplwiz, provide the password for the main user, uncheck “require a password to log in”.
17. (Traditional Windows test runner only): Set the `buildkite-agent` service to run as the testbot user and use delayed start: Choose “Automatic, delayed start” and on the “Log On” tab in the services widget it must be set up to log in as the testbot user, so it inherits environment variables and home directory (and can access NFS, has testbot Git config, etc).
18. (Traditional Windows test runner only): `git config --global --add safe.directory '*'`.
19. (Traditional Windows test runner only): Manually run `testbot_maintenance.sh`, `curl -sL -O https://raw.githubusercontent.com/ddev/ddev/main/.buildkite/testbot_maintenance.sh && bash testbot_maintenance.sh`.
20. (Traditional Windows test runner only): Run `.buildkite/sanetestbot.sh` to check your work.
21. (Traditional Windows test runner only): Reboot the machine and do a test run. (On Windows, the machine name only takes effect on reboot.)
22. (Traditional Windows test runner only): Verify that `go`, `ddev`, `git-bash` are in the path.
23. In “Advanced Windows Update Settings” enable “Receive updates for other Microsoft products” to make sure you get WSL2 kernel upgrades. Make sure to run Windows Update to get the latest kernel.
24. Turn off the settings that cause the "windows experience" prompts after new upgrades:
![disable_windows_experience](../images/disable_windows_experience.png)
25. In PowerShell: `wsl.exe --update`. Watch for the elevation to complete, it may require elevation.

## Both Docker Desktop/WSL2 and Docker-ce/WSL2

1. The Ubuntu distro should be set up with the user `buildkite-agent` and the password used for `ddevtestbot@gmail.com`.
2. Stop Docker Desktop if it is running.
3. (Optionally if hostname is not same as appropriate hostname for WSL2) set the hostname in `/etc/wsl.conf` in the network section, for example:

    ```
    [network]
    hostname=tb-wsl-16-dockerce
    ```

4. Log into Chrome with the user `ddevtestbot@gmail.com` and enable Chrome Remote Desktop.
5. Windows Terminal should be installed. Set "Ubuntu" (or this distro) as the default and have it start on Windows startup. Enable "copy on select" in behaviors.
6. `nc.exe -L -p 9003` on Windows to trigger and allow Windows Defender.
7. Optionally edit the `~/.wslconfig` on Windows to add appropriate WSL2 memory allocation and `autoMemoryReclaim`

    ```
    memory=12GB
    [experimental]
    autoMemoryReclaim=dropcache
    ```

8. In the Ubuntu distro:
    1. `export BUILDKITE_AGENT_TOKEN=<token>` with the token from 1Password `BUILDKITE_AGENT_TOKEN`.
    2. `export BUILDKITE_DOCKER_TYPE=dockerforwindows` or `export BUILDKITE_DOCKER_TYPE=wsl2`
    3. Optionally `export NGROK_TOKEN=<token>` with the `NGROK_TOKEN` from 1Password ngrok.com `nopaid` account.
    4. Run the script [wsl2-test-runner-setup.sh](scripts/wsl2-test-runner-setup.sh) in the Ubuntu distro.
9. Restart the distro with `wsl.exe -t Ubuntu` and then restart it by opening the Ubuntu window.
10. If using Docker Desktop, start Docker Desktop.
11. In `~/workspace/ddev/.buildkite`, run `./testbot_maintenance.sh`.
12. In `~/workspace/ddev/.buildkite`, run `./sanetestbot.sh` to check your work.

## Icinga2 monitoring setup for WSL2 instances

1. Icinga Director web UI, configure the host on `newmonitor.thefays.us`, normally making a copy of an existing identical item.
2. Deploy the new host using Icinga Director.
3. `sudo icinga2 node wizard` to configure the agent, see [docs](https://icinga.com/docs/icinga-2/latest/doc/06-distributed-monitoring/#agentsatellite-setup-on-linux)
4. Restart `sudo systemctl restart icinga2`
5. On `newmonitor.thefays.us` edit `/usr/local/bin/check_buildkite_agents.sh` to include the hostname of the new instance.

## macOS Docker Desktop Test Agent Setup (Intel and Apple Silicon)

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
13. Install everything you’ll need with `brew install buildkite/buildkite/buildkite-agent bats-core composer ddev/ddev/ddev git golang jq mysql-client@8.0 mkcert netcat p7zip  && brew install --cask docker iterm2 ngrok && brew link --force mysql-client`.
14. Run `ngrok authtoken <token>` with token for free account from 1Password.
15. Run `mkcert -install`.
16. If Docker Desktop will be deployed, run Docker manually and go through its configuration routine.
17. If OrbStack will be deployed, install it from [orbstack.dev](https://orbstack.dev).
    * Install with Docker only.
    * Click "Sign in" in the lower left to sign in with OrbStack credentials (normal test runner gmail address; it will receive an email with a login code).
    * Configure it to automatically start and download updates, see ![OrbStack configuration](../images/orbstack_configuration.png).
18. If Rancher Desktop will be deployed, install it.
    * Turn off kubernetes.
    * Set virtualization to VZ
    * Set mount type to VirtioFS
19. Run iTerm. You may need to allow full disk access permissions.
20. Run `mkdir ~/workspace && cd ~/workspace && git clone https://github.com/ddev/ddev`.
21. Set up `nfsd` by running `bash ~/workspace/ddev/scripts/macos_ddev_nfs_setup.sh`.
22. `git config --global --add safe.directory '*'`.
23. Edit `/usr/local/etc/buildkite-agent/buildkite-agent.cfg` or `/opt/homebrew/etc/buildkite-agent/buildkite-agent.cfg` to add
    * the agent `token` (from [agents tab](https://buildkite.com/organizations/ddev/agents), "Reveal Agent Token").
    * the agent `name` (the name of the machine).
    * `tags`, like `"os=macos,architecture=arm64,osvariant=sonoma,dockertype=dockerformac,rancher-desktop=true,orbstack=true,docker-desktop=true"`
    * `build-path="~/tmp/buildkite-agent/builds"`
24. Run `brew services start buildkite-agent`.
25. Run `bash ~/workspace/ddev/.buildkite/testbot_maintenance.sh`.
26. Run `bash ~/workspace/ddev/.buildkite/sanetestbot.sh` to check your work.
27. The `testbot` user's SSH account is used for monitoring, so `ssh-keygen` and then add the public key `id_testbot` from 1Password to `~/.ssh/authorized_keys` and `chmod 600 ~/.ssh/authorized_keys`.
28. Add the new machine to Icinga by copying an existing Icinga service to the new one. This is done in **Icinga Director** → **Services** → **Single Services** → **Select a Service** → **Clone** → **Deploy**. The new service has to have `by-ssh-address` set to the name of the test runner, and that address needs to be added to `pi.ddev.site`'s `/etc/hosts` file.
29. If `zsh` is the shell configured, add `/etc/zshenv` so that `/usr/local/bin/docker` will be picked up:

    ```bash
    PATH=$PATH:/usr/local/bin:/opt/homebrew/bin
    ```

30. In macOS Settings visit "full disk access" and grant access to `buildkite-agent`, `docker`, `iterm`, `orbstack`. This may prevent startup modal dialogs that prevent `buildkite-agent` or `docker` from continuing properly.

## Additional Colima macOS setup

1. `brew install colima`
2. `colima start vz --cpu 4 --memory 6 --disk 60 --vm-type=vz --mount-type=virtiofs --dns=1.1.1.1`
3. `colima stop vz`

Then the Buildkite agent must be configured with tags `colima_vz=true`.

## Additional Lima macOS setup

1. `limactl create --name=lima-vz --vm-type=vz --mount-type=virtiofs --mount-writable --mount="~/:w" --memory=6 --cpus=4 --disk=60 template://docker`
2. `limactl start lima-vz`
3. `docker context use lima-lima-vz`

Then the Buildkite agent must be configured with tags `lima=true`.
