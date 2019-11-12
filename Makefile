.PHONY: all build test image tf clean dist lint protolint protofmt protodocs protoc import-spec install

# for dockerized prototool
USER := $(shell id -u):$(shell id -g)
DOCKER_BASE := docker run --rm --user=${USER} -v $(shell pwd):/work iov1/prototool:v0.2.2
PROTOTOOL := $(DOCKER_BASE) prototool
PROTOC := $(DOCKER_BASE) protoc
WEAVEDIR=$(shell go list -m -f '{{.Dir}}' github.com/iov-one/weave)


all:  dist

dist: clean lint test
	cd cmd/grafaind && $(MAKE) dist
	cd cmd/grafaincli && $(MAKE) dist
	cd cmd/grafainboard && $(MAKE) dist

install:
	cd cmd/grafaincli && $(MAKE) install

clean:
	rm -f ${BUILDOUT}

test:
	go test -v -race ./...

lint:
	go vet ./...

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
