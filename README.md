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
cd bootstrap && glide install
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

## Test runs

To build for local testing you should use.
```shell
make canary
```

This will create a drud/drudintegration container tagged with the current branch. You can then run tests against a working cluster by running the following:
```shell
 docker run -v /Users/beeradb/.drud-sanctuary-token:/root/.drud-sanctuary-token -e "GITHUB_TOKEN=$GITHUB_TOKEN" -e CLUSTER_DOMAIN=unsalted.pw -e DRUD_CLI_INT_NUM=2 -it drud/drudintegration:glide
 ```

