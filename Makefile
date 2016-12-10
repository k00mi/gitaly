PKG=gitlab.com/gitlab-org/git-access-daemon
BUILD_DIR=$(shell pwd)
CLIENT_BIN=git-daemon-client
SERVER_BIN=git-daemon-server

export GOPATH=${BUILD_DIR}/_build

.PHONY: ${BUILD_DIR}/_build

all: test build

build:
	go build -o ${SERVER_BIN} cmd/server/main.go
	go build -o ${CLIENT_BIN} cmd/client/main.go

${BUILD_DIR}/_build:
	mkdir -p $@/src/${PKG}
	tar -cf - --exclude _build --exclude .git . | (cd $@/src/${PKG} && tar -xf -)
	touch $@

test: ${BUILD_DIR}/_build
	cd ${BUILD_DIR}/_build/src/${PKG}/server && go test -v
	cd ${BUILD_DIR}/_build/src/${PKG}/client && go test -v

clean:
	rm -rf ${BUILD_DIR}/_build
	rm -rf client/testdata
	[ -f ${CLIENT_BIN} ] && rm ${CLIENT_BIN}
	[ -f ${SERVER_BIN} ] && rm ${SERVER_BIN}
