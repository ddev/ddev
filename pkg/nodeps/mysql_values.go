// +build !arm64

package nodeps

// ValidMySQLVersions is the versions of MySQL that are valid
var ValidMySQLVersions = map[string]bool{
	MySQL55: true,
	MySQL56: true,
	MySQL57: true,
	MySQL80: true,
}

// Oracle MySQL versions
const (
	MySQL55 = "5.5"
	MySQL56 = "5.6"
	MySQL57 = "5.7"
	MySQL80 = "8.0"
)
