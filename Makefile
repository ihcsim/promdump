SHELL ?= /bin/bash

BUILD_OS ?= linux
BUILD_ARCH ?= amd64

VERSION = $(shell git describe --abbrev=0)

BASE_DIR = $(shell pwd)
TARGET_DIR = $(BASE_DIR)/target
TARGET_BIN_DIR = $(TARGET_DIR)/bin
TARGET_DIST_DIR = $(TARGET_DIR)/dist
TARGET_RELEASES_DIR = $(TARGET_DIR)/releases/$(VERSION)
TARGET_PLUGINS_DIR = $(TARGET_RELEASES_DIR)/plugins

all: test lint build dist

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
	CGO_ENABLED=0 GOOS="$(BUILD_OS)" GOARCH="$(BUILD_ARCH)" go build -o "$(TARGET_BIN_DIR)/promdump" ./core/cmd
	shasum -a256 "$(TARGET_BIN_DIR)/promdump"  | awk '{print $$1}' > "$(TARGET_BIN_DIR)/promdump.sha256"

.PHONY: cli
cli:
	CGO_ENABLED=0 GOOS="$(BUILD_OS)" GOARCH="$(BUILD_ARCH)" go build -ldflags="-X 'main.Version=$(VERSION)'" -o "$(TARGET_BIN_DIR)/cli-$(BUILD_OS)-$(BUILD_ARCH)-$(VERSION)" ./cli/cmd ;\
	shasum -a256 "$(TARGET_BIN_DIR)/cli-$(BUILD_OS)-$(BUILD_ARCH)-$(VERSION)"  | awk '{print $$1}' > "$(TARGET_BIN_DIR)/cli-$(BUILD_OS)-$(BUILD_ARCH)-$(VERSION).sha256"

.PHONY: dist
dist:
	rm -rf "$(TARGET_DIST_DIR)"
	mkdir -p "$(TARGET_DIST_DIR)"
	tar -C "$(TARGET_BIN_DIR)" -czvf "$(TARGET_DIST_DIR)/promdump-$(VERSION).tar.gz" promdump
	shasum -a256 "$(TARGET_DIST_DIR)/promdump-$(VERSION).tar.gz"  | awk '{print $$1}' > "$(TARGET_DIST_DIR)/promdump-$(VERSION).tar.gz.sha256"
	gsutil cp "$(TARGET_DIST_DIR)/promdump-$(VERSION).tar.gz" "$(TARGET_DIST_DIR)/promdump-$(VERSION).tar.gz.sha256" gs://promdump
	sleep 5
	gsutil acl ch -u AllUsers:R gs://promdump/promdump-$(VERSION).tar.gz gs://promdump/promdump-$(VERSION).tar.gz.sha256

.PHONY: plugins
plugins:
	rm -rf "$(TARGET_PLUGINS_DIR)" ;\
	mkdir -p "$(TARGET_PLUGINS_DIR)" ;\
	arch=( amd64 386 );\
	goos=( linux darwin windows ) ;\
	for arch in "$${arch[@]}" ; do \
		for os in "$${goos[@]}" ; do \
			cp "$(TARGET_RELEASES_DIR)/cli-$${os}-$${arch}-$(VERSION)" "$(TARGET_PLUGINS_DIR)/kubectl-promdump" ;\
			tar -C "$(TARGET_PLUGINS_DIR)" -czvf "$(TARGET_PLUGINS_DIR)/kubectl-promdump-$${os}-$${arch}-$(VERSION).tar.gz" kubectl-promdump ;\
			rm "$(TARGET_PLUGINS_DIR)/kubectl-promdump"
		done ;\
	done ;\

.PHONY: release
release: test
	rm -rf "$(TARGET_RELEASES_DIR)" ;\
	mkdir -p "$(TARGET_RELEASES_DIR)" ;\
	arch=( amd64 386 );\
	goos=( linux darwin windows ) ;\
	for arch in "$${arch[@]}" ; do \
		for os in "$${goos[@]}" ; do \
			$(MAKE) BUILD_OS="$${os}" BUILD_ARCH="$${arch}" TARGET_BIN_DIR=$(TARGET_RELEASES_DIR) cli ;\
		done ;\
	done ;\
	$(MAKE) TARGET_BIN_DIR=$(TARGET_RELEASES_DIR) core ;\
	$(MAKE) TARGET_BIN_DIR=$(TARGET_RELEASES_DIR) dist ;\

.PHONY: test
test/prometheus-repos:
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	helm repo add kube-state-metrics https://kubernetes.github.io/kube-state-metrics
	helm repo update

test/prometheus:
	helm install prometheus prometheus-community/prometheus

HACK_NAMESPACE ?= default
HACK_DATA_DIR ?= /data
.PHONY: hack
hack/deploy:
	pod="$$(kubectl get pods --namespace $(HACK_NAMESPACE) -l "app=prometheus,component=server" -o jsonpath="{.items[0].metadata.name}")" ;\
	kubectl -n "$(HACK_NAMESPACE)" cp -c prometheus-server "$(TARGET_BIN_DIR)/promdump" "$${pod}:$(HACK_DATA_DIR)" ;\
