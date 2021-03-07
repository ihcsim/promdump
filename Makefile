TARGET_NAMESPACE ?= prometheus
TARGET_POD ?= prometheus-7f7bd476fc-xl5ds
TARGET_DIR ?= /prometheus

BUILD_OS ?= linux
BUILD_ARCH ?= amd64

.PHONY: promdump
promdump:
	rm ./promdump
	CGO_ENABLED=0 GOOS="$(BUILD_OS)" GOARCH="$(BUILD_ARCH)" go build -o promdump ./cmd/dump

deploy: promdump
	target_pod="$$(kubectl -n "$(TARGET_NAMESPACE)" get po -oname | awk -F'/' '{print $$2}')" ;\
	kubectl -n "$(TARGET_NAMESPACE)" cp promdump "$${target_pod}:$(TARGET_DIR)"

test:
	rm -rf target
	mkdir -p target
	target_pod="$$(kubectl -n "$(TARGET_NAMESPACE)" get po -oname | awk -F'/' '{print $$2}')" ;\
	dump_file="$$(kubectl -n "$(TARGET_NAMESPACE)" exec $${target_pod} -- "$(TARGET_DIR)/promdump")" ;\
	kubectl -n "$(TARGET_NAMESPACE)" cp "$${target_pod}:$${dump_file}" "target/blocks.tar.gz"
