PKG=gitlab.com/gitlab-org/gitaly
BUILD_DIR=$(shell pwd)
PKG_BUILD_DIR:=${BUILD_DIR}/_build/src/${PKG}
CMDS:=$(shell cd cmd && ls)

export GOPATH=${BUILD_DIR}/_build
export PATH:=${GOPATH}/bin:$(PATH)

.PHONY: all
all: build

.PHONY: ${BUILD_DIR}/_build
${BUILD_DIR}/_build:
	mkdir -p $@/src/${PKG}
	tar -cf - --exclude _build --exclude .git . | (cd $@/src/${PKG} && tar -xf -)
	touch $@

build:	clean-build ${BUILD_DIR}/_build $(shell find . -name '*.go' -not -path './vendor/*')
	cd ${PKG_BUILD_DIR} && $(foreach cmd,${CMDS},go build ./cmd/${cmd} && ) true
	mv $(foreach cmd,${CMDS},${PKG_BUILD_DIR}/${cmd}) ${BUILD_DIR}/

verify: lint check-formatting govendor-status

check-formatting:
	@if [ -n "$$(_support/gofmt-all -n)" ]; then echo please run \"make format\"; exit 1; fi

govendor-status: ${BUILD_DIR}/_build
	go run _support/go-get-if-missing.go govendor github.com/kardianos/govendor
	cd ${PKG_BUILD_DIR} && govendor status

test: clean-build ${BUILD_DIR}/_build verify
	cd ${PKG_BUILD_DIR} && go test ./...

lint: install-developer-tools
	@./_support/lint

package: build
	./_support/package/package ${CMDS}

clean:	clean-build
	rm -rf client/testdata
	rm -f $(foreach cmd,${CMDS},./${cmd})

clean-build:
	rm -rf ${BUILD_DIR}/_build

.PHONY: format
format:
	@_support/gofmt-all -f

.PHONY: install-developer-tools
install-developer-tools:
	go run _support/go-get-if-missing.go golint github.com/golang/lint/golint
