package pantheon

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"
)

// AuthSession is responsible for authentication and session management with the pantheon API.
type AuthSession struct {
	Token   string `json:"machine_token"`
	Email   string `json:"email,omitempty"`
	Client  string `json:"client"`
	Expires int64  `json:"expires_at"`
	Session string `json:"session,omitempty"`
	UserID  string `json:"user_id,omitempty"`
}

// NewAuthSession creates a new authentication session using a given token. It is the preferred method for creating a new AuthSession.
func NewAuthSession(token string) *AuthSession {
	return &AuthSession{
		Token:  token,
		Client: "terminus",
	}
}

// Request performs the HTTP request. You must provide it a request type and the entity the result should be stored in.
func (a *AuthSession) Request(requestType string, entity RequestEntity) error {
	var json []byte
	var err error

	requestType = strings.ToUpper(requestType)
	if requestType != "GET" {
		json, err = entity.JSON()
		if err != nil {
			return err
		}
	}

	headers, err := a.Headers()
	if err != nil {
		return err
	}

	bytes, err := httpRequest(requestType, entity.Path(requestType, *a), json, headers)

	if err != nil {
		return err
	}

	return entity.Unmarshal(bytes)
}

// Auth checks the expiration time of the current session (if there is one) and re-authenticates as necessary.
func (a *AuthSession) Auth() error {
	now := time.Now().UTC().Unix()
	// Determine if an expiration has happened.
	if a.Expires <= now {

		// If so, re-authenticate.
		json, err := a.JSON()
		if err != nil {
			return err
		}

		bytes, err := httpRequest("POST", a.Path("POST"), json, make(map[string]string))
		if err != nil {
			return err
		}

		return a.Unmarshal(bytes)
	}
	return nil
}

// Headers returns authentication headers needed for accessing the pantheon API. It should be used on all authenticated requests.
func (a *AuthSession) Headers() (map[string]string, error) {
	err := a.Auth()
	if err != nil {
		return nil, err
	}

	return map[string]string{"Authorization": fmt.Sprintf("Bearer %s", a.Session)}, nil
}

// Path returns the API endpoint for authenticating.
func (a *AuthSession) Path(method string) string {
	return "/authorize/machine-token"
}

// JSON prepares the AuthSession struct for an HTTP request by stripping out fields which should not be sent, and returning the JSON representation of the struct as byte array.
func (a AuthSession) JSON() ([]byte, error) {
	a.Email = ""
	a.UserID = ""
	a.Session = ""
	a.Expires = 0
	return json.Marshal(a)
}

// Unmarshal converts the byte array from an HTTP response back into the AuthSession struct.
func (a *AuthSession) Unmarshal(data []byte) error {
	return json.Unmarshal(data, a)
}

// GetUser returns the UserID for the current AuthSession.
func (a *AuthSession) GetUser() (string, error) {
	err := a.Auth()
	if err != nil {
		return "", err
	}

	return a.UserID, nil
}

// Write converts the current AuthSession to JSON and writes persists result to the specified location.
func (a *AuthSession) Write(location string) error {
	contents, err := json.Marshal(a)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(location, contents, 0644)
}

// Read loads the JSON representation of the AuthSession from the specified location.
func (a *AuthSession) Read(location string) error {
	bytes, err := ioutil.ReadFile(location)
	if err != nil {
		return err
	}

	return a.Unmarshal(bytes)
}
