.PHONY: all build test image tf protoc clean dist archive

BUILD_VERSION ?= manual
DIST_SRC=$(pwd)/$(go list -m)
LDFLAGS=-extldflags=-static -X main.version=${BUILD_VERSION} -s -w
BUILDOUT ?= grafaind
IMAGE_NAME = "alpetest/grafain:${BUILD_VERSION}"

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
	tar -czvf ${BUILDOUT}-linux_amd64-${BUILD_VERSION}.tar.gz ${BUILDOUT}
	rm -f ${BUILDOUT}-linux_amd64-${BUILD_VERSION}.sha256
	sha256sum ${BUILDOUT}-linux_amd64-${BUILD_VERSION}.tar.gz | awk '{print $$1}' > ${BUILDOUT}-linux_amd64-${BUILD_VERSION}.sha256

image:
	docker build --pull -t $(IMAGE_NAME) .

install:
	go install $(BUILD_FLAGS) .

# Test fast
tf:
	go test -short ./...
