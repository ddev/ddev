package types

type IngestionMetadata struct {
	SourceName    string `json:"source_name,omitempty"`
	SourceVersion string `json:"source_version,omitempty"`
}
