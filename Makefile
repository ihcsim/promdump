TARGET_NAMESPACE ?= prometheus
TARGET_POD ?= prometheus-7f7bd476fc-xl5ds
TARGET_DIR ?= /prometheus

PROMDUMP_BIN ?= promdump

BUILD_OS ?= linux
BUILD_ARCH ?= amd64

.PHONY:
build:
	CGO_ENABLED=0 GOOS="$(BUILD_OS)" GOARCH="$(BUILD_ARCH)" go build -o "$(PROMDUMP_BIN)" .

deploy: build
	target_pod="$(shell kubectl -n "${TARGET_NAMESPACE}" get po -oname | awk -F'/' '{print $$2}')" ;\
	kubectl -n "$(TARGET_NAMESPACE)" cp promdump "$${target_pod}:$(TARGET_DIR)"

dump:
	rm -rf target
	mkdir -p target
	target_pod="$(shell kubectl -n "${TARGET_NAMESPACE}" get po -oname | awk -F'/' '{print $$2}')" ;\
	kubectl -n "$(TARGET_NAMESPACE)" exec "$${target_pod}" -- "$(TARGET_DIR)/promdump"
