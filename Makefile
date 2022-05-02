SHELL ?= /bin/bash

BUILD_OS ?= linux
BUILD_ARCH ?= amd64

VERSION = $(shell git describe --abbrev=0)
GIT_COMMIT=$(shell git rev-parse --short HEAD)

BASE_DIR = $(shell pwd)
TARGET_DIR = $(BASE_DIR)/target
TARGET_BIN_DIR = $(TARGET_DIR)/bin
TARGET_RELEASE_DIR = $(TARGET_DIR)/releases/$(VERSION)
TARGET_PLUGINS_DIR = $(TARGET_RELEASE_DIR)/plugins

all: test lint build

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
	cd ./$* && golangci-lint run --timeout 5m

lint: lint-core lint-cli
	golangci-lint run

tidy: tidy-core tidy-cli test

tidy-%:
	cd ./$* && go mod tidy

.PHONY: core
core:
	CGO_ENABLED=0 GOOS="$(BUILD_OS)" GOARCH="$(BUILD_ARCH)" go build -o "$(TARGET_BIN_DIR)/promdump" ./core/cmd
	shasum -a256 "$(TARGET_BIN_DIR)/promdump"  | awk '{print $$1}' > "$(TARGET_BIN_DIR)/promdump.sha256"
	tar -C "$(TARGET_BIN_DIR)" -czvf "$(TARGET_BIN_DIR)/promdump.tar.gz" promdump
	cp "$(TARGET_BIN_DIR)/promdump.tar.gz" ./cli/cmd/

.PHONY: cli
cli:
	if [ "$(BUILD_OS)" = "windows" ]; then \
		extension=".exe" ;\
	fi && \
	CGO_ENABLED=0 GOOS="$(BUILD_OS)" GOARCH="$(BUILD_ARCH)" go build -ldflags="-X 'main.Version=$(VERSION)' -X 'main.Commit=$(GIT_COMMIT)'" -o "$(TARGET_BIN_DIR)/cli-$(BUILD_OS)-$(BUILD_ARCH)-$(VERSION)$${extension}" ./cli/cmd &&\
	shasum -a256 "$(TARGET_BIN_DIR)/cli-$(BUILD_OS)-$(BUILD_ARCH)-$(VERSION)"$${extension}  | awk '{print $$1}' > "$(TARGET_BIN_DIR)/cli-$(BUILD_OS)-$(BUILD_ARCH)-$(VERSION).sha256"

.PHONY: release
release:
	rm -rf "$(TARGET_RELEASE_DIR)" && \
	mkdir -p "$(TARGET_RELEASE_DIR)" && \
	for os in linux darwin windows ; do \
		$(MAKE) BUILD_OS="$${os}" BUILD_ARCH="amd64" TARGET_BIN_DIR="$(TARGET_RELEASE_DIR)" cli plugin ;\
	done && \
	$(MAKE) BUILD_OS="darwin" BUILD_ARCH="arm64" TARGET_BIN_DIR="$(TARGET_RELEASE_DIR)" cli plugin && \
	$(MAKE) TARGET_BIN_DIR="$(TARGET_RELEASE_DIR)" core

.PHONY: plugin
plugin:
	mkdir -p "$(TARGET_PLUGINS_DIR)" && \
	if [ "$(BUILD_OS)" = "windows" ]; then \
		extension=".exe" ;\
	fi && \
	cp LICENSE "$(TARGET_PLUGINS_DIR)" && \
	cp "$(TARGET_RELEASE_DIR)/cli-$(BUILD_OS)-$(BUILD_ARCH)-$(VERSION)$${extension}" "$(TARGET_PLUGINS_DIR)/kubectl-promdump$${extension}" && \
	tar -C "$(TARGET_PLUGINS_DIR)" -czvf "$(TARGET_PLUGINS_DIR)/kubectl-promdump-$(BUILD_OS)-$(BUILD_ARCH)-$(VERSION).tar.gz" kubectl-promdump$${extension} LICENSE && \
	rm "$(TARGET_PLUGINS_DIR)/kubectl-promdump$${extension}" && \
	shasum -a256 $(TARGET_PLUGINS_DIR)/kubectl-promdump-$(BUILD_OS)-$(BUILD_ARCH)-$(VERSION).tar.gz | awk '{print $$1}' > $(TARGET_PLUGINS_DIR)/kubectl-promdump-$(BUILD_OS)-$(BUILD_ARCH)-$(VERSION).tar.gz.sha256

.PHONY: hack/prometheus-repos
hack/prometheus-repos:
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	helm repo add kube-state-metrics https://kubernetes.github.io/kube-state-metrics
	helm repo update

.PHONY: hack/prometheus
hack/prometheus: hack/prometheus-repos
	helm install prometheus prometheus-community/prometheus

HACK_NAMESPACE ?= default
HACK_DATA_DIR ?= /data

.PHONY: hack/deploy
hack/deploy:
	pod="$$(kubectl get pods --namespace $(HACK_NAMESPACE) -l "app=prometheus,component=server" -o jsonpath="{.items[0].metadata.name}")" ;\
	kubectl -n "$(HACK_NAMESPACE)" cp -c prometheus-server "$(TARGET_BIN_DIR)/promdump" "$${pod}:$(HACK_DATA_DIR)" ;\
