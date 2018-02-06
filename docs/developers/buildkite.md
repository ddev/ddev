# Buildkite

We are using [Buildkite](https://buildkite.com) for Windows and macOS testing.

## Logging into Buildkite

Buildkite is SSO-enabled. Go to https://buildkite.com/login, enter your Drud
email address, leave the password field blank, and then click Login.

## MacStadium

We use MacStadium for running the Mac and Windows build agents because they support
a variety of OSes (not just macOS). To gain access to the 
[MacStadium portal](https://portal.macstadium.com), check `drud secret`, or ask
in Slack.

## macOS machine setup:

1. Set hostname: `sudo scutil --set HostName mac-highsierra01.build.drud.com`
2. Add DNS record that points to the appropriate IP address (match the hostname).
3. Generate an ssh key: `ssh-keygen -t rsa -b 4096`
4. Add the public key to the github user `drud-test-machine-account`
5. Install [homebrew](https://brew.sh/): `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"`
6. Install xcode command line tools: `xcode-select --install`
7. Install all OS updates.
8. Install build dependencies: `brew install git golang`
9. Get an agent token from the Buildkite UI.
10. Install buildkite agent: `brew install --token="AGENT-TOKEN" buildkite/buildkite/buildkite-agent`
11. Start buildkite agent on login: `brew services start buildkite/buildkite/buildkite-agent`
12. Check that the build agent shows up in the Buildkite UI.
13. Install Docker for Mac: `brew cask install docker`
14. Run Docker.app and go through the interactive setup process: `open /Applications/Docker.app`
15. Reboot the machine.

## macOS machine maintenance:

1. Install all OS updates.
2. Update homebrew: `brew update`
3. Upgrade casks: `brew cask upgrade`
4. Upgrade other brew formulae: `brew upgrade`


