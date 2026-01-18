BINARY_NAME = sbot

BIN_DIR = bin

VERSION    := $(shell git describe --tags --dirty --always)
COMMIT     := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

##Заменить на свое имя конфига
CONFIG_NAME=local.yaml

CONFIG_PATH=$(shell pwd)/config/$(CONFIG_NAME)

MAIN_PACKAGE = ./cmd/bot/main.go

GOOS = linux
GOARCH = amd64

BUILD_FLAGS = -ldflags "\
			  -s -w \
			  -X main.Version=$(VERSION) \
			  -X main.Commit=$(COMMIT) \
			  -X main.BuildTime=$(BUILD_TIME)"

.PHONY: all build run clean

all: build

build:
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

run: build
	GOOS=$(GOOS) GOARCH=$(GOARCH) go run $(BUILD_FLAGS) $(MAIN_PACKAGE) --config=$(CONFIG_PATH)

clean:
	rm -f ./bin/$(BINARY_NAME)

$(BIN_DIR):
	mkdir -p $(BIN_DIR)
