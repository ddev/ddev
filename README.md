#DRUD CLI

[Full CLI docs](docs/drud.md)

## Setup

```shell
mkdir -p ~/goworkspace/{bin,src,pkg}
mkdir ~/goworkspace/src/github.com/drud
export GOPATH=$HOME/goworkspace
export PATH=$PATH:$GOPATH/bin
mkdir -p $GOPATH/src/github.com/drud
cd $GOPATH/src/github.com/drud && git clone git@github.com:drud/bootstrap.git
cd bootstrap
make osxbin
```

## Building Binary

You can build the binary for osx by running

```shell
make osxbin
```

And for linux with

```shell
make linuxbin
```

## Test runs for CLI

```
go test -timeout 20m -v ./cmd
```

## Test runs for integration (hosting)

To build for local testing you can build a dockerhub image with:
```shell
make canary
```


This will create a drud/drud container tagged with the current branch. You can then run tests against a working cluster by setting environment variables and running:

* CLUSTER_DOMAIN should be set to the domain in use. For example, Brad uses unsalted.pw
* DRUDAPI_PROTOCOL should be http or https
* GITHUB_TOKEN is the authorizing token for the github.com user. used to create a vault token.
* CIRCLE_BRANCH is the branch built with "make canary" above


`docker run -e "GITHUB_TOKEN=$GITHUB_TOKEN" -e "CLUSTER_DOMAIN=$CLUSTER_DOMAIN" -e "DRUDAPI_PROTOCOL=$DRUDAPI_PROTOCOL" -it drud/drud:$CIRCLE_BRANCH go test -timeout 20m -v ./integration`

## Local testing

from the bootstrap/cli directory.

```
go test -v ./cmd
```

To skip tests that require a docker-compose environment use this `export SKIP_COMPOSE_TESTS=true`
