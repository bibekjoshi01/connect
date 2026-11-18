SHELL := /bin/bash
GO ?= go
GOVULNCHECK_VERSION := v1.10.0
ROOT_DIR := $(shell cd .. && pwd)
BUILD_DIR := $(ROOT_DIR)/.build/backend
export GOCACHE := $(ROOT_DIR)/.cache/go-build
export GOMODCACHE := $(ROOT_DIR)/.cache/go-mod
export CONNECT_API_TEST_BINARY := $(BUILD_DIR)/connect-api

.PHONY: help run build test lint check format migrate integration security

help:
	@echo 'Backend only: make build, test, lint, check, run, migrate [ARGS=up|down|status], integration, security.'
	@echo 'Integration uses its own disposable PostgreSQL; no root .env, Flutter or native library required.'

build:
	mkdir -p "$(BUILD_DIR)"
	$(GO) build -trimpath -o "$(BUILD_DIR)/connect-api" ./cmd/api
	$(GO) build -trimpath -o "$(BUILD_DIR)/connect-migrate" ./cmd/migrate

test:
	$(GO) test -race -count=1 ./...

lint:
	@test -z "$$(gofmt -l .)" || (echo 'Run make format in backend/'; exit 1)
	$(GO) vet ./...

check: lint test build

format:
	gofmt -w .

run:
	python3 scripts/dev.py run

migrate:
	python3 scripts/dev.py migrate $(or $(ARGS),up)

integration: build
	python3 scripts/dev.py integration

# Keep the scanner pinned and separate from offline build/test checks.
security:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...
