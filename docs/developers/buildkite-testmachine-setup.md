<h1>Buildkite Test Agent Setup</h1>

We are using [Buildkite](https://buildkite.com/drud) for Windows and macOS testing. The build agents have to be set up before use.

Before beginning:
1. Obtain a DDEV_PANTHEON_API_TOKEN with privileges on a set-up pantheon account for testing.

## Windows Test Agent Setup:

1. Install [chocolatey](https://chocolatey.org/)
2. Install golang/make/git/docker-ce/nssm with `choco install -y golang make git docker-for-windows nssm`
3. If a laptop, set the "lid closing" setting in settings to do nothing.
4. Set the "Sleep after time" setting in settings to never.
5. Install the buildkite-agent. Use the latest release from [github.com/buildkite/agent](https://github.com/buildkite/agent/releases). 
6. Set up the agent to [run as a service](https://buildkite.com/docs/agent/v3/windows#running-as-a-service), preferably as the primary user of the machine, so it inherits environment variables and such.
7. Reboot the machine and do a test run.

### macOS Test Agent Setup

1. Install [homebrew](https://brew.sh/)
2. Install golang/git/docker with `brew install golang git buildkite-agent`
3. Install docker with `brew cask install docker`
4. If the xcode command line tools are not yet installed, install them with `xcode select --install`
5. Install nosleep `brew cask install nosleep`
6. Enable nosleep using its shortcut in the Mac status bar.
7. In nosleep Preferences, enable "Never sleep on AC Adapter", "Never sleep on Battery", and "Start nosleep utility on system startup".
8. Set up Mac to [automatically log in on boot](https://support.apple.com/en-us/HT201476).
9. Reboot the machine and do a test run. 
