// +build !arm64

package nodeps

// MariaDBDefaultVersion is the default MariaDB version
const MariaDBDefaultVersion = MariaDB103

var ValidMariaDBVersions = map[string]bool{
	MariaDB55:  true,
	MariaDB100: true,
	MariaDB101: true,
	MariaDB102: true,
	MariaDB103: true,
	MariaDB104: true,
	MariaDB105: true,
}

// MariaDB Versions
const (
	MariaDB55  = "5.5"
	MariaDB100 = "10.0"
	MariaDB101 = "10.1"
	MariaDB102 = "10.2"
	MariaDB103 = "10.3"
	MariaDB104 = "10.4"
	MariaDB105 = "10.5"
)
