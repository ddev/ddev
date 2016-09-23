package drudapi

import "encoding/json"

// Build ...
type Build struct {
	Created     string      `json:"_created,omitempty"`
	Etag        string      `json:"_etag,omitempty"`
	ID          string      `json:"_id,omitempty"`
	Updated     string      `json:"_updated,omitempty"`
	Type        string      `json:"type,omitempty"`
	Name        string      `json:"name,omitempty"`
	CloneURL    string      `json:"clone_url,omitempty"`
	Registry    string      `json:"registry,omitempty"`
	ImageName   string      `json:"image_name,omitempty"`
	Branch      string      `json:"branch,omitempty"`
	DeployName  string      `json:"deploy_name,omitempty"`
	State       string      `json:"state,omitempty"`
	Logs        string      `json:"logs,omitempty"`
	Build       int         `json:"build,omitempty"`
	Client      Client      `json:"client,omitempty"`
	Template    string      `json:"template,omitempty"`
	Application Application `json:"application,omitempty"`
	Container   Container   `json:"container,omitempty"`
	TagName     string      `json:"tag_name,omitempty"`
}

// Path ...
func (b Build) Path(method string) string {
	var path string

	if method == "POST" {
		path = "builds"
	} else {
		path = "builds/" + b.ID
	}
	return path
}

// Unmarshal ...
func (b *Build) Unmarshal(data []byte) error {
	err := json.Unmarshal(data, &b)
	return err
}

// JSON ...
func (b Build) JSON() []byte {
	b.ID = ""
	b.Etag = ""
	b.Created = ""
	b.Updated = ""

	jbytes, _ := json.Marshal(b)
	return jbytes
}

// PatchJSON ...
func (b Build) PatchJSON() []byte {
	b.ID = ""
	b.Etag = ""
	b.Created = ""
	b.Updated = ""
	// removing name because it has been setup as the id param in drudapi and cannot be  patched
	b.Name = ""

	jbytes, _ := json.Marshal(b)
	return jbytes
}

// ETAG ...
func (b Build) ETAG() string {
	return b.Etag
}

// BuildList ...
type BuildList struct {
	Items []Build `json:"_items"`
	Meta  struct {
		MaxResults int `json:"max_results"`
		Page       int `json:"page"`
		Total      int `json:"total"`
	} `json:"_meta"`
}

// Path ...
func (b BuildList) Path(method string) string {
	return "builds"
}

// Unmarshal ...
func (b *BuildList) Unmarshal(data []byte) error {
	err := json.Unmarshal(data, &b)
	return err
}
