build:
	@go build

dep:
	@dep ensure

test:
	@go test .

all: clean test build
