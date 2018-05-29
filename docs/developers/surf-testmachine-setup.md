#Surf Test Machine Setup

**(Obsolete, can be removed in the future)**

Until 2018-05 we used [surf](https://github.com/surf-build/surf) for Windows and macOS testing. This is a very simple test setup that uses on-premise machines that have to be configured correctly to run.

Before beginning:
1. Obtain a GITHUB_TOKEN with repo and gist privileges (and for a user who has write/push privs on the repo). Currently this GITHUB_TOKEN is associated with drud-test-machine-account and the token is shared in lastpass.
2. Obtain a DDEV_PANTHEON_API_TOKEN with privileges on a set-up pantheon account for testing.

## Windows Test Machine Setup:

1. Install [chocolatey](https://chocolatey.org/)
2. Install golang/make/git with `choco install -y golang make git`
3. Install current stable [docker-for-windows](https://docs.docker.com/docker-for-windows/install/#download-docker-for-windows)
4. In docker config, share the "C" drive  (this must also be done after any docker reset)
5. Set the "lid closing" setting in settings to do nothing.
6. Set the "Sleep after time" setting in settings to never.
7. Install Surf:
    - The quickest way to install Surf is with NPM:
    `npm install -g surf-build`
    - Alternatively, download a release from https://github.com/surf-build/surf/releases.
8. Configure the windows machine to [automatically log in on boot](https://www.howtogeek.com/112919/how-to-make-your-windows-8-computer-logon-automatically/).
9. In an *administrative cmd shell*,
```
set GITHUB_TOKEN=012345...
set DDEV_PANTHEON_API_TOKEN=012356...
surf-install -e DDEV_PANTHEON_API_TOKEN -n surf-windows-ddev -c "surf-run -r https://github.com/drud/ddev -j 1 -- surf-build -n  surf-windows"

```
where `surf-windows-ddev` is the name of the job on the windows machine (it's free-form) and `surf-windows` is the identifier used on github when the build changes statuses; it's also free-form. **Note that current versions of surf-install will fail claiming that schtasks failed, but it seems to work fine anyway. See [issue](https://github.com/surf-build/surf/issues/64).**

### macOS Test Machine Setup

1. Install [homebrew](https://brew.sh/)
2. Install golang/git/docker with `brew install golang git docker`
3. If the xcode command line tools are not yet installed, install them with `xcode select --install`
4. Install nosleep `brew cask install nosleep`
5. Enable nosleep using its shortcut in the Mac status bar.
6. In nosleep Preferences, enable "Never sleep on AC Adapter", "Never sleep on Battery", and "Start nosleep utility on system startup".
7. Set up Mac to [automatically log in on boot](https://support.apple.com/en-us/HT201476).
8. Install Surf:
    - The quickest way to install Surf is with NPM:
    `npm install -g surf-build`
    - OR you can download a release from https://github.com/surf-build/surf/releases.
9. In a shell,
```
export GITHUB_TOKEN=012345...
export DDEV_PANTHEON_API_TOKEN=012356...
surf-install -e DDEV_PANTHEON_API_TOKEN -n surf-darwin-ddev -c "surf-run -r https://github.com/drud/ddev -j 1 -- surf-build -n  surf-darwin"
```
where `surf-darwin-ddev` is the name of the job on the mac machine (it's free-form) and `surf-darwin` is the identifier used on github when the build changes statuses; it's also free-form.

9. Reboot the machine and do a test run. Alternatively, test with `surf-build -r https://github.com/drud/ddev -s <some_sha>`
