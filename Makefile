TAG = $(shell git rev-parse --abbrev-ref HEAD | tr -d '\n')
PREFIX = drud/drud
INTEGRATION_PREFIX = drud/drudintegration

glide:
	glide install
	@mkdir -p ./bin

osxbin: glide
	CGO_ENABLED=0 GOOS=darwin go build -a -installsuffix cgo -ldflags '-w' -o $(GOPATH)/bin/drud  ./main.go
	@cp -p $(GOPATH)/bin/drud ./bin

linuxbin: glide
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w' -o $(GOPATH)/bin/drud  ./main.go
	@cp -p $(GOPATH)/bin/drud ./bin

dev:
	docker build -t $(PREFIX):$(TAG) .
	docker run -v $(shell pwd)/bin:/go/bin -it $(PREFIX):$(TAG)

latest: dev
	docker tag $(PREFIX):$(TAG) $(PREFIX):latest

canary: dev
	docker push $(PREFIX):$(TAG)

all: latest canary
	docker push $(PREFIX):latest
