package model

// DdevSettingsFileSignature is the text we use to detect whether a settings file is managed by us.
// If this string is found, we assume we can replace/update the settings file.
const DdevSettingsFileSignature = "#ddev-generated"
