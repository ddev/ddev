package drudapi

import (
	"log"
	"testing"
)

func TestGetContainer200(t *testing.T) {
	expectedResp := `{
    "_updated": "Mon, 23 May 2016 20:23:34 GMT",
    "name": "mycontainer",
    "branch": "master",
    "registry": "docker.io",
	"github_hook_id": 5,
    "_created": "Mon, 23 May 2016 20:23:34 GMT",
    "_id": "574366c6e2638a001f430115",
    "_etag": "9906d3a8584f0fabbd96451013b38d20fff5f5d3"
}`
	server := getTestServer(200, expectedResp)
	defer server.Close()

	r := Request{
		Host: server.URL,
		Auth: &Credentials{
			AdminToken: "dgdfg",
		},
	}

	ct := &Container{
		Name: "mycontainer",
	}

	err := r.Get(ct)
	if err != nil {
		log.Fatal(err)
	}

	expect(t, ct.Name, "mycontainer")
	expect(t, ct.Branch, "master")
	expect(t, ct.Registry, "docker.io")
	expect(t, ct.GithubHookID, 5)
	expect(t, ct.ID, "574366c6e2638a001f430115")

}

// TestPostContainer tests that when posting a container the server receives data that is formed
// correctly and that given a proper response the container will know its id and etag
func TestPostContainer(t *testing.T) {
	expectedResp := `{
    "_updated": "Mon, 13 Jun 2016 16:21:25 GMT",
    "name": "mycontainer",
	"branch": "master",
	"registry": "docker.io",
    "_created": "Mon, 13 Jun 2016 16:21:25 GMT",
    "_status": "OK",
    "_id": "98734598723094857023985",
    "_etag": "qwertyuiop12345"
}`

	reqData := map[string]string{
		"content-type":  "application/json",
		"authorization": "token dfgdfg",
		"endpoint":      "/containers",
	}

	ct := &Container{
		Name: "mycontainer",
	}

	reqData["payload"] = string(ct.JSON())

	server := postTestServer(t, expectedResp, reqData)
	defer server.Close()

	r := Request{
		Host: server.URL,
		Auth: &Credentials{
			AdminToken: "dfgdfg",
		},
	}

	err := r.Post(ct)
	if err != nil {
		log.Fatal(err)
	}

	expect(t, ct.Etag, "qwertyuiop12345")
	expect(t, ct.ID, "98734598723094857023985")
	expect(t, ct.Name, "mycontainer")
	expect(t, ct.Branch, "master")

}
