PREFIX := /usr/local
PKG := gitlab.com/gitlab-org/gitaly
BUILD_DIR := $(CURDIR)
TARGET_DIR := $(BUILD_DIR)/_build
TARGET_SETUP := $(TARGET_DIR)/.ok
BIN_BUILD_DIR := $(TARGET_DIR)/bin
PKG_BUILD_DIR := $(TARGET_DIR)/src/$(PKG)
TEST_REPO=internal/testhelper/testdata/data/gitlab-test.git
INSTALL_DEST_DIR := $(DESTDIR)$(PREFIX)/bin/

BUILDTIME = $(shell date -u +%Y%m%d.%H%M%S)
VERSION_PREFIXED = $(shell git describe)
VERSION = $(VERSION_PREFIXED:v%=%)
LDFLAGS = -ldflags '-X $(PKG)/internal/version.version=$(VERSION) -X $(PKG)/internal/version.buildtime=$(BUILDTIME)'

export GOPATH := $(TARGET_DIR)
export PATH := $(GOPATH)/bin:$(PATH)

# Returns a list of all non-vendored (local packages)
LOCAL_PACKAGES = $(shell cd "$(PKG_BUILD_DIR)" && GOPATH=$(GOPATH) PATH=$(PATH) govendor list -no-status +local)
COMMAND_PACKAGES = $(shell cd "$(PKG_BUILD_DIR)" && GOPATH=$(GOPATH) PATH=$(PATH) govendor list -no-status +local +p ./cmd/...)
COMMANDS = $(subst $(PKG)/cmd/,,$(COMMAND_PACKAGES))

# Developer Tools
GOVENDOR = $(BIN_BUILD_DIR)/govendor
GOLINT = $(BIN_BUILD_DIR)/golint

.NOTPARALLEL:

.PHONY: all
all: build

$(TARGET_SETUP):
	rm -rf $(TARGET_DIR)
	mkdir -p "$(dir $(PKG_BUILD_DIR))"
	ln -sf ../../../.. "$(PKG_BUILD_DIR)"
	mkdir -p "$(BIN_BUILD_DIR)"
	touch "$(TARGET_SETUP)"

.PHONY: build
build: $(TARGET_SETUP) $(GOVENDOR)
	go install $(LDFLAGS) $(COMMAND_PACKAGES)
	cp $(foreach cmd,$(COMMANDS),$(BIN_BUILD_DIR)/$(cmd)) $(BUILD_DIR)/

.PHONY: install
install: $(GOVENDOR) build
	mkdir -p $(INSTALL_DEST_DIR)
	cd $(BIN_BUILD_DIR) && install $(COMMANDS) $(INSTALL_DEST_DIR)

.PHONY: verify
verify: lint check-formatting govendor-status notice-up-to-date

.PHONY: check-formatting
check-formatting:
	go run _support/gofmt-all.go -n

.PHONY: govendor-status
govendor-status: $(TARGET_SETUP) $(GOVENDOR)
	cd $(PKG_BUILD_DIR) && govendor status

$(TEST_REPO):
	git clone --bare https://gitlab.com/gitlab-org/gitlab-test.git $@

.PHONY: test
test: $(TARGET_SETUP) $(TEST_REPO) $(GOVENDOR)
	go test $(LOCAL_PACKAGES)

.PHONY: lint
lint: $(GOLINT)
	go run _support/lint.go

.PHONY: package
package: build
	./_support/package/package $(COMMANDS)

.PHONY: notice
notice: $(TARGET_SETUP) $(GOVENDOR)
	cd $(PKG_BUILD_DIR) && govendor license -template _support/notice.template -o $(BUILD_DIR)/NOTICE

.PHONY: notice-up-to-date
notice-up-to-date: $(TARGET_SETUP) $(GOVENDOR)
	@(cd $(PKG_BUILD_DIR) && govendor license -template _support/notice.template | cmp - NOTICE) || (echo >&2 "NOTICE requires update: 'make notice'" && false)

.PHONY: clean
clean:
	rm -rf $(TARGET_DIR) $(TEST_REPO)

.PHONY: format
format:
	@go run _support/gofmt-all.go -f

# Install govendor
$(GOVENDOR): $(TARGET_SETUP)
	@go run _support/go-get-if-missing.go govendor github.com/kardianos/govendor

# Install golint
$(GOLINT): $(TARGET_SETUP)
	@go run _support/go-get-if-missing.go golint github.com/golang/lint/golint
