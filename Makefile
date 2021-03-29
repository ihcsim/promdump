TARGET_NAMESPACE ?= prometheus
TARGET_DIR ?= /prometheus

BUILD_OS ?= linux
BUILD_ARCH ?= amd64

BASE_DIR = "$(shell pwd)"
TARGET_DIR = "$(BASE_DIR)"/target
TARGET_BIN_DIR = "$(TARGET_DIR)"/bin

all: test lint build

build: prebuild core streaming cli
prebuild:
	rm -rf ./target/bin
	mkdir -p ./target/bin

.PHONY: test
test: test-core test-streaming test-cli
	go test -race ./...

test-%:
	cd ./$* && go test -race ./...

lint-%:
	cd ./$* && golangci-lint run

lint: lint-core lint-streaming lint-cli

tidy: tidy-core tidy-streaming tidy-cli test

tidy-%:
	cd ./$* && go mod tidy

.PHONY: core
core: test-core
	cd core ;\
	CGO_ENABLED=0 GOOS="$(BUILD_OS)" GOARCH="$(BUILD_ARCH)" go build -o "$(TARGET_BIN_DIR)"/promdump ./cmd

.PHONY: streaming
streaming: test-streaming
	cd streaming ;\
	CGO_ENABLED=0 GOOS="$(BUILD_OS)" GOARCH="$(BUILD_ARCH)" go build -o "$(TARGET_BIN_DIR)"/promdump-streaming ./cmd

.PHONY: cli
cli: test-cli
	cd cli ;\
	git_commit="$$(git rev-parse --short HEAD)" ;\
	CGO_ENABLED=0 GOOS="$(BUILD_OS)" GOARCH="$(BUILD_ARCH)" go build -ldflags="-X 'main.Version=$${git_commit}'" -o "$(TARGET_BIN_DIR)"/promdump-cli ./cmd

promdump_deploy: core
	target_pod="$$(kubectl -n "$(TARGET_NAMESPACE)" get po -oname | awk -F'/' '{print $$2}')" ;\
	kubectl -n "$(TARGET_NAMESPACE)" cp promdump "$${target_pod}:$(TARGET_DIR)"

promdump_test:
	rm -rf target
	mkdir -p target
	target_pod="$$(kubectl -n "$(TARGET_NAMESPACE)" get po -oname | awk -F'/' '{print $$2}')" ;\
	dump_file="$$(kubectl -n "$(TARGET_NAMESPACE)" exec $${target_pod} -- "$(TARGET_DIR)/promdump")" ;\
	kubectl -n "$(TARGET_NAMESPACE)" cp "$${target_pod}:$${dump_file}" "target/blocks.tar.gz"
