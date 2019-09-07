.PHONY: all build test image tf clean dist lint deps

BUILD_VERSION ?= manual
DIST_SRC=$(pwd)/$(go list -m)
LDFLAGS=-extldflags=-static -X main.version=${BUILD_VERSION} -s -w
BUILDOUT ?= grafain
IMAGE_NAME = "alpetest/grafain:${BUILD_VERSION}"

all: deps dist

dist: clean lint test build image

clean:
	rm -f ${BUILDOUT}

build:
	GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build -a \
	        -gcflags=all=-trimpath=${DIST_SRC} \
	        -asmflags=all=-trimpath=${DIST_SRC} \
	        -mod=readonly -tags "netgo" \
	        -ldflags="${LDFLAGS}" \
	        -o $(BUILDOUT) ./cmd/grafain

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
