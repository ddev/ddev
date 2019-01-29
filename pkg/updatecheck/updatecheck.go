package updatecheck

import (
	"context"
	"strings"

	"time"

	"os"

	"github.com/Masterminds/semver"
	"github.com/drud/ddev/pkg/util"
	"github.com/google/go-github/github"
)

// AvailableUpdates returns true (along with a release URL) if there is an update available in the specified repo which is newer than the currentVersion string.
func AvailableUpdates(repoOrg string, repoName string, currentVersion string) (bool, string, error) {
	client := github.NewClient(nil)
	opt := &github.ListOptions{Page: 1}
	releases, _, err := client.Repositories.ListReleases(context.Background(), repoOrg, repoName, opt)
	if err != nil {
		return false, "", err
	}

	if isReleaseVersion(currentVersion) {
		cv, err := semver.NewVersion(currentVersion)
		if err != nil {
			return false, "", err
		}
		for _, release := range releases {
			if *release.Prerelease {
				continue
			}
			releaseVersion, err := semver.NewVersion(*release.TagName)
			if err != nil {
				return false, "", err
			}

			if cv.Compare(releaseVersion) == -1 {
				return true, *release.HTMLURL, nil
			}
		}
	}

	return false, "", nil
}

// IsUpdateNeeded returns true if the modification date on filepath is older than the duration specified.
func IsUpdateNeeded(filepath string, updateInterval time.Duration) (bool, error) {
	info, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		return true, ResetUpdateTime(filepath)
	} else if err != nil {
		return false, err
	}

	timeSinceMod := time.Since(info.ModTime())

	if timeSinceMod >= updateInterval {
		return true, nil
	}

	return false, nil
}

// ResetUpdateTime resets the file modification date on filepath by removing and re-creating the file.
func ResetUpdateTime(filepath string) error {
	err := os.Remove(filepath)
	_ = err // We don't actually care if remove failed. All we care about is that the create succeeds.
	file, err := os.Create(filepath)
	util.CheckClose(file)
	return err
}

// isReleaseVersion does a (very naive) check on whether a version string consistutes a release version or a dev build.
func isReleaseVersion(version string) bool {
	parts := strings.Split(version, "-")

	if len(parts) > 1 {
		return false
	}

	return true
}
