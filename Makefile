PKG=gitlab.com/gitlab-org/gitaly
BUILD_DIR=$(shell pwd)
PKG_BUILD_DIR:=${BUILD_DIR}/_build/src/${PKG}
CMDS:=$(shell cd cmd && ls)

export GOPATH=${BUILD_DIR}/_build
export PATH:=${GOPATH}/bin:$(PATH)

.PHONY: ${BUILD_DIR}/_build

all: build

${BUILD_DIR}/_build:
	mkdir -p $@/src/${PKG}
	tar -cf - --exclude _build --exclude .git . | (cd $@/src/${PKG} && tar -xf -)
	touch $@

build:	${BUILD_DIR}/_build $(shell find . -name '*.go' -not -path './vendor/*')
	cd ${PKG_BUILD_DIR} && $(foreach cmd,${CMDS},go build ./cmd/${cmd} && ) true
	mv $(foreach cmd,${CMDS},${PKG_BUILD_DIR}/${cmd}) ${BUILD_DIR}/

test: ${BUILD_DIR}/_build
	cd ${PKG_BUILD_DIR} && go test ./...

clean:
	rm -rf ${BUILD_DIR}/_build
	rm -rf client/testdata
	rm -f ${CMDS}
