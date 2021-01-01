package nodeps

// PHP Versions
const (
	PHP56 = "5.6"
	PHP70 = "7.0"
	PHP71 = "7.1"
	PHP72 = "7.2"
	PHP73 = "7.3"
	PHP74 = "7.4"
	PHP80 = "8.0"
)

// PHPDefault is the default PHP version, overridden by $DDEV_PHP_VERSION
const PHPDefault = PHP74

// ValidPHPVersions should be updated whenever PHP versions are added or removed, and should
// be used to ensure user-supplied values are valid.
var ValidPHPVersions = map[string]bool{
	PHP56: true,
	PHP70: true,
	PHP71: true,
	PHP72: true,
	PHP73: true,
	PHP74: true,
	PHP80: true,
}
