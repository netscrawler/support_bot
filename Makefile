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
				-linkmode external \
				-extldflags '-static' \
				-s -w \
				-X main.Version=$(VERSION) \
				-X main.Commit=$(COMMIT) \
				-X main.BuildTime=$(BUILD_TIME)"

.PHONY: all build run clean

all: build

build:
	CC=musl-gcc CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -mod=mod $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

docker-build:
	docker build \
		-f build.Dockerfile \
		-t support_bot-builder .

docker-run: docker-build
	docker create \
		--name support_bot-builder-tmp \
		support_bot-builder

	mkdir -p bin

	docker cp \
		support_bot-builder-tmp:/out/sbot \
		./bin/sbot

	docker rm support_bot-builder-tmp

run: build
	GOOS=$(GOOS) GOARCH=$(GOARCH) go run $(BUILD_FLAGS) $(MAIN_PACKAGE) --config=$(CONFIG_PATH)

clean:
	rm -f ./bin/$(BINARY_NAME)

deploy: build
	scp $(BIN_DIR)/$(BINARY_NAME) ${RELEASE_USER}@${RELEASE_SRV}:${RELEASE_PATH}/support_bot_new

$(BIN_DIR):
	mkdir -p $(BIN_DIR)
