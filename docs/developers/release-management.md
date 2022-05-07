# DDEV-Local Release Management and Docker Images

## GitHub Actions Required Secrets

<!-- markdown-link-check-disable-next-line -->
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

## Creating a release (almost everything is now automated)

### Prerelease tasks

* Make sure the version-history.md file is up to date.
* Make sure the docker images are all tagged and pushed.
* Make sure the pkg/version/version.go is all set to point to the new images (and tests have been run)

### Actual release creation

1. Create a release for the new version using the GitHub UI. It should be "prerelease" if it's an edge release.
2. Use the "Auto-generate release notes" option to get the commit list, then edit to add all the other necessary info.
3. Verify that homebrew (linux and macOS) and Chocolatey and AUR are working correctly with the right versions

## Pushing docker images with the GitHub Actions workflow

The easiest way to push docker images is to use the GitHub Actions workflow, especially if the code for the image is already in the ddev repo.

<!-- markdown-link-check-disable-next-line -->
You can push an image at <https://github.com/drud/ddev/actions/workflows/push-tagged-image.yml>

<!-- markdown-link-check-disable-next-line -->
If you need to push from a forked PR, you'll have to do this from your fork (for example, <https://github.com/drud/rfay/actions/workflows/push-tagged-image.yml>), and you'll have to specify the branch on the fork. This requires that the DOCKERHUB_TOKEN and DOCKERHUB_USERNAME secrets be set on the forked PR, for example `https://github.com/rfay/ddev/settings/secrets/actions`.

* Visit `https://github.com/drud/ddev/actions/workflows/push-tagged-image.yml`
* Click the "Push tagged image" workflow on the left side of the page.
* Click the "Run workflow" button in the blue section above the workflow runs.
* Choose the branch to build from (usually master)
* Enter the image (ddev-webserver, ddev-dbserver, ddev-php-base, etc)

* Enter the tag that will be used in pkg/version/version.go.

## Pushing docker images manually

It's more error-prone, but images can be pushed from the command-line.

1. `docker login` with a user that has privileges to push.
2. `docker buildx create --name ddev-builder-multi --use` or if it already exists, `docker buildx use ddev-builder-multi`
3. `cd containers/<image>`
4. Before pushing ddev-webserver, make sure you've pushed a version of ddev-php-base and updated ddev-webserver's Dockerfile to use that as a base.
5. `make push VERSION=<release_version> DOCKER_ARGS=--no-cache` for most of the images. For ddev-dbserver it's `make PUSH=true VERSION=<release_version> DOCKER_ARGS=--no-cache`. There is a [push-all.sh](https://github.com/drud/ddev/blob/master/containers/push-all.sh) script to update all. But it takes forever.

## Maintaining ddev-dbserver mysql:5.7 and mysql:8.0 ARM64 images

Sadly, there are no arm64 Docker images for mysql:5.7 and mysql:8.0, so we have a whole process to maintain our own for ddev.

We maintain [drud/mysql-arm64-images](https://github.com/drud/mysql-arm64-images) and [drud/xtrabackup-build](https://github.com/drud/xtrabackup-build) for this reason.

* drud/mysql:5.7 usees Ubuntu 18.04 as the base image, and Ubuntu 18.04 arm64 has mysql-server 5.7 in it, so we can install.
* drud/mysql:8.0 uses Ubuntu 20.04 as the base image, and Ubuntu 20.04 arm64 has mysql-server 8.0 in it, so we can install it from packages.
* Unfortunately, the `ddev snapshot` feature depends on xtrabackup 8.0 being installed for mysql:8.0. And there are no arm64 packages or binaries provided by percona for xtrabackup. So we build it from source with [drud/xtrabackup-build](https://github.com/drud/xtrabackup-build). BUT... xtrabackup's development cycle lags behind mysql:8.0's development cycle, so you can't build a usable drud/mysql:8.0 image until there's an xtrabackup version released. Also unfortunately, when Ubuntu bumps mysql-server-8.0 to a new version, there's no way to use the old one. So the only time that you can maintain drud/mysql:8.0 is when Ubuntu 20.04 has the same version that's released for percona-xtrabackup. (In the case at this writeup, I was finally able to build percona-xtrabackup 8.0.28... and the same day Ubuntu bumped its packages to 8.0.29, meaning that it was unusable.)
* To build percona-xtrabackup, follow the instructions on [drud/xtrabackup-build](https://github.com/drud/xtrabackup-build). You just create a release with the release of Percona xtrabackup, for example `8.0.29-21`. When that succeeds, then there is an upstream xtrabackup to be used in the drud/mysql:8.0 build.
* To build drud/mysql (both 5.7 and 8.0) arm64 images, follow the instructions on [drud/mysql-arm64-images](https://github.com/drud/mysql-arm64-images) After the various files are updated, you can just push a new release and the proper images will be pushed.
* After building a new set of drud/mysql images, you'll need to push `drud/ddev-dbserver` with new tags. Make sure to update the [drud/ddev-dbserver Makefile](https://github.com/drud/ddev/blob/master/containers/ddev-dbserver/Makefile) to set the explicit version of the upstream mysql:8.0 (for example, 8.0.29, if you've succeed in getting 8.0.29 for percona-xtrabackup and mysql:8.0).

## Actual release docker image updates

I don't actually build every image for every point release. If there have been no changes to ddev-router or ddev-ssh-agent, for example, I only usually push those (and update pkg/version/version.go) on major releases.

But here are the steps for building:

1. The drud/ddev-php-base image must be updated as necessary with a new tag before pushing `ddev-webserver`. You can do this using the [process above](#pushing-docker-images-with-the-github-actions-workflow)
2. The drud/ddev-webserver Dockerfile must `FROM drud/ddev-php-base:<tag>` before building/pushing `ddev-webserver`. But then it can be pushed using either the Github Actions or the manual technique.

3. If you're bumping ddev-dbserver 8.0 minor release, follow the upstream instructions [here](#maintaining-ddev-dbserver-mysql57-and-mysql80-arm64-images).
4. Push images using the [process above](#pushing-docker-images-with-the-github-actions-workflow).
5. Update pkg/version/version.go with the correct versions for the new images, and run a full test run.

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
