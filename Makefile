.PHONY: all build test image tf clean dist lint protolint protofmt protodocs protoc import-spec

BUILD_VERSION ?= manual
DIST_SRC=$(pwd)/$(go list -m)
LDFLAGS=-extldflags=-static -X main.version=${BUILD_VERSION} -s -w
BUILDOUT ?= grafain
IMAGE_NAME = "alpetest/grafain:${BUILD_VERSION}"

# for dockerized prototool
USER := $(shell id -u):$(shell id -g)
DOCKER_BASE := docker run --rm --user=${USER} -v $(shell pwd):/work iov1/prototool:v0.2.2
PROTOTOOL := $(DOCKER_BASE) prototool
PROTOC := $(DOCKER_BASE) protoc
WEAVEDIR=$(shell go list -m -f '{{.Dir}}' github.com/iov-one/weave)


all:  dist

dist: clean lint
	cd cmd/grafaind && $(MAKE) dist
	cd cmd/grafaincli && $(MAKE) dist

clean:
	rm -f ${BUILDOUT}

build:
	GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build -a \
	        -gcflags=all=-trimpath=${DIST_SRC} \
	        -asmflags=all=-trimpath=${DIST_SRC} \
	        -mod=readonly -tags "netgo" \
	        -ldflags="${LDFLAGS}" \
	        -o $(BUILDOUT) ./cmd/grafaind

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

protolint:
	$(PROTOTOOL) lint

protofmt:
	$(PROTOTOOL) format -w

protodocs:
	./contrib/protodocs/build_protodocs.sh docs/proto

protoc: protolint protofmt
	$(PROTOTOOL) generate

import-spec:
	@rm -rf ./spec
	@mkdir -p spec/github.com/iov-one/weave
	@cp -r ${WEAVEDIR}/spec/gogo/* spec/github.com/iov-one/weave
	@chmod -R +w spec
