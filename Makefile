# Makefile

# Variables
BINARY_NAME=forge
PLUGIN_DIR=plugins/*
PLUGIN_FILES=$(wildcard $(PLUGIN_DIR)/*.go)
PLUGIN_OUTPUT=$(PLUGIN_FILES:.go=.so)

# Detect the operating system
OS := $(shell uname -s)

# Default target
all: build

# Build the main binary
build:
ifeq ($(OS),Linux)
	go build -o $(BINARY_NAME) cmd/main.go
else ifeq ($(OS),Darwin)
	go build -o $(BINARY_NAME) cmd/main.go
else ifeq ($(OS),Windows_NT)
	go build -o $(BINARY_NAME).exe cmd/main.go
endif

# Build all plugins
plugins: $(PLUGIN_OUTPUT)

# Build a single plugin
$(PLUGIN_DIR)/%.so: $(PLUGIN_DIR)/%.go
ifeq ($(OS),Linux)
	go build -buildmode=plugin -o $@ $<
else ifeq ($(OS),Darwin)
	go build -buildmode=plugin -o $@ $<
else ifeq ($(OS),Windows_NT)
	go build -buildmode=plugin -o $(@:.so=.dll) $<
endif

# Clean up build artifacts
clean:
ifeq ($(OS),Windows_NT)
	rm -f $(BINARY_NAME).exe $(PLUGIN_DIR)/*.dll
else
	rm -f $(BINARY_NAME) $(PLUGIN_OUTPUT)
endif

# Phony targets
.PHONY: all build plugins clean

#sudo mv forge /usr/local/bin/