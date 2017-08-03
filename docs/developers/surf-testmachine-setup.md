<h1>Surf Test Machine Setup</h1>

We are using [surf](https://github.com/surf-build/surf) for Windows and OSX testing. This is a very simple test setup that uses on-premise machines that have to be configured correctly to run.

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
7. Install surf from https://github.com/surf-build/surf/releases
8. Configure the windows machine to [automatically log in on boot](https://www.howtogeek.com/112919/how-to-make-your-windows-8-computer-logon-automatically/). 
9. In an *administrative cmd shell*, 
```
set GITHUB_TOKEN=012345...
set DDEV_PANTHEON_API_TOKEN=012356...
surf-install -e DDEV_PANTHEON_API_TOKEN -n surf-windows-ddev -c "surf-run -r https://github.com/drud/ddev -j 1 -- surf-build -n  surf-windows"

```
where `surf-windows-ddev` is the name of the job on the windows machine (it's free-form) and `surf-windows` is the identifier used on github when the build changes statuses; it's also free-form. **Note that current versions of surf-install will fail claiming that schtasks failed, but it seems to work fine anyway. See [issue](https://github.com/surf-build/surf/issues/64).**

### OSX Test Machine Setup

1. Install [homebrew](https://brew.sh/)
2. Install golang/git/docker with `brew install golang git docker`
3. Install nosleep `brew cask install nosleep`
4. Run nosleep and configure with "Never sleep on AC Adapter", "Never sleep on Battery", "Start nosleep utility on system startup" 
5. Set up Mac to [automatically log in on boot](https://support.apple.com/en-us/HT201476). 
6. Install surf from https://github.com/surf-build/surf/releases
7. In a shell, 
```
export GITHUB_TOKEN=012345...
export DDEV_PANTHEON_API_TOKEN=012356...
surf-install -e DDEV_PANTHEON_API_TOKEN -n surf-darwin-ddev -c "surf-run -r https://github.com/drud/ddev -j 1 -- surf-build -n  surf-darwin"

```
where `surf-darwin-ddev` is the name of the job on the mac machine (it's free-form) and `surf-darwin` is the identifier used on github when the build changes statuses; it's also free-form.

8. Reboot the machine and do a test run. Alternately test with `surf-build -r https://github.com/drud/ddev -s <some_sha>`