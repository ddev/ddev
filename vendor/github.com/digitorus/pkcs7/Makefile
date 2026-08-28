all: vet test

test:
	go test -race -count=1 -covermode=atomic -coverprofile=coverage.out .

test-legacy:
	go test -race -tags=legacy -count=1 -covermode=atomic -coverprofile=coverage.out .

showcoverage: test
	go tool cover -html=coverage.out

vet:
	go vet .

golangci-lint:
	golangci-lint run

gettools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
