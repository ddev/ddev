package drudapi

// Region ...
type Region struct {
	Name string `json:"name"`
}

// Provider ...
type Provider struct {
	Name    string   `json:"name"`
	Regions []Region `json:"regions"`
}

// FormatPlural is a simple wrapper which returns different strings based on the count value.
func FormatPlural(count int, single string, plural string) string {
	if count == 1 {
		return single
	}
	return plural
}
