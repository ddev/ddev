package types

import (
	"encoding/json"
	"fmt"
	"time"
)

// FlexibleString is a custom type that can unmarshal from string, number, or null
type FlexibleString struct {
	Value string
	IsSet bool
}

// UnmarshalJSON implements custom unmarshaling for FlexibleString
func (fs *FlexibleString) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as null first
	if string(data) == "null" {
		fs.Value = ""
		fs.IsSet = false
		return nil
	}

	// Try to unmarshal as string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		fs.Value = s
		fs.IsSet = true
		return nil
	}

	// Try to unmarshal as number (use %.0f to avoid scientific notation for large integers)
	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		// Check if it's an integer to format appropriately
		if n == float64(int64(n)) {
			fs.Value = fmt.Sprintf("%.0f", n)
		} else {
			fs.Value = fmt.Sprintf("%v", n)
		}
		fs.IsSet = true
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into FlexibleString", string(data))
}

// MarshalJSON implements custom marshaling for FlexibleString
func (fs FlexibleString) MarshalJSON() ([]byte, error) {
	if !fs.IsSet {
		return []byte("null"), nil
	}
	return json.Marshal(fs.Value)
}

// Addon represents a single DDEV add-on from the registry
type Addon struct {
	Title                 string         `json:"title"`
	GitHubURL             string         `json:"github_url"`
	Description           string         `json:"description"`
	User                  string         `json:"user"`
	Repo                  string         `json:"repo"`
	RepoID                int            `json:"repo_id"`
	DefaultBranch         string         `json:"default_branch"`
	TagName               FlexibleString `json:"tag_name"`
	DdevVersionConstraint string         `json:"ddev_version_constraint"`
	Dependencies          []string       `json:"dependencies"`
	Type                  string         `json:"type"`
	CreatedAt             string         `json:"created_at"`
	UpdatedAt             string         `json:"updated_at"`
	WorkflowStatus        string         `json:"workflow_status"`
	Stars                 int            `json:"stars"`
}

// AddonData represents the complete add-on registry from addons.ddev.com
type AddonData struct {
	UpdatedDateTime     time.Time `json:"updated_datetime"`
	TotalAddonsCount    int       `json:"total_addons_count"`
	OfficialAddonsCount int       `json:"official_addons_count"`
	ContribAddonsCount  int       `json:"contrib_addons_count"`
	Addons              []Addon   `json:"addons"`
}

// FindAddon looks up an addon by owner/repo format (e.g., "ddev/ddev-redis")
func (a *AddonData) FindAddon(ownerRepo string) *Addon {
	for i := range a.Addons {
		addonTitle := a.Addons[i].User + "/" + a.Addons[i].Repo
		if addonTitle == ownerRepo {
			return &a.Addons[i]
		}
	}
	return nil
}
