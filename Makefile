DATE=$(shell date -u '+%Y-%m-%d %H:%M:%S')
COMMIT=$(shell git log --format=%h -1)
VERSION=prometheus_helper.version=${TRAVIS_BUILD_NUMBER} ${COMMIT} ${DATE}
COMPILE_FLAGS=-ldflags="-X '${VERSION}'"

build:
	@go build ${COMPILE_FLAGS}

dep:
	@dep ensure

test:
	@go test .

all: clean test build
