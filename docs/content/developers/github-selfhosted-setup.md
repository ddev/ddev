---
hide:
  - toc
---

# GitHub Self-Hosted Agent Setup

We are using GitHub Self-Hosted Agents for Windows and macOS testing. The build machines and agents must be set up before use.

## Windows Agent Setup

1. Create the user “testbot” on the machine. Use the password for `ddevtestbot@gmail.com`, available in 1Password.
2. In admin PowerShell, `Set-ExecutionPolicy -Scope "CurrentUser" -ExecutionPolicy "RemoteSigned"`.
3. In admin Powershell, download and run [windows_buildkite_start.ps1](scripts/windows_buildkite_start.ps1) with `curl <url> -O windows_buildkite_start.ps1`.
4. After restart, in administrative Git Bash window, `Rename-Computer <testbot-win10(home|pro)-<description>-1`.
5. Now download and run [windows_github_agent_setup.sh](scripts/windows_github_agent_setup.sh).
6. Launch Docker. It may require you to take further actions.
7. Log into Chrome with the user `ddevtestbot` and enable Chrome Remote Desktop.
8. Enable `gd`, `fileinfo`, and `curl` extensions in `/c/tools/php*/php.ini`.
9. If a laptop, set the “lid closing” setting in settings to do nothing.
10. Set the “Sleep after time” setting in settings to never.
11. Install [`winaero tweaker`](https://winaero.com/request.php?1796) and “Enable user autologin checkbox”. Set up the machine to [automatically log in on boot](https://www.cnet.com/how-to/automatically-log-in-to-your-windows-10-pc/).  Then run `netplwiz`, provide the password for the main user, uncheck the “require a password to log in”.
12. Add the path `C:\Program Files\git\bin` to the very front of the *system* environment variables. Otherwise Windows will try to use its own `bash.exe` or PowerShell.
13. Install the GitHub self-hosted runner software using the “Add New” instructions on <https://github.com/organizations/ddev/settings/actions>. When it asks if you want it as a service: yes, you do.
14. Run `.buildkite/sanetestbot.sh` to check your work.
15. Reboot the machine and do a test run. (On Windows, the machine name only takes effect on reboot.)
