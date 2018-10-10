package ddevapp

// Providers
const (
	ProviderDrudS3   = "drud-s3"
	ProviderPantheon = "pantheon"

	// ProviderDefault contains the name of the default provider which will be used if one is not otherwise specified.
	ProviderDefault = "default"
)

// ValidProviders should be updated whenever provider plugins are added or removed, and should
// be used to ensure user-supplied values are valid.
var ValidProviders = map[string]bool{
	ProviderDefault:  true,
	ProviderDrudS3:   true,
	ProviderPantheon: true,
}

// PHP Versions
const (
	PHP56 = "5.6"
	PHP70 = "7.0"
	PHP71 = "7.1"
	PHP72 = "7.2"
)

// PHPDefault is the default PHP version, overridden by $DDEV_PHP_VERSION
const PHPDefault = PHP71

// ValidPHPVersions should be updated whenever PHP versions are added or removed, and should
// be used to ensure user-supplied values are valid.
var ValidPHPVersions = map[string]bool{
	PHP56: true,
	PHP70: true,
	PHP71: true,
	PHP72: true,
}

// Webserver types
const (
	WebserverNginxFPM  = "nginx-fpm"
	WebserverApacheFPM = "apache-fpm"
	WebserverApacheCGI = "apache-cgi"
)

// WebserverDefault is the default webserver type, overridden by $DDEV_WEBSERVER_TYPE
var WebserverDefault = WebserverNginxFPM

// ValidWebserverTypes should be updated whenever supported webserver types are added or
// removed, and should be used to ensure user-supplied values are valid.
var ValidWebserverTypes = map[string]bool{
	WebserverNginxFPM:  true,
	WebserverApacheFPM: true,
	WebserverApacheCGI: true,
}

// App types
const (
	AppTypeBackdrop  = "backdrop"
	AppTypeDrupal6   = "drupal6"
	AppTypeDrupal7   = "drupal7"
	AppTypeDrupal8   = "drupal8"
	AppTypePHP       = "php"
	AppTypeTYPO3     = "typo3"
	AppTypeWordpress = "wordpress"
)

// IsValidProvider is a helper function to determine if a provider value is valid, returning
// true if the supplied provider is valid and false otherwise.
func IsValidProvider(provider string) bool {
	if _, ok := ValidProviders[provider]; !ok {
		return false
	}

	return true
}

// GetValidProviders is a helper function that returns a list of valid providers.
func GetValidProviders() []string {
	s := make([]string, len(ValidProviders))

	for p := range ValidProviders {
		s = append(s, p)
	}

	return s
}

// IsValidPHPVersion is a helper function to determine if a PHP version is valid, returning
// true if the supplied PHP version is valid and false otherwise.
func IsValidPHPVersion(phpVersion string) bool {
	if _, ok := ValidPHPVersions[phpVersion]; !ok {
		return false
	}

	return true
}

// IsValidWebserverType is a helper function to determine if a webserver type is valid, returning
// true if the supplied webserver type is valid and false otherwise.
func IsValidWebserverType(webserverType string) bool {
	if _, ok := ValidWebserverTypes[webserverType]; !ok {
		return false
	}

	return true
}

// IsValidAppType is a helper function to determine if an app type is valid, returning
// true if the given app type is valid and configured and false otherwise.
func IsValidAppType(apptype string) bool {
	if _, ok := appTypeMatrix[apptype]; !ok {
		return false
	}

	return true
}

// GetValidAppTypes returns the valid apptype keys from the appTypeMatrix
func GetValidAppTypes() []string {
	keys := make([]string, 0, len(appTypeMatrix))
	for k := range appTypeMatrix {
		keys = append(keys, k)
	}
	return keys
}
