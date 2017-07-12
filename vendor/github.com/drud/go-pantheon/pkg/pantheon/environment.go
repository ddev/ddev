package pantheon

import (
	"encoding/json"
	"fmt"
)

// Environment contains meta-data about a specific environment
type Environment struct {
	Name               string `json:"-"`
	DNSZone            string `json:"dns_zone"`
	EnvironmentCreated int64  `json:"environment_created"`
	Lock               struct {
		Locked   bool        `json:"locked"`
		Password interface{} `json:"password"`
		Username interface{} `json:"username"`
	} `json:"lock"`
	Maintenance struct {
		Enabled bool `json:"enabled"`
	} `json:"maintenance"`
	Randseed     string `json:"randseed"`
	StyxCluster  string `json:"styx_cluster"`
	TargetCommit string `json:"target_commit"`
	TargetRef    string `json:"target_ref"`
}

// EnvironmentList provides a list of environments for a given site.
type EnvironmentList struct {
	SiteID       string
	Environments map[string]Environment
}

// NewEnvironmentList creates an EnvironmentList for a given site. You are responsible for making the HTTP request.
func NewEnvironmentList(siteID string) *EnvironmentList {
	return &EnvironmentList{
		SiteID:       siteID,
		Environments: make(map[string]Environment),
	}
}

// Path returns the API endpoint which can be used to get a SiteList for the current user.
func (el EnvironmentList) Path(method string, auth AuthSession) string {
	return fmt.Sprintf("/sites/%s/environments", el.SiteID)
}

// JSON prepares the EnvironmentList for HTTP transport.
func (el EnvironmentList) JSON() ([]byte, error) {
	return json.Marshal(el.Environments)
}

// Unmarshal is responsible for converting a HTTP response into a EnvironmentList struct.
func (el *EnvironmentList) Unmarshal(data []byte) error {
	err := json.Unmarshal(data, &el.Environments)
	if err != nil {
		return err
	}

	if len(el.Environments) > 0 {
		for name, env := range el.Environments {
			env.Name = name
			el.Environments[name] = env
		}
	}

	return nil
}
