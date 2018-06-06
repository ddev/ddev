## `ddev` Release Checklist 

- [ ] Create a release, initiating a CircleCI build
- [ ] Push all containers with `make push`
- [ ] Add the commit list to the release page
- [ ] Download artifacts from CircleCI and upload them (except for the Windows installer) to the release page
- [ ] Update the `ddev` [Homebrew formula](https://github.com/drud/homebrew-ddev) with the MacOS `.tar.gz` and SHA checksum
- [ ] Test `brew ddev upgrade`
- [ ] Sign the Windows installer according to steps in #840 (requires Windows and certificate)
- [ ] Create a SHA checksum (`sha256.txt`) for the signed Windows installer
- [ ] Upload the signed Windows installer and SHA checksum to the release page
- [ ] Test the Windows installer and confirm it's signed correctly
- [ ] Publish the release page 

### Additional Information

This checklist outlines the release process specific to building, packaging, and testing `ddev` releases.  For additional information on the product release process, please see the [Product Release](https://github.com/drud/community/blob/master/development/product_release.md) instructions.
