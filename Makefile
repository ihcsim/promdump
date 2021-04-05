SHELL ?= /bin/bash
NAMESPACE ?= prometheus
REMOTE_DIR ?= /prometheus

BUILD_OS ?= linux
BUILD_ARCH ?= amd64

VERSION = $(shell git describe --abbrev=0)

BASE_DIR = $(shell pwd)
TARGET_DIR = $(BASE_DIR)/target
TARGET_BIN_DIR = $(TARGET_DIR)/bin
TARGET_RELEASE_DIR = $(TARGET_DIR)/$(VERSION)

dev: test lint build publish

build: prebuild core cli
prebuild:
	rm -rf $(TARGET_BIN_DIR)
	mkdir -p $(TARGET_BIN_DIR)

.PHONY: test
test: test-core test-cli
	go test ./...

test-%:
	go test ./$*/...

lint-%:
	cd ./$* && golangci-lint run

lint: lint-core lint-cli
	golangci-lint run

tidy: tidy-core tidy-cli test

tidy-%:
	cd ./$* && go mod tidy

.PHONY: core
core:
	CGO_ENABLED=0 GOOS="$(BUILD_OS)" GOARCH="$(BUILD_ARCH)" go build -o "$(TARGET_BIN_DIR)/promdump-$(VERSION)" ./core/cmd
	shasum -a256 "$(TARGET_BIN_DIR)/promdump-$(BUILD_OS)-$(BUILD_ARCH)-$(VERSION)"  | awk '{print $$1}' > "$(TARGET_BIN_DIR)/promdump-$(VERSION).sha256"

.PHONY: cli
cli:
	CGO_ENABLED=0 GOOS="$(BUILD_OS)" GOARCH="$(BUILD_ARCH)" go build -ldflags="-X 'main.Version=$(VERSION)'" -o "$(TARGET_BIN_DIR)/cli-$(BUILD_OS)-$(BUILD_ARCH)-$(VERSION)" ./cli/cmd ;\
	shasum -a256 "$(TARGET_BIN_DIR)/cli-$(BUILD_OS)-$(BUILD_ARCH)-$(VERSION)"  | awk '{print $$1}' > "$(TARGET_BIN_DIR)/cli-$(BUILD_OS)-$(BUILD_ARCH)-$(VERSION).sha256"
	cp ./cli/cmd/promdump.yaml "$(TARGET_BIN_DIR)/"

publish:
	rm -f "$(TARGET_DIR)/promdump-$(VERSION).tar.gz" "$(TARGET_DIR)/promdump-$(VERSION).sha256"
	tar -C "$(TARGET_BIN_DIR)" -cvf "$(TARGET_DIR)/promdump-$(VERSION).tar.gz" promdump-$(VERSION)
	shasum -a256 "$(TARGET_DIR)/promdump-$(VERSION).tar.gz"  | awk '{print $$1}' > "$(TARGET_DIR)/promdump-$(VERSION).tar.gz.sha256"
	gsutil cp "$(TARGET_DIR)/promdump-$(VERSION).tar.gz" "$(TARGET_DIR)/promdump-$(VERSION).tar.gz.sha256" gs://promdump
	gsutil acl ch -u AllUsers:R gs://promdump/promdump-$(VERSION).tar.gz gs://promdump/promdump-$(VERSION).sha256

.PHONY: release
release: test
	rm -rf $(TARGET_RELEASE_DIR) ;\
	mkdir -p $(TARGET_RELEASE_DIR) ;\
	arch=( amd64 386 );\
	goos=( linux darwin windows ) ;\
	for arch in "$${arch[@]}" ; do \
		for os in "$${goos[@]}" ; do \
			$(MAKE) BUILD_OS="$${os}" BUILD_ARCH="$${arch}" TARGET_BIN_DIR=$(TARGET_RELEASE_DIR) cli ;\
		done ;\
	done ;\
	$(MAKE) TARGET_BIN_DIR=$(TARGET_RELEASE_DIR) core ;\
	$(MAKE) TARGET_BIN_DIR=$(TARGET_RELEASE_DIR) publish ;\
	cp ./cli/cmd/promdump.yaml "$(TARGET_RELEASE_DIR)/"

promdump_deploy: core
	target_pod="$$(kubectl -n "$(NAMESPACE)" get po -oname | awk -F'/' '{print $$2}')" ;\
	kubectl -n "$(NAMESPACE)" cp "$(TARGET_BIN_DIR)/promdump" "$${target_pod}:$(REMOTE_DIR)"

promdump_test:
	rm -rf $(TARGET_DIR)
	mkdir -p $(TARGET_DIR)
	target_pod="$$(kubectl -n "$(NAMESPACE)" get po -oname | awk -F'/' '{print $$2}')" ;\
	dump_file="$$(kubectl -n "$(NAMESPACE)" exec $${target_pod} -- "$(REMOTE_DIR)/promdump")" ;\
	kubectl -n "$(NAMESPACE)" cp "$${target_pod}:$${dump_file}" "target/blocks.tar.gz"
