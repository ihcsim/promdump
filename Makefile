TARGET_NAMESPACE ?= prometheus
TARGET_DIR ?= /prometheus

BUILD_OS ?= linux
BUILD_ARCH ?= amd64

all: test build

build: promdump server

test:
	go test -race ./...

.PHONY: server
server:
	rm ./server
	CGO_ENABLED=0 GOOS="$(BUILD_OS)" GOARCH="$(BUILD_ARCH)" go build -o server ./cmd/server

.PHONY: promdump
promdump:
	rm ./promdump
	CGO_ENABLED=0 GOOS="$(BUILD_OS)" GOARCH="$(BUILD_ARCH)" go build -o promdump ./cmd/dump

promdump_deploy: promdump
	target_pod="$$(kubectl -n "$(TARGET_NAMESPACE)" get po -oname | awk -F'/' '{print $$2}')" ;\
	kubectl -n "$(TARGET_NAMESPACE)" cp promdump "$${target_pod}:$(TARGET_DIR)"

promdump_test:
	rm -rf target
	mkdir -p target
	target_pod="$$(kubectl -n "$(TARGET_NAMESPACE)" get po -oname | awk -F'/' '{print $$2}')" ;\
	dump_file="$$(kubectl -n "$(TARGET_NAMESPACE)" exec $${target_pod} -- "$(TARGET_DIR)/promdump")" ;\
	kubectl -n "$(TARGET_NAMESPACE)" cp "$${target_pod}:$${dump_file}" "target/blocks.tar.gz"
