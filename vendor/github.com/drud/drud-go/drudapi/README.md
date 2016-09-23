# drudapi


Client CRUD example

```go
package main

import (
	"fmt"

	"github.com/drud/drud-go/drudapi"
)

func main() {

	r := drudapi.Request{
		Host: "https://drudapi.genesis.drud.io/v0.1",
		Auth: &drudapi.Credentials{
			AdminToken: "gittoken",
		},
	}

	c := &drudapi.Client{
		Name: "turtle",
	}

	fmt.Println("POsting")
	err := r.Post(c)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("%+v\n", c)
	}

	c.Phone = "123-123-1235"
	c.Email = "my@email.com"

	fmt.Println("Patching")
	err = r.Patch(c)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("%+v\n", c)
	}

	fmt.Println("Deleting")
	err = r.Delete(c)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Getting")
	err = r.Get(c)
	if err != nil {
		fmt.Println(err)
	}

}

```

expected output

```shell
POsting
&{Created:Thu, 02 Jun 2016 19:32:32 GMT Etag:8f054b6026600896d29f74dc1d1d138f8ed50867 ID:575089d0e2638a0017796b77 Updated:Thu, 02 Jun 2016 19:32:32 GMT Email: Name:turtle Phone:}
Patching
&{Created:Thu, 02 Jun 2016 19:32:32 GMT Etag:adbd8203e5ceaedeaeac5fff8f77be491e9c6b40 ID:575089d0e2638a0017796b77 Updated:Thu, 02 Jun 2016 19:32:32 GMT Email:my@email.com Name:turtle Phone:123-123-1235}
Deleting
Getting
404 Not Found: 404
```

Working with lists and filters

```go
package main

import (
	"fmt"

	"github.com/drud/drud-go/drudapi"
)

func main() {

	r := drudapi.Request{
		Host: "https://drudapi.genesis.drud.io/v0.1",
		Auth: &drudapi.Credentials{
			AdminToken: "gittoken",
		},
	}

	cl := &drudapi.ClientList{}
	r.Query = `where={"name":"1fee"}`

	err := r.Get(cl)
	if err != nil {
		fmt.Println(err)
	} else {
		for _, v := range cl.Items {
			fmt.Printf("%+v\n", v)
		}

	}

}

```
