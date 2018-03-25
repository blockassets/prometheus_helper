build:
	@go build

dep:
	@dep ensure

test:
	@go test .

fmt:
	gofmt -s -w .

all: clean test build
