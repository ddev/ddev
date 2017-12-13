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
	"":    AppTypeFuncs{},
	"php": AppTypeFuncs{},
	"drupal7": AppTypeFuncs{
		CreateDrupalSettingsFile, GetDrupalUploadDir, GetDrupal7Hooks, SetDrupalSiteSettingsPaths,
	},
	"drupal8": AppTypeFuncs{
		CreateDrupalSettingsFile, GetDrupalUploadDir, GetDrupal8Hooks, SetDrupalSiteSettingsPaths,
	},
	"wordpress": AppTypeFuncs{
		CreateWordpressSettingsFile, nil, GetWordpressHooks, SetWordpressSiteSettingsPaths,
	},
	"backdrop": AppTypeFuncs{},
}

// GetAllowedAppTypes returns the valid apptype keys from the AppTypeMatrix
func GetAllowedAppTypes() []string {
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

func UploadDirFunc(l *DdevApp) string {
	if appFuncs, ok := AppTypeMatrix[l.GetType()]; ok && appFuncs.uploadDir != nil {
		uploadDir := appFuncs.uploadDir(l)
		return uploadDir
	}
	return ""
}

func GetHookSuggestions(apptype string) []byte {
	if appFuncs, ok := AppTypeMatrix[apptype]; ok && appFuncs.hookSuggestions != nil {
		suggestions := appFuncs.hookSuggestions()
		return suggestions
	}
	return []byte("")
}

func SetSiteSettingsPaths(app *DdevApp) {
	if appFuncs, ok := AppTypeMatrix[app.Type]; ok && appFuncs.siteSettingsPaths != nil {
		appFuncs.siteSettingsPaths(app)
	}
}
