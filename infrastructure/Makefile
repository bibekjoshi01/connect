SHELL := /bin/bash
ROOT_DIR := $(shell cd .. && pwd)
COMPOSE := docker compose --env-file "$(ROOT_DIR)/.env" -f compose/compose.yaml

.PHONY: help env config up up-cache up-turn check check-cache check-turn down

help:
	@echo 'Local dependencies: make env, config, up, up-cache, up-turn, check, check-cache, check-turn, down.'
	@echo 'PostgreSQL by default; Redis and coturn are optional. This does not build or deploy the API.'

env:
	python3 scripts/init_env.py

config:
	$(COMPOSE) --profile cache --profile turn config --quiet

up:
	$(COMPOSE) up -d --wait postgres

up-cache:
	$(COMPOSE) --profile cache up -d --wait postgres redis

up-turn:
	@test -f "$(ROOT_DIR)/.build/coturn/turnserver.conf" || (echo 'Run make env'; exit 1)
	$(COMPOSE) --profile turn up -d coturn

check:
	bash scripts/check_infra.sh

check-cache:
	bash scripts/check_infra.sh cache

check-turn:
	python3 scripts/check_turn.py

down:
	$(COMPOSE) --profile cache --profile turn down
