package drudapi

import (
	"log"
	"testing"
)

func TestGetApplication200(t *testing.T) {
	expectedResp := `
{
    "_updated": "Mon, 23 May 2016 20:23:37 GMT",
    "github_hook_id": 123455,
    "name": "killah",
    "app_id": "mumbojumbo",
    "repo_org": "ourorg",
    "repo": "",
    "client": {
        "_updated": "Mon, 23 May 2016 20:23:34 GMT",
        "name": "somebiz",
        "email": "",
        "phone": "",
        "_created": "Mon, 23 May 2016 20:23:34 GMT",
        "_id": "574366c6e2638a001f430115",
        "_etag": "9906d3a8584f0fabbd96451013b38d20fff5f5d3"
    },
    "deploys": [
        {
            "protocol": "http",
            "basicauth_pass": "drud",
            "basicauth_user": "drud",
            "name": "default",
            "branch": "master"
        }
    ],
    "_created": "Mon, 23 May 2016 20:23:37 GMT",
    "_id": "d546fh5ert456",
    "_etag": "34b4d4c312a2d5cb916fef5330b2a14f53acac4b"
}`
	server := getTestServer(200, expectedResp)
	defer server.Close()

	r := Request{
		Host: server.URL,
		Auth: &Credentials{
			AdminToken: "dfgdfg",
		},
	}

	a := &Application{
		AppID: "mumbojumbo",
	}

	err := r.Get(a)
	if err != nil {
		log.Fatal(err)
	}

	expect(t, a.GithubHookID, 123455)
	expect(t, a.Name, "killah")
	expect(t, a.RepoOrg, "ourorg")
	expect(t, a.Created, "Mon, 23 May 2016 20:23:37 GMT")
	expect(t, a.Etag, "34b4d4c312a2d5cb916fef5330b2a14f53acac4b")
	expect(t, a.Client.Name, "somebiz")

}

// TestPostClient tests that when posting a client the server receives data that is formed
// correctly and that given a proper response the client will know its id and etag
func TestPostApplication(t *testing.T) {
	expectedResp := `{
    "_updated": "Mon, 13 Jun 2016 19:43:57 GMT",
    "github_hook_id": 0,
    "name": "KubeJobWatcher",
    "app_id": "drud-kubejobwatcher",
    "repo_org": "",
    "repo": "",
    "client": {
        "_updated": "Tue, 07 Jun 2016 21:49:56 GMT",
        "name": "drud",
        "email": "",
        "phone": "",
        "_created": "Tue, 07 Jun 2016 21:49:56 GMT",
        "_id": "575741845aeade0018a29423",
        "_etag": "9fb9dd879b86757f0f6d0e65e3714b07abfa7b0d"
    },
    "_links": {
        "self": {
            "href": "application/drud-kubejobwatcher",
            "title": "Application"
        }
    },
    "deploys": [
        {
            "protocol": "http",
            "name": "default",
            "branch": "master",
            "auto_managed": false
        }
    ],
    "_created": "Mon, 13 Jun 2016 19:43:57 GMT",
    "_status": "OK",
    "_id": "98734598723094857023985",
    "_etag": "qwertyuiop12345"
}`

	reqData := map[string]string{
		"content-type":  "application/json",
		"authorization": "token dfgdfg",
		"endpoint":      "/application",
	}

	c := &Client{
		Name:  "testyclient",
		Email: "client@testy.com",
		Phone: "3192832323",
	}

	deploy := Deploy{
		Name:     "default",
		Protocol: "http",
	}

	app := &Application{
		Name:    "KubeJobWatcher",
		Client:  *c,
		Repo:    "",
		RepoOrg: "",
		Deploys: []Deploy{deploy},
	}

	reqData["payload"] = string(app.JSON())

	server := postTestServer(t, expectedResp, reqData)
	defer server.Close()

	r := Request{
		Host: server.URL,
		Auth: &Credentials{
			AdminToken: "dfgdfg",
		},
	}

	err := r.Post(app)
	if err != nil {
		log.Fatal(err)
	}

	expect(t, app.AppID, "drud-kubejobwatcher")
	expect(t, app.Etag, "qwertyuiop12345")
	expect(t, app.ID, "98734598723094857023985")
	expect(t, app.Client.ID, "575741845aeade0018a29423")

}
