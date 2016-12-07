PKG=gitlab.com/gitlab-org/git-access-daemon
BUILD_DIR = $(shell pwd)

export GOPATH:=$(GOPATH):${BUILD_DIR}/_build

all: test build

build:
	go build -o git-daemon-server cmd/server/main.go
	go build -o git-daemon-client cmd/client/main.go

${BUILD_DIR}/_build:
	mkdir -p $@/src/${PKG}
	tar -cf - --exclude _build --exclude .git . | (cd $@/src/${PKG} && tar -xf -)
	touch $@

test: ${BUILD_DIR}/_build
	cd ${BUILD_DIR}/_build/src/${PKG}/server && go test
	cd ${BUILD_DIR}/_build/src/${PKG}/client && go test

clean:
	rm -rf ${BUILD_DIR}/_build
	rm -rf client/testdata
