SHELL := /bin/bash
FLUTTER ?= flutter
DART ?= dart
CLANG_FORMAT ?= clang-format
CACHE_DIR := $(CURDIR)/.cache
export GOCACHE := $(CACHE_DIR)/go-build
export GOMODCACHE := $(CACHE_DIR)/go-mod
COMPOSE := docker compose --env-file .env -f infra/compose/compose.yaml
CPP_FILES := $(shell find engine/rtc -type f \( -name '*.cpp' -o -name '*.hpp' \))

.PHONY: help env api-run api-build api-test api-lint api-migrate api-integration native-build native-test native-format native-format-check mobile-deps mobile-check mobile-apk format check infra-config infra-up infra-up-cache infra-up-turn infra-check infra-check-cache infra-check-turn infra-down

help:
	@echo 'See README.md for setup. Main targets: env, check, api-migrate, api-integration, api-run, mobile-apk, infra-up, infra-down.'

env:
	python3 infra/scripts/init_env.py

api-run:
	python3 infra/scripts/api_local.py run

api-migrate:
	python3 infra/scripts/api_local.py migrate $(or $(ARGS),up)

api-integration:
	python3 infra/scripts/api_local.py integration

api-build:
	mkdir -p .build/api
	cd services/api && go build -trimpath -o ../../.build/api/connect-api ./cmd/api
	cd services/api && go build -trimpath -o ../../.build/api/connect-migrate ./cmd/migrate

api-test:
	cd services/api && go test -race -count=1 ./...

api-lint:
	@test -z "$$(cd services/api && gofmt -l .)" || (echo 'Run make format'; exit 1)
	cd services/api && go vet ./...

native-build:
	cmake -S engine/rtc -B .build/rtc -DCMAKE_BUILD_TYPE=Debug -DBUILD_TESTING=ON
	cmake --build .build/rtc --parallel

native-test: native-build
	ctest --test-dir .build/rtc --output-on-failure

native-format:
	$(CLANG_FORMAT) -i $(CPP_FILES)

native-format-check:
	$(CLANG_FORMAT) --dry-run --Werror $(CPP_FILES)

mobile-deps:
	cd apps/mobile && $(FLUTTER) pub get --enforce-lockfile

mobile-check: mobile-deps
	cd apps/mobile && $(DART) format --output=none --set-exit-if-changed lib test
	cd apps/mobile && $(FLUTTER) analyze --no-pub --fatal-infos
	cd apps/mobile && $(FLUTTER) test --no-pub

mobile-apk: mobile-deps
	cd apps/mobile && $(FLUTTER) build apk --debug --no-pub

format: native-format
	cd services/api && gofmt -w .
	cd apps/mobile && $(DART) format lib test

check: api-lint api-test api-build native-format-check native-test mobile-check

infra-config:
	$(COMPOSE) --profile cache --profile turn config --quiet

infra-up:
	$(COMPOSE) up -d --wait postgres

infra-up-cache:
	$(COMPOSE) --profile cache up -d --wait postgres redis

infra-up-turn:
	@test -f .build/coturn/turnserver.conf || (echo 'Run make env'; exit 1)
	$(COMPOSE) --profile turn up -d coturn

infra-check:
	bash infra/scripts/check_infra.sh

infra-check-cache:
	bash infra/scripts/check_infra.sh cache

infra-check-turn:
	python3 infra/scripts/check_turn.py

infra-down:
	$(COMPOSE) --profile cache --profile turn down
