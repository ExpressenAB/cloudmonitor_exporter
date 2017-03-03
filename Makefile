GOTOOLS=github.com/mitchellh/gox/...

NAME=cloudmonitor_exporter

VERSION=0.1.4
BUILD_TIME=`date '+%F%T%z'`

LDFLAGS=-ldflags "-X main.version=${VERSION} -X main.buildtime=${BUILD_TIME}"

all: tools build

build:
	@mkdir -p bin/
	go build ${LDFLAGS} -o bin/${NAME} ${NAME}.go

xbuild: clean
	@mkdir -p build
	gox \
		-os="linux" \
		-os="windows" \
		-os="darwin" \
		-arch="amd64" \
		${LDFLAGS} \
		-output="build/{{.Dir}}_$(VERSION)_{{.OS}}_{{.Arch}}/$(NAME)"

package: xbuild
	$(eval FILES := $(shell ls build))
	@mkdir -p build/tgz
	for f in $(FILES); do \
		(cd $(shell pwd)/build && tar -zcvf tgz/$$f.tar.gz $$f); \
		echo $$f; \
	done

clean:
	@rm -rf bin/ && rm -rf build/

tools:
	go get -u -v $(GOTOOLS)

rpm:
	@mkdir -p build/rpm
	docker run --rm -it -v $(shell pwd):/docker centos:7 /docker/package/rpm/build_rpm.sh ${VERSION}

ci: tools package rpm


.PHONY: all build clean ci tools