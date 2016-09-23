# drudapi
--
    import "github.com/drud/drud-go/drudapi"


## Usage

#### type Application

```go
type Application struct {
	AppID        string   `json:"app_id"`
	Client       Client   `json:"client"`
	Deploys      []Deploy `json:"deploys"`
	GithubHookID int      `json:"github_hook_id"`
	RepoOrg      string   `json:"repo_org"`
	Name         string   `json:"name"`
	Repo         string   `json:"repo"`
	Created      string   `json:"_created,omitempty"`
	Etag         string   `json:"_etag,omitempty"`
	ID           string   `json:"_id,omitempty"`
	Updated      string   `json:"_updated,omitempty"`
}
```

Application ...

#### func (Application) ETAG

```go
func (a Application) ETAG() string
```
ETAG ...

#### func (Application) JSON

```go
func (a Application) JSON() []byte
```
JSON ...

#### func (Application) PatchJSON

```go
func (a Application) PatchJSON() []byte
```
PatchJSON ...

#### func (Application) Path

```go
func (a Application) Path(method string) string
```
Path ...

#### func (*Application) Unmarshal

```go
func (a *Application) Unmarshal(data []byte) error
```
Unmarshal ...

#### type ApplicationList

```go
type ApplicationList struct {
	Name  string
	Items []Application `json:"_items"`
	Meta  struct {
		MaxResults int `json:"max_results"`
		Page       int `json:"page"`
		Total      int `json:"total"`
	} `json:"_meta"`
}
```

ApplicationList entity

#### func (ApplicationList) Path

```go
func (a ApplicationList) Path(method string) string
```
Path ...

#### func (*ApplicationList) Unmarshal

```go
func (a *ApplicationList) Unmarshal(data []byte) error
```
Unmarshal ...

#### type Build

```go
type Build struct {
	Created  string `json:"_created,omitempty"`
	Etag     string `json:"_etag,omitempty"`
	ID       string `json:"_id,omitempty"`
	Updated  string `json:"_updated,omitempty"`
	Name     string `json:"name,omitempty"`
	RepoName string `json:"repo_name"`
	Registry string `json:"registry"`
	Branch   string `json:"branch"`
	State    string `json:"state"`
	Logs     string `json:"logs"`
	Build    int    `json:"build"`
	Client   Client `json:"client"`
}
```

Build ...

#### func (Build) ETAG

```go
func (b Build) ETAG() string
```
ETAG ...

#### func (Build) JSON

```go
func (b Build) JSON() []byte
```
JSON ...

#### func (Build) PatchJSON

```go
func (b Build) PatchJSON() []byte
```
PatchJSON ...

#### func (Build) Path

```go
func (b Build) Path(method string) string
```
Path ...

#### func (*Build) Unmarshal

```go
func (b *Build) Unmarshal(data []byte) error
```
Unmarshal ...

#### type BuildList

```go
type BuildList struct {
	Items []Build `json:"_items"`
	Meta  struct {
		MaxResults int `json:"max_results"`
		Page       int `json:"page"`
		Total      int `json:"total"`
	} `json:"_meta"`
}
```

BuildList ...

#### func (BuildList) Path

```go
func (b BuildList) Path(method string) string
```
Path ...

#### func (*BuildList) Unmarshal

```go
func (b *BuildList) Unmarshal(data []byte) error
```
Unmarshal ...

#### type Client

```go
type Client struct {
	Created string `json:"_created,omitempty"`
	Etag    string `json:"_etag,omitempty"`
	ID      string `json:"_id,omitempty"`
	Updated string `json:"_updated,omitempty"`
	Email   string `json:"email"`
	Name    string `json:"name,omitempty"`
	Phone   string `json:"phone"`
}
```

Client ...

#### func (Client) ETAG

```go
func (c Client) ETAG() string
```
ETAG ...

#### func (Client) JSON

```go
func (c Client) JSON() []byte
```
JSON ...

#### func (Client) PatchJSON

```go
func (c Client) PatchJSON() []byte
```
PatchJSON ...

#### func (Client) Path

```go
func (c Client) Path(method string) string
```
Path ...

#### func (*Client) Unmarshal

```go
func (c *Client) Unmarshal(data []byte) error
```
Unmarshal ...

#### type ClientList

```go
type ClientList struct {
	Items []Client `json:"_items"`
	Meta  struct {
		MaxResults int `json:"max_results"`
		Page       int `json:"page"`
		Total      int `json:"total"`
	} `json:"_meta"`
}
```

ClientList ...

#### func (ClientList) Path

```go
func (c ClientList) Path(method string) string
```
Path ...

#### func (*ClientList) Unmarshal

```go
func (c *ClientList) Unmarshal(data []byte) error
```
Unmarshal ...

#### type Container

