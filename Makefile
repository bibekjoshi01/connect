SHELL := /bin/bash
FLUTTER ?= flutter
DART ?= dart
CLANG_FORMAT ?= clang-format

.PHONY: help build check format env backend-run backend-build backend-test backend-lint backend-migrate backend-integration backend-check backend-security api-run api-build api-test api-lint api-migrate api-integration native-build native-test native-check native-format native-format-check mobile-run mobile-deps mobile-check mobile-apk infra-config infra-up infra-up-cache infra-up-turn infra-check infra-check-cache infra-check-turn infra-down

help:
	@echo 'Independent projects: mobile-app/, backend/, native-engine/, infrastructure/, contracts/.'
	@echo 'Root commands: build, check, backend-run, backend-integration, mobile-run, mobile-apk, infra-up, infra-down.'
	@echo 'Run make help inside a project for its own commands. Legacy api-* aliases still work.'

# Coordination only. Each component owns its implementation and build commands.
build: backend-build native-build mobile-apk
check: backend-check native-check mobile-check
format: native-format
	$(MAKE) -C backend format
	$(MAKE) -C mobile-app format

env:
	$(MAKE) -C infrastructure env

backend-run backend-build backend-test backend-lint backend-migrate backend-integration backend-check backend-security:
	$(MAKE) -C backend $(patsubst backend-%,%,$@)

# Keep existing developer commands working after the directory move.
api-run: backend-run
api-build: backend-build
api-test: backend-test
api-lint: backend-lint
api-migrate: backend-migrate
api-integration: backend-integration

native-build native-test native-check native-format native-format-check:
	@formatter="$(CLANG_FORMAT)"; \
	if [[ "$$formatter" == */* && "$$formatter" != /* ]]; then formatter="$(CURDIR)/$$formatter"; fi; \
	$(MAKE) -C native-engine $(patsubst native-%,%,$@) CLANG_FORMAT="$$formatter"

mobile-run mobile-deps mobile-check mobile-apk:
	$(MAKE) -C mobile-app $(patsubst mobile-%,%,$@)

infra-config infra-up infra-up-cache infra-up-turn infra-check infra-check-cache infra-check-turn infra-down:
	$(MAKE) -C infrastructure $(patsubst infra-%,%,$@)
