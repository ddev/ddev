package ddevapp

type settingsCreator func(*DdevApp) error
type uploadDir func(*DdevApp) string

// hookSuggestions should probably change its arg from string to app when
// config refactor is done.
type hookSuggestions func() []byte

type siteSettingsPaths func(app *DdevApp)

// AppTypeFuncs struct defines the functions that can be called (if populated)
// for a given appType.
type AppTypeFuncs struct {
	settingsCreator
	uploadDir
	hookSuggestions
	siteSettingsPaths
}

// AppTypeMatrix is a static map that defines the various functions to be called
// for each apptype (CMS).
// Example: AppTypeMatrix["drupal"]["7"] == { settingsCreator etc }
var AppTypeMatrix = map[string]AppTypeFuncs{
	"php": AppTypeFuncs{},
	"drupal7": AppTypeFuncs{
		createDrupalSettingsFile, getDrupalUploadDir, getDrupal7Hooks, setDrupalSiteSettingsPaths,
	},
	"drupal8": AppTypeFuncs{
		createDrupalSettingsFile, getDrupalUploadDir, getDrupal8Hooks, setDrupalSiteSettingsPaths,
	},
	"wordpress": AppTypeFuncs{
		createWordpressSettingsFile, getWordpressUploadDir, getWordpressHooks, setWordpressSiteSettingsPaths,
	},
	"backdrop": AppTypeFuncs{},
	"typo3":    AppTypeFuncs{},
}

// GetValidAppTypes returns the valid apptype keys from the AppTypeMatrix
func GetValidAppTypes() []string {
	keys := make([]string, 0, len(AppTypeMatrix))
	for k := range AppTypeMatrix {
		keys = append(keys, k)
	}
	return keys
}

// IsValidAppType checks to see if the given apptype string is a valid configured
// apptype.
func IsValidAppType(apptype string) bool {
	if _, ok := AppTypeMatrix[apptype]; ok {
		return true
	}
	return false
}

// CreateSettingsFile creates the settings file (like settings.php) for the
// provided app is the apptype has a settingsCreator function.
func CreateSettingsFile(l *DdevApp) error {
	if appFuncs, ok := AppTypeMatrix[l.GetType()]; ok && appFuncs.settingsCreator != nil {
		err := appFuncs.settingsCreator(l)
		return err
	}
	return nil
}

// UploadDirFunc returns the upload (public files) directory for the given app
func UploadDirFunc(l *DdevApp) string {
	if appFuncs, ok := AppTypeMatrix[l.GetType()]; ok && appFuncs.uploadDir != nil {
		uploadDir := appFuncs.uploadDir(l)
		return uploadDir
	}
	return ""
}

// GetHookSuggestions gets the actual text of the config.yaml hook suggestions
// for a given apptype
func GetHookSuggestions(apptype string) []byte {
	if appFuncs, ok := AppTypeMatrix[apptype]; ok && appFuncs.hookSuggestions != nil {
		suggestions := appFuncs.hookSuggestions()
		return suggestions
	}
	return []byte("")
}

// SetSiteSettingsPaths chooses and sets the settings.php/settings.local.php
// and related paths for a given app.
func SetSiteSettingsPaths(app *DdevApp) {
	if appFuncs, ok := AppTypeMatrix[app.Type]; ok && appFuncs.siteSettingsPaths != nil {
		appFuncs.siteSettingsPaths(app)
	}
}
