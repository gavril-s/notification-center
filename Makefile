SHELL := /bin/sh
COMPOSE := $(shell if command -v docker-compose >/dev/null 2>&1; then printf 'docker-compose'; else printf 'docker compose'; fi)

.PHONY: up down build test lint validate-contracts compose-config smoke

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down -v

build:
	$(COMPOSE) build

test: smoke

smoke:
	sh scripts/test-platform.sh

lint:
	sh scripts/lint-platform.sh

validate-contracts:
	sh scripts/validate-contracts.sh

compose-config:
	$(COMPOSE) config
