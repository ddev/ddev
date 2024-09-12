---
search:
  boost: .5
---
# Release Management & Docker Images

## Release process and tools

* [GoReleaser Pro](https://goreleaser.com/pro/) is used to do the actual releasing using [.goreleaser.yml](https://github.com/ddev/ddev/blob/master/.goreleaser.yml). GoReleaser Pro is a licensed product that requires separate installation and a license key, which is in the GitHub Workflow configuration and is available in 1Password to DDEV maintainers who need it.
* The [Master Build/Release GitHub Action](https://github.com/ddev/ddev/blob/master/.github/workflows/master-build.yml) does the actual running of the GoReleaser actions and provides the needed secrets.

## GitHub Actions Required Secrets

### How to add new people to these accounts

* AUR is Arch Linux User Repository. `ddev-bin` is at `https://aur.archlinux.org/packages/ddev-bin`. The current maintainer of this is @rfay, who can add co-maintainers.
* The [chocolatey](https://community.chocolatey.org/packages/ddev/) package. Additional maintainers can be added at (login required) `https://community.chocolatey.org/packages/ddev/ManagePackageOwners`; they could then create tokens to push it.
* GitHub requires write access to this repository, either via permissions on the repository or on the org.
* Apple signing and notarization requires access to the DDEV Foundation group on `https://developer.apple.com`. It's easy enough to add additional people.
* Windows signing is an awkward process that requires a dongle. When the current signing certificate expires we definitely want the simpler approach.
* Discord
* Docker

### Environment variables required

These are normally configured in the repository environment variables.

* `AUR_EDGE_GIT_URL`: The Git URL for AUR edge (normally `ddev-edge-bin`), for example `ssh://aur@aur.archlinux.org/ddev-edge-bin.git`.
* `AUR_STABLE_GIT_URL`: The Git URL for AUR stable (normally `ddev-bin`), for example `ssh://aur@aur.archlinux.org/ddev-bin.git`.
* `DOCKERHUB_USERNAME`: Username for pushing to `hub.docker.com` or updating image descriptions.
* `FURY_ACCOUNT`: [Gemfury](https://gemfury.com) account that receives package pushes.
* `HOMEBREW_EDGE_REPOSITORY`: Like `ddev/homebrew-ddev-edge` but might be another repo like be `ddev-test/homebrew-ddev-edge`.
* `HOMEBREW_STABLE_REPOSITORY`: Like `ddev/homebrew-ddev` but might be another repo like `ddev-test/homebrew-ddev`.

### Actual secrets required

<!-- markdown-link-check-disable-next-line -->
The following “Repository secret” environment variables must be configured in 1Password:

* `AMPLITUDE_API_KEY`: Key that enables Amplitude reporting. Environment variable for Make is `AmplitudeAPIKey`.
* `AMPLITUDE_API_KEY_DEV`: Key that enables Amplitude reporting for development versions e.g. a PR build. Environment variable for Make is `AmplitudeAPIKey`.
* `AUR_SSH_PRIVATE_KEY`: Private SSH key for the `ddev-releaser` user. This must be processed into a single line, for example, `perl -p -e 's/\n/<SPLIT>/' ~/.ssh/id_rsa_ddev_releaser| pbcopy`.
* `CHOCOLATEY_API_KEY`: API key for Chocolatey.
* `DDEV_GITHUB_TOKEN`: GitHub personal token (`repo` scope, classic PAT) that gives access to create releases and push to the Homebrew repositories.
* `DDEV_MACOS_APP_PASSWORD`: Password used for notarization, see [signing_tools](https://github.com/ddev/signing_tools).
* `DDEV_MACOS_SIGNING_PASSWORD`: Password for the macOS signing key, see [signing_tools](https://github.com/ddev/signing_tools).
* `DDEV_WINDOWS_SIGNING_PASSWORD`: Windows signing password.
* `DOCKERHUB_TOKEN`: Token for pushing to `hub.docker.com`. or updating image descriptions.
* `FURY_TOKEN`: Push token assigned to the above Gemfury account.
* `GORELEASER_KEY`: License key for GoReleaser Pro.

## Creating a Release

!!!tip "This is completely automated now, so nothing needs to be done unless something goes wrong."

### Prerelease Tasks

* Create and execute a test plan.
* Make sure [`version-history.md`](https://github.com/ddev/ddev/blob/master/version-history.md) is up to date.
* Push the new version of `ddev/ddev-php-base`.
* Update `ddev/ddev-webserver` to use the new version of `ddev/ddev-php-base` and push it with the proper tag.
* Make sure the Docker images are all tagged and pushed.
* Make sure [`pkg/versionconstants/versionconstants.go`](https://github.com/ddev/ddev/blob/master/pkg/versionconstants/versionconstants.go) is all set to point to the new images and tests have been run.
* If the [`devcontainer-feature.json`](https://github.com/ddev/ddev/blob/master/.github/devcontainers/src/install-ddev/devcontainer-feature.json) (for GitHub Codespaces) needs to be updated, use the [`devcontainer` CLI](https://github.com/devcontainers/cli) and a `GITHUB_TOKEN` that has power to manage packages (`write:packages` scope, classic PAT), change the version in the [`devcontainer-feature.json`](https://github.com/ddev/ddev/blob/master/.github/devcontainers/src/install-ddev/devcontainer-feature.json) and run:

    ```bash
    cd .github/devcontainers/src
    export GITHUB_TOKEN=<personal-access-token-with-power-to-manage-packages>
    devcontainer features publish -n ddev/ddev .
    ```

### Actual Release Creation

1. Create a [release](https://github.com/ddev/ddev/releases) for the new version using the GitHub UI. It should be “prerelease” if it’s an edge release.
2. Make sure you're about to create the right release tag.
3. Use the “Auto-generate release notes” option to get the commit list, then edit to add all the other necessary info.

### Post-Release Tasks

1. After the release has been created, the new gitpod image must be pushed.
    1. `cd .gitpod/images && DOCKER_TAG="<YYMMDD>" ./push.sh`
    2. PR to update `.gitpod.yml` with the new image.
    3. PR to update [ddev-gitpod-launcher](https://github.com/ddev/ddev-gitpod-launcher) with the new image.

## Pushing Docker Images with the GitHub Actions Workflow

The easiest way to push Docker images is to use the GitHub Actions workflow, especially if the code for the image is already in the [ddev/ddev](https://github.com/ddev/ddev) repository.

### Actual release creation

1. Create a [release](https://github.com/ddev/ddev/releases) for the new version using the GitHub UI. It should be “prerelease” if it’s an edge release.
2. Use the “Auto-generate release notes” option to get the commit list, then edit to add all the other necessary info.
3. Verify that Homebrew (Linux and macOS) and Chocolatey and AUR are working correctly with the right versions.

<!-- markdown-link-check-disable-next-line -->
You can push all images besides `ddev-dbserver` at <https://github.com/ddev/ddev/actions/workflows/push-tagged-image.yml>
<!-- markdown-link-check-disable-next-line -->
You can push `ddev-dbserver` images at <https://github.com/ddev/ddev/actions/workflows/push-tagged-dbimage.yml>

<!-- markdown-link-check-disable-next-line -->
If you need to push from a forked PR, you’ll have to do this from your fork (for example, `https://github.com/rfay/ddev/actions/workflows/push-tagged-image.yml`), and you’ll have to specify the branch on the fork. This requires setting the `DOCKERHUB_TOKEN` and `DOCKERHUB_USERNAME` secrets on the forked PR, for example `https://github.com/rfay/ddev/settings/secrets/actions`. You can do the same with `ddev-dbserver` at `https://github.com/rfay/ddev/actions/workflows/push-tagged-dbimage.yml` for example.

* Visit `https://github.com/ddev/ddev/actions/workflows/push-tagged-image.yml`.
* Click the “Push tagged image” workflow on the left side of the page.
* Click the “Run workflow” button in the blue section above the workflow runs.
* Choose the branch to build from (usually `master`).
* Enter the image (`ddev-webserver`, `ddev-php-base`, etc.).
* Enter the tag that will be used in `pkg/version/version.go`.

## Pushing Docker Images Manually

While it’s more error-prone, images can be pushed from the command line:

1. `docker login` with a user that has push privileges.
2. `docker buildx create --name ddev-builder-multi --use` or if it already exists, `docker buildx use ddev-builder-multi`.
3. `cd containers/<image>`.
4. Before pushing `ddev-webserver`, make sure you’ve pushed a version of `ddev-php-base` and updated `ddev-webserver`’s Dockerfile to use that as a base.
5. `make push VERSION=<release_version> DOCKER_ARGS=--no-cache` for most of the images. For `ddev-dbserver` it’s `make PUSH=true VERSION=<release_version> DOCKER_ARGS=--no-cache`. There’s a [push-all.sh](https://github.com/ddev/ddev/blob/master/containers/push-all.sh) script to update all of them, but it takes forever.
6. `ddev-dbserver` images can be pushed with `make PUSH=true VERSION=<release_version> DOCKER_ARGS=--no-cache` from the `containers/ddev-dbserver` directory.

## Maintaining `ddev-dbserver` MySQL 5.7 and 8.0 ARM64 Images

Sadly, there are no ARM64 Docker images for MySQL 5.7 and 8.0, so we have our own process to maintain [ddev/mysql-arm64-images](https://github.com/ddev/mysql-arm64-images) and [ddev/xtrabackup-build](https://github.com/ddev/xtrabackup-build) images for DDEV.

* `ddev/mysql:5.7` uses Ubuntu 18.04 as the base image, and Ubuntu 18.04 ARM64 has `mysql-server` 5.7 in it, so we can install.
* `ddev/mysql:8.0` uses Ubuntu 20.04 as the base image, and Ubuntu 20.04 ARM64 has `mysql-server` 8.0 in it, so we can install it from packages.
* Unfortunately, the [`ddev snapshot`](../users/usage/commands.md#snapshot) command depends on `xtrabackup` 8.0 being installed for `mysql:8.0`. There are no ARM64 packages or binaries provided by Percona for `xtrabackup`, so we build it from source with [ddev/xtrabackup-build](https://github.com/ddev/xtrabackup-build). **There’s a catch, however:** `xtrabackup`’s development cycle lags behind `mysql:8.0`’s development cycle, so you can’t build a usable `ddev/mysql:8.0` image until there’s an `xtrabackup` version released. Further, when Ubuntu bumps `mysql-server-8.0` to a new version, there’s no way to use the old one. So the only time that you can maintain `ddev/mysql:8.0` is when Ubuntu 20.04 has the same version that’s released for `percona-xtrabackup`. (In the case at this writeup, I was finally able to build `percona-xtrabackup` 8.0.28, and the same day Ubuntu bumped its packages to 8.0.29, meaning that it was unusable.)
* To build percona-xtrabackup, follow the instructions on [ddev/xtrabackup-build](https://github.com/ddev/xtrabackup-build). Create a release with the release of Percona xtrabackup, for example `8.0.29-21`. When that succeeds, then there is an upstream xtrabackup to be used in the ddev/mysql:8.0 build.
* To build `ddev/mysql` (both 5.7 and 8.0) ARM64 images, follow the instructions on [ddev/mysql-arm64-images](https://github.com/ddev/mysql-arm64-images). After the various files are updated, you can push a new release and the proper images will be pushed.
* After building a new set of `ddev/mysql` images, you’ll need to push `ddev/ddev-dbserver` with new tags. Make sure to update the [ddev/`ddev-dbserver` Makefile](https://github.com/ddev/ddev/blob/master/containers/ddev-dbserver/Makefile) to set the explicit version of the upstream `mysql:8.0` (for example, 8.0.29, if you’ve succeeded in getting 8.0.29 for `percona-xtrabackup` and `mysql:8.0`).

## Actual Release Docker Image Updates

We may not build every image for every point release. If there have been no changes to `ddev-traefik-router` or `ddev-ssh-agent`, for example, we may not push those and update `pkg/version/version.go` on major releases.

But here are the steps for building:

1. The `ddev/ddev-php-base` image must be updated as necessary with a new tag before pushing `ddev-webserver`. You can do this using the [process above](#pushing-docker-images-with-the-github-actions-workflow).
2. The `ddev/ddev-webserver` Dockerfile must `FROM ddev/ddev-php-base:<tag>` before building/pushing `ddev-webserver`. But then it can be pushed using either the GitHub Actions or the manual technique.
3. If you’re bumping `ddev-dbserver` 8.0 minor release, follow the upstream [Maintaining ddev-dbserver MySQL 5.7 & 8.0 ARM64 Images](#maintaining-ddev-dbserver-mysql-57-and-80-arm64-images) instructions.
4. Update `pkg/version/version.go` with the correct versions for the new images, and run all the tests.

## Manually Updating Homebrew Formulas

Homebrew formulas normally update with the release process, so nothing needs to be done.

If you have to temporarily update the Homebrew formulas, you can do that with a commit to <https://github.com/ddev/homebrew-ddev> and <https://github.com/ddev/homebrew-ddev-edge>. The bottles and checksums for macOS (High Sierra) and x86_64_linux are built and pushed to the release page automatically by the release build process (see [bump_homebrew.sh](https://github.com/ddev/ddev/blob/master/.ci-scripts/bump_homebrew.sh)). Test `brew upgrade ddev` both on macOS and Linux and make sure DDEV is the right version and behaves well.

## Manually Updating Chocolatey

Normally the release process does okay with pushing to Chocolatey, but at times a failure can happen and it’s not worth doing the whole release process again.

Note that if an existing approved release is being updated you have to have a new version. So for example, if `v1.21.3` failed, you'll need to work with `v1.21.3.1`, so `make chocolatey VERSION=v1.21.3.1` below.

* Open up Gitpod, <https://gitpod.io/#https://github.com/ddev/ddev> and

```bash
cd /workspace/ddev
git checkout <tag>
sudo apt-get update && sudo apt-get install -y nsis
sudo .ci-scripts/nsis_setup.sh /usr/share/nsis
```

* Edit the checksum in `tools/chocolateyinstall.ps1` to match the released checksum of the `ddev-windows-installer` in `checksums.txt` of the release that is being repaired, for example, for `v1.21.3` this would be the checksum for `ddev_windows_installer.v1.21.3.exe` in [v1.21.3 checksums.txt](https://github.com/ddev/ddev/releases/download/v1.21.3/checksums.txt).
* Edit `url64` in `tools/chocolateyinstall.ps1` to be the intended actual DDEV download version - edit the version where it appears and edit the GitHub org. For example, if the actual version of DDEV to be downloaded is `v1.21.3` then put that there.

```bash
make chocolatey VERSION=<tag>
export CHOCOLATEY_API_KEY=key33333
cd .gotmp/bin/windows_amd64/chocolatey
docker run --rm -v $PWD:/tmp/chocolatey -w /tmp/chocolatey linuturk/mono-choco push -s https://push.chocolatey.org/ --api-key "${CHOCOLATEY_API_KEY}"

```

## Manually Updating AUR Repository

The AUR repository normally updates with the release process, so nothing needs to be done.

However, you can manually publish the release to [the DDEV AUR repository](https://aur.archlinux.org/packages/ddev-bin/). The README.md in the AUR Git repository (`ssh://aur@aur.archlinux.org/ddev-bin.git` or `https://aur.archlinux.org/ddev-bin.git`) has instructions on how to update, including how to do it with a Docker container, so it doesn’t have to be done on an ArchLinux or Manjaro VM.

## Manually Signing the Windows Installer

!!!tip "This is done by the release process, but the manual process is documented here."

This is done automatically by the release build on a dedicated Windows test runner (GitHub Actions runner) named `testbot-asus-win10pro`. You would need to do this process manually on that build machine or install the fob on another machine.

**After rebooting this machine, sometimes an automated reboot, the password for the security fob has to be re-entered or Windows signing will fail. We do this by opening up `tb-win11-06` using Chrome Remote Desktop (or manually physically opening it), opening Git Bash, and `cd ~/tmp && signtool sign gsudo.exe`. There happens to be a `gsudo.exe` there but it doesn’t matter what you sign—the idea is to pop up the GUI where you enter the password (which is in 1Password).**

### Basic Instructions

1. Install the suggested [Windows SDK](https://developer.microsoft.com/en-us/windows/downloads/windows-sdk/). Only the signing component is required.
2. Add the path of the kit binaries to the Windows system PATH, `C:/Program Files (x86)/Windows Kits/10/bin/10.0.22621.0/x64/`.
3. The keyfob and Safenet Authentication Client must be installed. The best documentation for the Safenet software is at <https://support.globalsign.com/ssl/ssl-certificates-installation/safenet-drivers>. You must configure the advanced client settings to “Enable single logon” or it will require the password on each run.
4. After `make windows_install` the `ddev-windows-installer.exe` will be in `.ddev/bin/windows_amd64/ddev_windows_installer.exe` and you can sign it with `signtool sign ddev-windows-installer.exe`.
5. If you need to install the GitHub self-hosted Windows runner, do it with the instructions in project settings → Actions → Runners.
6. Currently the `actions/cache` runner does not work out of the box on Windows, so you have to install tar and zstd as described in [this issue](https://github.com/actions/cache/issues/580#issuecomment-1165839728).

!!!tip "We shouldn’t use this high-security keyfob approach to signing on the next go-around with the certs."
    It’s way too difficult to manage, and the Safenet software is atrocious.

## APT and YUM/RPM Package Management

The Linux `apt` and `yum`/`rpm` packages are built and pushed by the `nfpms` and `furies` sections of the [.goreleaser.yml](https://github.com/ddev/ddev/blob/master/.goreleaser.yml) file.

* The actual packages are served by [gemfury.com](https://gemfury.com/).
* The name of the organization in GemFury is `drud`, managed at `https://manage.fury.io/dashboard/drud`.
* [Randy Fay](https://github.com/rfay), [Matt Stein](https://github.com/mattstein), and [Simon Gillis](https://github.com/gilbertsoft) are authorized as owners on this dashboard.
* The `pkg.ddev.com` domain name is set up as a custom alias for our package repositories; see `https://manage.fury.io/manage/drud/domains`. (Users do not see `drud` anywhere. Although we could have moved to a new organization for this, the existing repositories contain all the historical versions so it made sense to be less disruptive.)
* The `pkg.ddev.com` `CNAME` is managed in CloudFlare because `ddev.com` is managed there.
* The fury.io tokens are in DDEV’s shared 1Password account.

## Testing Release Creation

When significant changes are made to the `.goreleaser.yml` or related configuration, it's important to be able to test without actually deploying to `ddev/ddev/releases` of course. We have two ways to test the configuration; we can run `goreleaser` manually for simpler tests, or run a full release on `ddev-test/ddev` where needed.

### Running `goreleaser` manually

This approach is great for seeing what artifacts get created, without deploying them.

Prerequisites:

* GoReleaser Pro must be installed
* `export GORELEASER_KEY=<key>`

```bash
export REPOSITORY_OWNER=ddev-test
git tag <tagname> # Try to include context like PR number, for example v1.22.8-PR5824
make windows_amd64 windows_arm64 darwin_amd64 darwin_arm64 linux_amd64 linux_arm64 completions
goreleaser release --prepare --nightly --clean
```

This will create all the artifacts that would have been pushed in the `dist` directory. You can copy Linux packages from there to test them manually, download the built tarballs for use elsewhere, install Homebrew package manually, for example:

```bash
brew install ./dist/homebrew/Formula/ddev.rb
```

### Creating a test release on `ddev-test/ddev`

[ddev-test/ddev](https://github.com/ddev-test/ddev) is now set up for actual release testing. It has all or most of the environment variables set up already. It also acts against `ddev-test/homebrew-ddev` and `ddev-test/homebrew-ddev-edge` so you can test Homebrew publishing.

1. Create a branch on `ddev-test/ddev`.
2. Using the web UI, create a release using that branch as base. The release tag must start with `v1.`. Where possible, please use a release tag that includes context about the PR you are working against, like `v1.28.8-PR2022FixStuff`, and include in the release notes a link to the issue. The tag must be a valid Semantic Version tag, so don't use underscores, etc.
3. Test out the resulting artifacts that get published or deployed.
