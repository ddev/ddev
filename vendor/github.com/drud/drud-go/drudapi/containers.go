package drudapi

import "encoding/json"

// Container ...
type Container struct {
	Created      string `json:"_created,omitempty"`
	Etag         string `json:"_etag,omitempty"`
	ID           string `json:"_id,omitempty"`
	Updated      string `json:"_updated,omitempty"`
	Name         string `json:"name,omitempty"`
	Registry     string `json:"registry,omitempty"`
	Branch       string `json:"branch,omitempty"`
	GithubHookID int    `json:"github_hook_id,omitempty"`
	Client       Client `json:"client,omitempty"`
}

// Path ...
func (c Container) Path(method string) string {
	var path string

	if method == "POST" {
		path = "containers"
	} else {
		path = "containers/" + c.ID
	}
	return path
}

// Unmarshal ...
func (c *Container) Unmarshal(data []byte) error {
	err := json.Unmarshal(data, &c)
	return err
}

// JSON ...
func (c Container) JSON() []byte {
	c.ID = ""
	c.Etag = ""
	c.Created = ""
	c.Updated = ""

	jbytes, _ := json.Marshal(c)
	return jbytes
}

// PatchJSON ...
func (c Container) PatchJSON() []byte {
	c.ID = ""
	c.Etag = ""
	c.Created = ""
	c.Updated = ""
	// removing name because it has been setup as the id param in drudapi and cannot be  patched
	c.Name = ""

	jbytes, _ := json.Marshal(c)
	return jbytes
}

// ETAG ...
func (c Container) ETAG() string {
	return c.Etag
}

// ContainerList ...
type ContainerList struct {
	Items []Container `json:"_items"`
	Meta  struct {
		MaxResults int `json:"max_results"`
		Page       int `json:"page"`
		Total      int `json:"total"`
	} `json:"_meta"`
}

// Path ...
func (c ContainerList) Path(method string) string {
	return "containers"
}

// Unmarshal ...
func (c *ContainerList) Unmarshal(data []byte) error {
	err := json.Unmarshal(data, &c)
	return err
}
