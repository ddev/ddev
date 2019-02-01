## `ddev` Release Checklist 
- [ ] Create provisional tagged images. `git fetch upstream && git checkout upstream/master && cd containers` and `foreach item in *; do pushd $item; make push VERSION=<release_version> DOCKER_ARGS=--no-cache ; popd; done`
- [ ] Update the default container versions in `pkg/version/version.go` and create a pull request
- [ ] Ensure all updates have been merged into the master branch
- [ ] Create a tag for the new version according to the instructions below, initiating a tag build
- [ ] Build and push artifacts with the .circleci/trigger_release.sh tool: `.circleci/trigger_release.sh circlepikey0908b3443ea58316baf928b <VERSION> githubpersonaltokenc590a1ad9f7c353962dea  | jq -r 'del(.circle_yml)'`
- [ ] Add the commit list (`git log vXXX..vYYY --oneline --decorate=no`) to the release page
- [ ] Download and sign the Windows installer executable according to the steps below
- [ ] Generate a new SHA checksum for the signed Windows installer: `shasum -a256 ${artifact} > ${artifact}.sha256.txt`
- [ ] Upload the signed Windows installer to the release page
- [ ] Remove the unsigned Windows installer (if it still exists)
- [ ] Download and confirm the integrity of each artifact with `shasum -a256 -c ${artifact}.sha256.txt`
- [ ] Update the `ddev` [Homebrew formula](https://github.com/drud/homebrew-ddev) with the MacOS `.tar.gz` and SHA checksum
- [ ] Test `brew upgrade ddev` and make sure ddev is the right version and behaves well
- [ ] Test the Windows installer and confirm it's signed correctly
- [ ] Update the release page with specifics about the current release
- [ ] Publish the release
- [ ] Ensure the new version is marked as active on [ReadTheDocs](https://readthedocs.org/dashboard/ddev/versions/)

### Creating a Tag

1. Fetch all changes locally: `git fetch upstream`
2. Merge updates into local master branch: `git merge upstream/master`
3. Confirm the state of the master branch: `git log --oneline`
4. Create a tag pointing to the current revision: `git tag vXXX` where `vXXX` is the version being released
5. Push the tag to the remote: `git push upstream vXXX`

### Signing with Windows installer

See the [Digicert instructions](https://www.digicert.com/code-signing/signcode-signtool-command-line.htm)

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
