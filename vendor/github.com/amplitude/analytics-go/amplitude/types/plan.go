package types

type Plan struct {
	Branch    string `json:"branch,omitempty"`
	Source    string `json:"source,omitempty"`
	Version   string `json:"version,omitempty"`
	VersionID string `json:"versionId,omitempty"`
}
