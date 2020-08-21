NAME=cloudmonitor_exporter

ifndef VERSION
VERSION=0.0.0
endif

COMMIT=$(shell git rev-parse HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
BUILDER=$(shell whoami)@$(shell hostname)
BUILD_DATE=$(shell date '+%F-%T%z')

VERSION_PATH=github.com/prometheus/common/version
LDFLAGS=-ldflags "-X ${VERSION_PATH}.Version=${VERSION} \
                  -X ${VERSION_PATH}.Revision=${COMMIT} \
                  -X ${VERSION_PATH}.Branch=${BRANCH} \
                  -X ${VERSION_PATH}.BuildUser=${BUILDER} \
                  -X ${VERSION_PATH}.BuildDate=${BUILD_DATE}"

.PHONY: build clean rpm xbuild package

build:
	@mkdir -p bin/
	go build ${LDFLAGS} -o bin/${NAME} ${NAME}.go

xbuild: clean
	@mkdir -p build
	GOARCH=amd64 GOOS="linux" go build ${LDFLAGS} -o "build/$(NAME)_$(VERSION)_linux_amd64/$(NAME)"
	GOARCH=amd64 GOOS="darwin" go build ${LDFLAGS} -o "build/$(NAME)_$(VERSION)_darwin_amd64/$(NAME)"
	GOARCH=amd64 GOOS="windows" go build ${LDFLAGS} -o "build/$(NAME)_$(VERSION)_windows_amd64/$(NAME)"

package: xbuild
	$(eval FILES := $(shell ls build))
	@mkdir -p build/tgz
	for f in $(FILES); do \
		(cd $(shell pwd)/build && tar -zcvf tgz/$$f.tar.gz $$f); \
		echo $$f; \
	done

clean:
	@rm -rf bin/ && rm -rf build/

rpm: package
	@mkdir -p build/rpm
	docker run --rm -i -v $(shell pwd):/docker centos:7 /docker/package/rpm/build_rpm.sh ${VERSION}

docker: xbuild
	docker build --build-arg version=${VERSION} . -t cloudmonitor_exporter:${VERSION}