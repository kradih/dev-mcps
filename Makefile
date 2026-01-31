.PHONY: all build build-all test test-unit test-integration clean install

BINARY_DIR := bin
GO := go

all: build

build: build-all

build-all:
	$(GO) build -o $(BINARY_DIR)/all-server ./cmd/all
	$(GO) build -o $(BINARY_DIR)/filesystem-server ./cmd/filesystem
	$(GO) build -o $(BINARY_DIR)/command-server ./cmd/command
	$(GO) build -o $(BINARY_DIR)/environment-server ./cmd/environment
	$(GO) build -o $(BINARY_DIR)/git-server ./cmd/git
	$(GO) build -o $(BINARY_DIR)/process-server ./cmd/process
	$(GO) build -o $(BINARY_DIR)/web-server ./cmd/web

build-filesystem:
	$(GO) build -o $(BINARY_DIR)/filesystem-server ./cmd/filesystem

build-command:
	$(GO) build -o $(BINARY_DIR)/command-server ./cmd/command

build-environment:
	$(GO) build -o $(BINARY_DIR)/environment-server ./cmd/environment

build-git:
	$(GO) build -o $(BINARY_DIR)/git-server ./cmd/git

build-process:
	$(GO) build -o $(BINARY_DIR)/process-server ./cmd/process

build-web:
	$(GO) build -o $(BINARY_DIR)/web-server ./cmd/web

test: test-unit

test-unit:
	$(GO) test -v ./...

test-integration:
	$(GO) test -v -tags=integration ./tests/integration/...

test-coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html

install: build
	cp $(BINARY_DIR)/* /usr/local/bin/

install-local: build
	mkdir -p $(HOME)/.local/bin
	cp $(BINARY_DIR)/* $(HOME)/.local/bin/

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint: fmt vet
	@echo "Linting complete"

deps:
	$(GO) mod tidy
	$(GO) mod download

.DEFAULT_GOAL := build
