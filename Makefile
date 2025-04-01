BINARY_NAME = sbot

BIN_DIR = bin

##Заменить на свое имя конфига
CONFIG_NAME=localMake.yaml

CONFIG_PATH=$(shell pwd)/config/$(CONFIG_NAME)

MAIN_PACKAGE = ./cmd/bot/main.go

GOOS = linux
GOARCH = amd64

BUILD_FLAGS = -ldflags="-s -w"

.PHONY: all build run clean

all: build

build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

run: build
	env CONFIG_PATH=$(CONFIG_PATH) ./bin/$(BINARY_NAME)

clean:
	rm -f ./bin/$(BINARY_NAME)

$(BIN_DIR):
	mkdir -p $(BIN_DIR)
