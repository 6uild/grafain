.PHONY: all build test image tf clean dist lint deps

BUILD_VERSION ?= manual
BUILD_FLAGS := -a -ldflags '-extldflags "-static"  -X main.version=${BUILD_VERSION}'
BUILDOUT ?= grafain
IMAGE_NAME = "alpetest/grafain:${BUILD_VERSION}"

all: deps dist

dist: clean lint test build image

clean:
	rm -f ${BUILDOUT}

build:
	GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build $(BUILD_FLAGS) -o $(BUILDOUT) .

image:
	docker build --pull -t $(IMAGE_NAME) .

test:
	go test -race ./...

lint:
	go vet ./...

install:
	go install $(BUILD_FLAGS) .

# Test fast
tf:
	go test -short ./...

deps:
	dep ensure -vendor-only
