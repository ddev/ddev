package ddevapp

// newSettingsFile does the grunt work of determining which and where the
// settings file will be created, then sets permissions so it can do the right
// thing. It does housekeeping like chmodding the file and related directories.
// The work is done here instead of CreateSettingsFile() because it would
// be duplicated by the functions called in CreateSettingsFile().
//func newSettingsFile(app *DdevApp) (string, error) {
//
//	SetApptypeSettingsPaths(app)
//
//	// If neither settings file options are set, then
//	if app.SiteLocalSettingsPath == "" && app.SiteSettingsPath == "" {
//		return "", fmt.Errorf("Neither SiteLocalSettingsPath nor SiteSettingsPath is set")
//	}
//
//	// Drupal and WordPress love to change settings files to be unwriteable.
//	// Chmod them to something we can work with in the event that they already
//	// exist.
//	chmodTargets := []string{filepath.Dir(app.SiteSettingsPath), app.SiteLocalSettingsPath}
//	for _, fp := range chmodTargets {
//		if fileInfo, err := os.Stat(fp); !os.IsNotExist(err) {
//			perms := 0644
//			if fileInfo.IsDir() {
//				perms = 0755
//			}
//
//			err = os.Chmod(fp, os.FileMode(perms))
//			if err != nil {
//				return "", fmt.Errorf("could not change permissions on file %s to make it writeable: %v", fp, err)
//			}
//		}
//	}
//	settingsFilePath, err := CreateSettingsFile(app)
//	return settingsFilePath, err
//}
