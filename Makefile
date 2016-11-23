TAG = $(shell git rev-parse HEAD | tr -d '\n')
PREFIX = drud/drud

osxbin:
	CGO_ENABLED=0 GOOS=darwin go build -a -installsuffix cgo -ldflags '-w' -o $(GOPATH)/bin/drud  ./main.go
	@mkdir -p ./bin
	@cp -p $(GOPATH)/bin/drud ./bin

linuxbin:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w' -o $(GOPATH)/bin/drud  ./main.go
	@mkdir -p ./bin
	@cp -p $(GOPATH)/bin/drud ./bin

test:
	DRUD_BINARY_FULLPATH="$(PWD)/bin/drud" go test -timeout 20m -v ./cmd

dockerimage:
	docker build -t $(PREFIX):$(TAG) .
	docker run -v $(shell pwd)/bin:/go/bin -it $(PREFIX):$(TAG)

devcircle:
	# The remove flag helps with CircleCI
	# https://discuss.circleci.com/t/docker-error-removing-intermediate-container/70/23
	docker build --rm=false -t $(PREFIX):$(TAG) .
	docker run -v $(shell pwd)/bin:/go/bin -it $(PREFIX):$(TAG)

latest: dockerimage
	docker tag $(PREFIX):$(TAG) $(PREFIX):latest

canary: dockerimage
	docker push $(PREFIX):$(TAG)

circle: devcircle
	docker push $(PREFIX):$(TAG)

# Warning: Pushes "latest" to dockerhub
all: latest canary
	echo "Warning: this 'all' target pushes $(PREFIX):latest to hub.docker.com" >&2
	docker push $(PREFIX):latest

