package model

import "github.com/drud/ddev/pkg/util"

// WordpressConfig encapsulates all the configurations for a WordPress site.
type WordpressConfig struct {
	WPGeneric        bool
	DeployName       string
	DeployURL        string
	DatabaseName     string
	DatabaseUsername string
	DatabasePassword string
	DatabaseHost     string
	AuthKey          string
	SecureAuthKey    string
	LoggedInKey      string
	NonceKey         string
	AuthSalt         string
	SecureAuthSalt   string
	LoggedInSalt     string
	NonceSalt        string
	Docroot          string
	TablePrefix      string
}

// NewWordpressConfig produces a WordpressConfig object with defaults.
func NewWordpressConfig() *WordpressConfig {
	return &WordpressConfig{
		WPGeneric:        false,
		DatabaseName:     "db",
		DatabaseUsername: "db",
		DatabasePassword: "db",
		DatabaseHost:     "db",
		Docroot:          "/var/www/html/docroot",
		TablePrefix:      "wp_",
		AuthKey:          util.RandomString(64),
		AuthSalt:         util.RandomString(64),
		LoggedInKey:      util.RandomString(64),
		LoggedInSalt:     util.RandomString(64),
		NonceKey:         util.RandomString(64),
		NonceSalt:        util.RandomString(64),
		SecureAuthKey:    util.RandomString(64),
		SecureAuthSalt:   util.RandomString(64),
	}
}
