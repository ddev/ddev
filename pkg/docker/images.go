package docker

import (
	"fmt"
	"regexp"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/versionconstants"
)

// DdevImageTagLabel is the image label recording the tag an image was built as.
// The tag is otherwise lost in a derived image, so this is what lets DDEV tell
// which of its image generations a pinned webimage/dbimage came from.
const DdevImageTagLabel = "com.ddev.image-tag"

// releaseTagPattern matches a vX.Y.Z release tag.
var releaseTagPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// resolveImageTag prefers branch over tag when branch is a vX.Y.Z release
// tag. release-prep.sh stamps <TagVar>Branch with the release tag while
// leaving <TagVar> as the content hash autotag.sh relies on, but both
// resolve to the identical manifest, so a released binary can pull by the
// readable tag.
func resolveImageTag(tag, branch string) string {
	if releaseTagPattern.MatchString(branch) {
		return branch
	}
	return tag
}

// GetWebImage returns the correctly formatted web image:tag reference
func GetWebImage() string {
	fullWebImg := versionconstants.WebImg
	if globalconfig.DdevGlobalConfig.UseHardenedImages {
		fullWebImg = fullWebImg + "-prod"
	}
	return fmt.Sprintf("%s:%s", fullWebImg, resolveImageTag(versionconstants.WebTag, versionconstants.WebTagBranch))
}

// GetDBImage returns the correctly formatted db image:tag reference
func GetDBImage(dbType string, dbVersion string) string {
	v := nodeps.MariaDBDefaultVersion
	if dbVersion != "" {
		v = dbVersion
	}
	if dbType == "" {
		dbType = nodeps.MariaDB
	}
	switch dbType {
	case nodeps.Postgres:
		return fmt.Sprintf("%s:%s", dbType, v)
	case nodeps.MySQL:
		fallthrough
	case nodeps.MariaDB:
		fallthrough
	default:
		return fmt.Sprintf("%s-%s-%s:%s", versionconstants.DBImg, dbType, v, resolveImageTag(versionconstants.BaseDBTag, versionconstants.BaseDBTagBranch))
	}
}

// GetSSHAuthImage returns the correctly formatted sshauth image:tag reference
func GetSSHAuthImage() string {
	return fmt.Sprintf("%s:%s", versionconstants.SSHAuthImage, resolveImageTag(versionconstants.SSHAuthTag, versionconstants.SSHAuthTagBranch))
}

// GetRouterImage returns the router image:tag reference
func GetRouterImage() string {
	return fmt.Sprintf("%s:%s", versionconstants.TraefikRouterImage, resolveImageTag(versionconstants.TraefikRouterTag, versionconstants.TraefikRouterTagBranch))
}

// GetXhguiImage returns the xhgui image:tag reference
func GetXhguiImage() string {
	return fmt.Sprintf("%s:%s", versionconstants.XhguiImage, resolveImageTag(versionconstants.XhguiTag, versionconstants.XhguiTagBranch))
}
