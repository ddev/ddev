package model

// DrupalConfig encapsulates all the configurations for a Drupal site.
type DrupalConfig struct {
	DeployName       string
	DeployURL        string
	DatabaseName     string
	DatabaseUsername string
	DatabasePassword string
	DatabaseHost     string
	DatabaseDriver   string
	DatabasePort     int
	DatabasePrefix   string
	HashSalt         string
	IsDrupal8        bool
}

// NewDrupalConfig produces a DrupalConfig object with default.
func NewDrupalConfig() *DrupalConfig {
	return &DrupalConfig{
		DatabaseName:     "data",
		DatabaseUsername: "root",
		DatabasePassword: "root",
		DatabaseHost:     "127.0.0.1",
		DatabaseDriver:   "mysql",
		DatabasePort:     3306,
		DatabasePrefix:   "",
		IsDrupal8:        false,
	}
}

// DrushConfig encapsulates configuration for a drush settings file.
type DrushConfig struct {
	DatabasePort int64
	DatabaseHost string
	IsDrupal8    bool
}

// NewDrushConfig produces a DrushConfig object with default.
func NewDrushConfig() *DrushConfig {
	return &DrushConfig{
		DatabaseHost: "127.0.0.1",
		DatabasePort: 3306,
		IsDrupal8:    false,
	}
}
