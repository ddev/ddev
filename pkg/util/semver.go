package util

import (
	"github.com/Masterminds/semver/v3"
)

// SemverValidate simply validates a string ver against a string constraint
// Returns bool, error
func SemverValidate(constraint string, ver string) (bool, error) {
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return false, err
	}
	sv, err := semver.NewVersion(ver)
	if err != nil {
		return false, err
	}
	match, _ := c.Validate(sv)
	return match, nil
}
