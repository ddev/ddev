# Goodhosts

Simple [hosts file](http://en.wikipedia.org/wiki/Hosts_%28file%29) (```/etc/hosts```) management in Go (golang).

<img src="http://www.hangthebankers.com/wp-content/uploads/2013/09/Masks-Rothschild-party1.jpg" width=400><br>
[A Surrealist Parisian Dinner Party chez Madame Rothschild, 1972](http://www.messynessychic.com/2013/08/27/a-surrealist-parisian-dinner-party-chez-madame-rothschild-1972/)

[![Build Status](https://travis-ci.org/lextoumbourou/goodhosts.svg)](https://travis-ci.org/lextoumbourou/goodhosts)

## Features

* List, add, remove and check hosts file entries from code or the command-line.
* Windows support.

## Command-Line Usage

### Installation

Download the [binary](http://github.com/lextoumbourou/goodhosts/releases/latest) and put it in your path.

```bash
$ wget https://github.com/lextoumbourou/goodhosts/releases/download/v2.0.0/goodhosts
$ chmod +x goodhosts
$ export PATH=$(pwd):$PATH
$ goodhosts --help
```

### List entries

```bash
$ goodhosts list
127.0.0.1 localhost
10.0.0.5 my-home-server xbmc-server
10.0.0.6 my-desktop
```

Add ```--all``` flag to include comments.

### Check for an entry

```bash
$ goodhosts check 127.0.0.1 facebook
```

### Add an entry

```bash
$ goodhosts add 127.0.0.1 facebook
```

Or *entries*.

```bash
$ goodhosts add 127.0.0.1 facebook twitter gmail
```

### Remove an entry

```bash
$ goodhosts rm 127.0.0.1 facebook
```

Or *entries*.

```bash
$ goodhosts rm 127.0.0.1 facebook twitter gmail
```

### More

```bash
$ goodhosts --help
```

## API Usage

### Installation

```bash
$ go get github.com/lextoumbourou/goodhosts
```

### List entries

```go
package main

import (
    "fmt"
    "github.com/lextoumbourou/goodhosts"
)

func main() {
    hosts := goodhosts.NewHosts()

    for _, line := range hosts.Lines {
        fmt.Println(line.Raw)
    }
}
```

### Check for an entry

```go
package main

import (
    "fmt"
    "github.com/lextoumbourou/goodhosts"
)

func main() {
    hosts := goodhosts.NewHosts()

    if hosts.Has("127.0.0.1", "facebook") {
        fmt.Println("Entry exists!")
        return
    }

    fmt.Println("Entry doesn't exist!")
}
```

### Add an entry

```go
package main

import (
    "fmt"
    "github.com/lextoumbourou/goodhosts"
)

func main() {
    hosts := goodhosts.NewHosts()

    // Note that nothing will be added to the hosts file until ``hosts.Flush`` is called.
    hosts.Add("127.0.0.1", "facebook.com", "twitter")

    if err := hosts.Flush(); err != nil {
        panic(err)
    }
}
```

### Remove an entry

```go
package main

import (
    "fmt"
    "github.com/lextoumbourou/goodhosts"
)

func main() {
    hosts := goodhosts.NewHosts()

    // Same deal, yo: call hosts.Flush() to make permanent.
    hosts.Remove("127.0.0.1", "facebook", "twitter")

    if err := hosts.Flush(); err != nil {
        panic(err)
    }
}
```

### [More](API.md)

## Changelog

### 2.0.0 (2015-05-04)

* Breaking API change.
* Add support for adding and removing multiple hosts.
* Added ``--all`` flag.
* Handle malformed IP addresses.

### 1.0.0 (2015-05-03)

- Initial release.

## License

[MIT](LICENSE)

<img src="http://static.messynessychic.com/wp-content/uploads/2013/08/rothschildparty2.jpg" width=400><br>