```go
type Container struct {
	Created      string `json:"_created,omitempty"`
	Etag         string `json:"_etag,omitempty"`
	ID           string `json:"_id,omitempty"`
	Updated      string `json:"_updated,omitempty"`
	Name         string `json:"name,omitempty"`
	Registry     string `json:"registry"`
	Branch       string `json:"branch"`
	GithubHookID int    `json:"github_hook_id"`
	Client       Client `json:"client"`
}
```

Container ...

#### func (Container) ETAG

```go
func (c Container) ETAG() string
```
ETAG ...

#### func (Container) JSON

```go
func (c Container) JSON() []byte
```
JSON ...

#### func (Container) PatchJSON

```go
func (c Container) PatchJSON() []byte
```
PatchJSON ...

#### func (Container) Path

```go
func (c Container) Path(method string) string
```
Path ...

#### func (*Container) Unmarshal

```go
func (c *Container) Unmarshal(data []byte) error
```
Unmarshal ...

#### type ContainerList

```go
type ContainerList struct {
	Items []Container `json:"_items"`
	Meta  struct {
		MaxResults int `json:"max_results"`
		Page       int `json:"page"`
		Total      int `json:"total"`
	} `json:"_meta"`
}
```

ContainerList ...

#### func (ContainerList) Path

```go
func (c ContainerList) Path(method string) string
```
Path ...

#### func (*ContainerList) Unmarshal

```go
func (c *ContainerList) Unmarshal(data []byte) error
```
Unmarshal ...

#### type Credentials

```go
type Credentials struct {
	Username   string `json:"username"`
	Password   string
	Token      string `json:"auth_token"`
	AdminToken string `json:"admin_token"`
}
```

Credentials gets passed around to functions for authenticating with the api

#### type Deploy

```go
type Deploy struct {
	Name          string `json:"name,omitempty"`
	Controller    string `json:"controller,omitempty"`
	Branch        string `json:"branch,omitempty"`
	Hostname      string `json:"hostname,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
	BasicAuthUser string `json:"basicauth_user,omitempty"`
	BasicAuthPass string `json:"basicauth_pass,omitempty"`
}
```

Deploy ...

#### type Entity

```go
type Entity interface {
	Path(method string) string   // returns the path that must be added to host to get the entity
	Unmarshal(data []byte) error // unmarshal json into entity's fields
	JSON() []byte                //returns the entity's json representation
	PatchJSON() []byte           //returns the entity's json repr minus id field
	ETAG() string                // returns etag
}
```

Entity interface represents eve entities in some functinos

#### type EntityGetter

```go
type EntityGetter interface {
	Path(method string) string   // returns the path that must be added to host to get the entity
	Unmarshal(data []byte) error // unmarshal json into entity's fields
}
```

EntityGetter lets you pass entity/entity list to Get without having to implement
all the same methods for both

#### type Provider

```go
type Provider struct {
	Name    string   `json:"name"`
	Regions []Region `json:"regions"`
}
```

Provider ...

#### type Region

```go
type Region struct {
	Name string `json:"name"`
}
```

Region ...

#### type Request

```go
type Request struct {
	Host  string // base path of the api  e.g. https://drudapi.genesis.drud.io/v0.1
	Query string // optional query params e.g. `where={"name":"fred"}``
	Auth  *Credentials
}
```

Request type used for building requests

#### func (*Request) Delete

```go
func (r *Request) Delete(entity Entity) error
```
Delete ...

#### func (*Request) Get

```go
func (r *Request) Get(entity EntityGetter) error
```
Get ...

#### func (*Request) Patch

```go
func (r *Request) Patch(entity Entity) error
```
Patch ...

#### func (*Request) Post

```go
func (r *Request) Post(entity Entity) error
```
Post ...

#### type User

```go
type User struct {
	Username string      `json:"username"`
	Hashpw   string      `json:"hashpw"`
	Token    string      `json:"auth_token,omitempty"`
	Created  string      `json:"_created,omitempty"`
	Etag     string      `json:"_etag,omitempty"`
	ID       string      `json:"_id,omitempty"`
	Updated  string      `json:"_updated,omitempty"`
	Auth     Credentials `json:"-"`
}
```

User represents a user entity from the api

#### func (User) ETAG

```go
func (u User) ETAG() string
```
ETAG ...

#### func (User) JSON

```go
func (u User) JSON() []byte
```
JSON ...

#### func (User) PatchJSON

```go
func (u User) PatchJSON() []byte
```
PatchJSON ...

#### func (User) Path

```go
func (u User) Path(method string) string
```
Path ...

#### func (*User) Unmarshal

```go
func (u *User) Unmarshal(data []byte) error
```
Unmarshal ...

#### type UserList

```go
type UserList struct {
	Items []User `json:"_items"`
	Meta  struct {
		MaxResults int `json:"max_results"`
		Page       int `json:"page"`
		Total      int `json:"total"`
	} `json:"_meta"`
}
```

UserList entity

#### func (UserList) Path

```go
func (u UserList) Path(method string) string
```
Path ...

#### func (*UserList) Unmarshal

```go
func (u *UserList) Unmarshal(data []byte) error
```
Unmarshal ...
