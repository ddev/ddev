## `ddev` Release Checklist 
1. Create provisional tagged images. `git fetch upstream && git checkout upstream/master && cd containers` and `for item in *; do pushd $item; make push VERSION=<release_version> DOCKER_ARGS=--no-cache ; popd; done`
2. Update the default container versions in `pkg/version/version.go` and create a pull request
3. Ensure all updates have been merged into the master branch
4. Create a tag for the new version according to the instructions below, initiating a tag build
5. Build and push artifacts with the .circleci/trigger_release.sh tool: `.circleci/trigger_release.sh --release-tag=v1.10.0 --circleci-token=circleToken900908b3443ea58316baf928b --github-token=githubPersonalToken853ae6f72c40525cd21036f742904a   --windows-signing-password=windowscodepassword --macos-signing-password=macossigningpassword | jq -r 'del(.circle_yml)'  | jq -r 'del(.circle_yml)'`
6. Add the commit list (`git log vXXX..vYYY --oneline --decorate=no`) to the release page
7. Update the `ddev` homebrew formulas (ddev-edge and ddev) as necessary, https://github.com/drud/homebrew-ddev and https://github.com/drud/homebrew-ddev-edge, with the source .tar.gz and SHA checksum of the tarball and the bottle builds and tarballs. The bottles and checksums for macOS (sierra) and x86_64_linux are built and pushed to the release page automatically by the CircleCI release build process.
8. Test `brew upgrade ddev` both on macOS and Linux and make sure ddev is the right version and behaves well
9. Test the Windows installer and confirm it's signed correctly
10. Update the release page with specifics about the current release
11. Publish the release (unmark it as "prerelease")
12. Download the ddev_chocolatey tarball and extract it. cd into the extraction directory and push it to chocolatey with `docker run --rm -v $PWD:/tmp/chocolatey -w /tmp/chocolatey linuturk/mono-choco push -s https://push.chocolatey.org/ --api-key=choco-apikey-a720-7890909913f7`  (Although this ought to be done by the release build process on CircleCI it's not successful as of v1.7.1.)
13. On [ReadTheDocs](https://readthedocs.org/projects/ddev/builds) click the button to "build version" "latest".  Then on [versions](https://readthedocs.org/projects/ddev/versions/) page make sure that "stable" reflects the hash of the new version.

### Creating a Tag

1. Fetch all changes locally: `git fetch upstream`
2. Merge updates into local master branch: `git merge upstream/master`
3. Confirm the state of the master branch: `git log --oneline`
4. Create a tag pointing to the current revision: `git tag vXXX` where `vXXX` is the version being released
5. Push the tag to the remote: `git push upstream vXXX`

### Signing with Windows installer

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

### Additional Information

This checklist outlines the release process specific to building, packaging, and testing `ddev` releases.  For additional information on the product release process, please see the [Product Release](https://github.com/drud/community/blob/master/development/product_release.md) instructions.
