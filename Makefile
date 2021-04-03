NAMESPACE ?= prometheus
REMOTE_DIR ?= /prometheus

BUILD_OS ?= linux
BUILD_ARCH ?= amd64

BASE_DIR = $(shell pwd)
TARGET_DIR = $(BASE_DIR)/target
TARGET_BIN_DIR = $(TARGET_DIR)/bin

VERSION = $(shell git describe --abbrev=0)

all: test lint build

build: prebuild core cli
prebuild:
	rm -rf ./target/bin
	mkdir -p ./target/bin

.PHONY: test
test: test-core test-cli
	go test -race ./...

test-%:
	cd ./$* && go test -race ./...

lint-%:
	cd ./$* && golangci-lint run

lint: lint-core lint-cli
	golangci-lint run

tidy: tidy-core tidy-cli test

tidy-%:
	cd ./$* && go mod tidy

.PHONY: core
core: test-core
	cd core ;\
	CGO_ENABLED=0 GOOS="$(BUILD_OS)" GOARCH="$(BUILD_ARCH)" go build -o "$(TARGET_BIN_DIR)/promdump" ./cmd

.PHONY: cli
cli: test-cli
	cd cli ;\
	git_commit="$$(git rev-parse --short HEAD)" ;\
	CGO_ENABLED=0 GOOS="$(BUILD_OS)" GOARCH="$(BUILD_ARCH)" go build -ldflags="-X 'main.Version=$${git_commit}'" -o "$(TARGET_BIN_DIR)/promdump-cli" ./cmd

publish: build
	rm -f "$(TARGET_DIR)/promdump-$(VERSION).tar.gz" "$(TARGET_DIR)/promdump-$(VERSION).sha256"
	tar -C "$(TARGET_BIN_DIR)" -cvf "$(TARGET_DIR)/promdump-$(VERSION).tar.gz" promdump
	shasum -a256 "$(TARGET_DIR)/promdump-$(VERSION).tar.gz"  | awk '{print $$1}' > "$(TARGET_DIR)/promdump-$(VERSION).sha256"

promdump_deploy: core
	target_pod="$$(kubectl -n "$(NAMESPACE)" get po -oname | awk -F'/' '{print $$2}')" ;\
	kubectl -n "$(NAMESPACE)" cp "$(TARGET_BIN_DIR)/promdump" "$${target_pod}:$(REMOTE_DIR)"

promdump_test:
	rm -rf target
	mkdir -p target
	target_pod="$$(kubectl -n "$(NAMESPACE)" get po -oname | awk -F'/' '{print $$2}')" ;\
	dump_file="$$(kubectl -n "$(NAMESPACE)" exec $${target_pod} -- "$(REMOTE_DIR)/promdump")" ;\
	kubectl -n "$(NAMESPACE)" cp "$${target_pod}:$${dump_file}" "target/blocks.tar.gz"
