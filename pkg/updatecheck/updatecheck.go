package updatecheck

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	ddevgh "github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/util"
	"github.com/google/go-github/v52/github"
)

// AvailableUpdates returns true (along with a release URL) if there is an update available in the specified repo which is newer than the currentVersion string.
func AvailableUpdates(repoOrg string, repoName string, currentVersion string) (avail bool, newVersion string, releaseURL string, err error) {
	newVersion = ""
	ctx := context.Background()
	client := ddevgh.GetGithubClient(ctx)
	opt := &github.ListOptions{Page: 1}
	releases, _, err := client.Repositories.ListReleases(ctx, repoOrg, repoName, opt)
	if err != nil {
		return false, newVersion, "", err
	}

	if isReleaseVersion(currentVersion) {
		cv, err := semver.NewVersion(currentVersion)
		if err != nil {
			return false, newVersion, "", err
		}
		for _, release := range releases {
			if *release.Prerelease {
				continue
			}
			newReleaseVersion, err := semver.NewVersion(*release.TagName)
			if err != nil {
				return false, newVersion, "", err
			}
			newVersion = *release.TagName

			if cv.Compare(newReleaseVersion) == -1 {
				return true, newVersion, *release.HTMLURL, nil
			}
		}
	}

	return false, newVersion, "", nil
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

	if len(parts) > 2 || strings.Contains(version, "dirty") || strings.Contains(version, "-g") {
		return false
	}

	return true
}
