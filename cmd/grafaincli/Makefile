.PHONY: all build test image tf protoc clean dist archive

BUILD_HASH ?= manual
BUILD_VERSION ?=$(shell git describe --tags --abbrev=9 | sed 's/^v//')
DIST_SRC=$(shell pwd)
LDFLAGS=-extldflags=-static -X main.BUILD_VERSION=${BUILD_HASH} -s -w
BUILDOUT ?= grafaincli
IMAGE_NAME = "alpetest/grafaincli:${BUILD_HASH}"

all: dist

dist: clean build image archive

clean:
	rm -f ${BUILDOUT}

build:
	GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build -a \
	        -gcflags=all=-trimpath=${DIST_SRC} \
	        -asmflags=all=-trimpath=${DIST_SRC} \
	        -mod=readonly -tags "netgo" \
	        -ldflags="${LDFLAGS}" \
	        -o ${BUILDOUT} .

archive:
	tar -czvf ${BUILDOUT}-${BUILD_VERSION}-linux-amd64.tar.gz ${BUILDOUT}
	rm -f ${BUILDOUT}-${BUILD_VERSION}-linux-amd64.sha256
	sha256sum ${BUILDOUT}-${BUILD_VERSION}-linux-amd64.tar.gz | awk '{print $$1}' > ${BUILDOUT}-${BUILD_VERSION}-linux-amd64.sha256

image:
	docker build --pull -t $(IMAGE_NAME) .

install:
	go install $(BUILD_FLAGS) .

# Test fast
tf:
	go test -short ./...
