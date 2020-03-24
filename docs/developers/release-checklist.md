# DDEV-Local Release Checklist

1. Create provisional tagged images. `git fetch upstream && git checkout upstream/master && cd containers` and `for item in *; do pushd $item; make push VERSION=<release_version> DOCKER_ARGS=--no-cache ; popd; done`

2. Update the default container versions in `pkg/version/version.go` and create a pull request
3. Ensure all updates have been merged into the master branch
4. Create a release for the new version using the github UI. It should be "prerelease" if it's only an edge release.
5. Add the commit list (`git log vXXX..vYYY --oneline --decorate=no`) to the release page
6. Update the `ddev` homebrew formulas (ddev-edge and ddev) as necessary, <https://github.com/drud/homebrew-ddev> and <https://github.com/drud/homebrew-ddev-edge,> with the source .tar.gz and SHA checksum of the tarball and the bottle builds and tarballs. The bottles and checksums for macOS (sierra) and x86_64_linux are built and pushed to the release page automatically by the CircleCI release build process. Test `brew upgrade ddev` both on macOS and Linux and make sure ddev is the right version and behaves well.
7. Download the ddev_chocolatey tarball and extract it. It should be available on a Windows machine or VM. (It can be network-mounted into a Windows VM). cd into the extraction directory and push it to chocolatey with `choco push -s https://push.chocolatey.org/ --api-key=choco-apikey-a720-asome-api-key` Before 2020-02-03 this coudl be done from Linux or macOS with mono-choco, but doesn't work any more due to [mono-choco#20](https://github.com/Linuturk/mono-choco/issues/20) ). After the push responses have come back, install with `choco install -y --pre ddev --version <version>` and verify correct behavior.
8. Publish the release to the ddev [AUR repository](https://aur.archlinux.org/packages/ddev-bin/). The README.md in the AUR git repo (`ssh://aur@aur.archlinux.org/ddev-bin.git` or `https://aur.archlinux.org/ddev-bin.git`) has instructions on how to update, including how to do it with a Docker container, so it doesn't have to be done on an ArchLinux or Manjaro VM.
9. Update the release page with full details about the current release
10. Publish the release (unmark it as "prerelease") if it's a normal (non-edge) release
11. On [ReadTheDocs](https://readthedocs.org/projects/ddev/builds) click the button to "build version" "latest".  Then on [versions](https://readthedocs.org/projects/ddev/versions/) page make sure that "stable" reflects the hash of the new version.

## Manually Signing with Windows installer

(This is done by the release process, but manual process documented here.)

See the [Digicert instructions](https://www.digicert.com/code-signing/signcode-signtool-command-line.htm)

Note that this is done automatically by the CircleCI release build if the signing password is included in trigger_release.sh.

Basic instructions:

1. On a Windows machine, install the certificate as suggested. You need the cert file and password, and you install it into Chrome or IE (This is a one-time operation)
2. Install the suggested [Windows SDK 10](https://developer.microsoft.com/en-us/windows/downloads/windows-10-sdk)
3. Install [Visual Studio Community 2015](https://msdn.microsoft.com/en-us/library/mt613162.aspx)
4. Run the [Developer Command Prompt](https://docs.microsoft.com/en-us/dotnet/framework/tools/developer-command-prompt-for-vs)
5. Sign the binary with something like `signtool sign /tr http://timestamp.digicert.com /td sha256 /fd sha256 /a z:\Downloads\ddev_windows_installer.v0.18.0.exe` (I did this with the downloaded ddev_windows_installer physically on my mac (Z: drive))
6. Generate a new sha256 file (I did this on mac): `shasum -a 256 ddev_windows_installer.v0.18.0.exe >ddev_windows_installer.v0.18.0.exe.sha256.txt`
7. Upload the ddev_windows_installer and sha256.txt to the release
