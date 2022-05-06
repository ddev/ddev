# DDEV-Local Release Management

## Pushing necessary docker images

Before a release (or some PRs) a docker image needs to be pushed. This can be done via the GitHub web interface, or manually.

1. The drud/ddev-php-base image must be updated as necessary with a new tag before pushing `ddev-webserver`. You can do this at <https://github.com/drud/ddev/actions/workflows/push-tagged-image.yml>:

* Choose "Push tagged image" in the "Workflows" section on the left side.
* Click "Run workflow" in the blue section at the top of "workflow runs".
* Choose the branch to run from (normally "master").
* The image should be "ddev-php-base"

The build takes something over an hour.

If you need to push this from a forked PR, you'll have to do this from your fork (for example, <https://github.com/drud/rfay/actions/workflows/push-tagged-image.yml>), and you'll have to specify the branch on the fork. This requires that the DOCKERHUB_TOKEN and DOCKERHUB_USERNAME secrets be set on the forked PR, for example <https://github.com/rfay/ddev/settings/secrets/actions>.

*
*

*

## GitHub Actions Environment Preparation

The following "Repository secret" environment variables must be added to <https://github.com/drud/ddev/settings/secrets/actions>

* AUR_SSH_PRIVATE_KEY: The private ssh key for the ddev-releaser user. This must be processed into a single line, for example, `perl -p -e 's/\n/<SPLIT>/' ~/.ssh/id_rsa_ddev_releaser| pbcopy`.

* CHOCOLATEY_API_KEY: API key for chocolatey.

* DDEV_GITHUB_TOKEN: The GitHub token that gives access to create releases and push to the homebrew repositories.

* DDEV_MACOS_APP_PASSWORD: The password used for notarization, see [signing_tools](https://github.com/drud/signing_tools)

* DDEV_MACOS_SIGNING_PASSWORD: The password the access the signing key on macOS, see [signing_tools](https://github.com/drud/signing_tools)

* DDEV_WINDOWS_SIGNING_PASSWORD: The windows signing password.

* HOMEBREW_EDGE_REPOSITORY: The name of the GitHub repo used for the edge channel on homebrew, drud/homebrew-ddev-ege

* HOMEBREW_STABLE_REPOSITORY: The name of the GitHub repo used for the stable channel on homebrew/ drud/homebrew-ddev

* SegmentKey: The key that enabled the Segment reporting

## Creating a release (almost everything should be automated)

1. Create tagged images. `git fetch upstream && git checkout upstream/master && cd containers` and `for item in *; do pushd $item; make push VERSION=<release_version> DOCKER_ARGS=--no-cache ; popd; done`
2. Update the default container versions in `pkg/version/version.go` and create a pull request
3. Create a release for the new version using the GitHub UI. It should be "prerelease" if it's an edge release.
4. Use the "Auto-generate release notes" option to get the commit list, then edit to add all the other necessary info.
5. Verify that homebrew (linux and macOS) and Chocolatey and AUR are working correctly with the right versions

## Manually updating homebrew formulas

Homebrew formulas are normally updated just fine by the release process, so nothing needs to be done.

If you have to temporarily update the homebrew formulas, you can do that with a commit to <https://github.com/drud/homebrew-ddev> and <https://github.com/drud/homebrew-ddev-edge>. The bottles and checksums for macOS (high sierra) and x86_64_linux are built and pushed to the release page automatically by the release build process (see [bump_homebrew.sh](https://github.com/drud/ddev/blob/master/.ci-scripts/bump_homebrew.sh). Test `brew upgrade ddev` both on macOS and Linux and make sure ddev is the right version and behaves well.

## Manually updating Chocolatey

Normallly the release process does OK with pushing to Chocolatey, but at times a failure can happen and it's not worth doing the whole release process again.

* Open up gitpod, <https://gitpod.io/#https://github.com/drud/ddev> and

```bash
cd /workspace/ddev
sudo apt-get update && sudo apt-get install -y nsis
sudo .ci-scripts/nsis_setup.sh /usr/share/nsis
make chocolatey
cd .gotmp/bin/windows_amd64/chocolatey
````

* edit the checksum in tools/chocolateyinstall.ps1 to match the released checksum of the ddev-windows-installer (not the choco package. For v1.19.2 this was <https://github.com/drud/ddev/releases/download/v1.19.2/ddev_windows_installer.v1.19.2.exe.sha256.txt>)

```bash
rm .gotmp/bin/windows_amd64/chocolatey/*.nupkg
export CHOCOLATEY_API_KEY=key33333
docker run --rm -v "/$PWD:/tmp/chocolatey" -w "//tmp/chocolatey" linuturk/mono-choco pack ddev.nuspec;
docker run --rm -v $PWD:/tmp/chocolatey -w /tmp/chocolatey linuturk/mono-choco push -s [https://push.chocolatey.org/](https://push.chocolatey.org/) --api-key "${CHOCOLATEY_API_KEY}"

```

## Manually updating AUR repository

The AUR repository is normally updated just fine by the release process, so nothing needs to be done.

However, you can manually publish the release to the ddev [AUR repository](https://aur.archlinux.org/packages/ddev-bin/). The README.md in the AUR git repo (`ssh://aur@aur.archlinux.org/ddev-bin.git` or `https://aur.archlinux.org/ddev-bin.git`) has instructions on how to update, including how to do it with a Docker container, so it doesn't have to be done on an ArchLinux or Manjaro VM.

## Manually Signing the Windows installer

(This is done by the release process, but the manual process documented here.)

Note that this is done automatically by the release build, on a dedicated Windows test runner (GitHub Actions runner) named testbot-asus-win10pro. If it is to be done manually it has to be done on that machine or the fob has to be installed on another machine.

**After a reboot of this machine, sometimes an automated reboot, the password for the security fob has to be re-entered and Windows signing will fail until it is. I do this by opening up testbot-asus-win10pro using Chrome Remote Desktop (or manually physically opening it) and opening git-bash and `cd ~/tmp && signtool sign gsudo.exe`. There just happens to be a gsudo.exe there but it doesn't matter what you sign, the idea is to pop up the gui where you enter the password (which is in lastpass).**

### Basic instructions

1. Install the suggested [Windows SDK 10](https://developer.microsoft.com/en-us/windows/downloads/windows-10-sdk)
2. Install [Visual Studio Community 2015](https://msdn.microsoft.com/en-us/library/mt613162.aspx)
3. Run the [Developer Command Prompt](https://docs.microsoft.com/en-us/visualstudio/ide/reference/command-prompt-powershell?view=vs-2019)
4. The keyfob and Safenet Authentication Client must be installed. The best documentation for the Safenet software is at <https://support.globalsign.com/ssl/ssl-certificates-installation/safenet-drivers>.
5. After `make windows_install` the `ddev-windows-installer.exe` will be in `.ddev/bin/windows_amd64/ddev_windows_installer.exe` and you can sign it with `signtool sign ddev-windows-installer.exe`.

I do not believe that we should use this keyfob high-security approach to signing on the next go-around with the certs. It is way too difficult to manage, and the Safenet software is atrocious.
