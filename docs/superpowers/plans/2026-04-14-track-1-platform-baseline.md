# Track 1 Platform Baseline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a runnable Track 1 platform baseline with one root Docker Compose stack, minimal service stubs, shared infra wiring, and root validation entrypoints.

**Architecture:** The baseline keeps business ownership frozen and implements only the shared runtime spine. Each backend service is a small HTTP stub exposing `/health` and `/ready`, `delivery` is a non-business worker stub, the frontend is a static Russian placeholder app, and NGINX routes frozen public paths while blocking `/internal`. RabbitMQ and PostgreSQL are bootstrapped structurally, and root commands validate compose, contracts, and basic layout without implementing domain logic.

**Tech Stack:** Docker Compose, Go, Gin, React + NGINX static serving, Traefik, PostgreSQL, RabbitMQ, Prometheus, GitHub Actions

---

### Task 1: Create root runtime and infra skeleton

**Files:**
- Create: `docker-compose.yml`
- Create: `.env.example`
- Create: `Makefile`
- Create: `infra/nginx/nginx.conf`
- Create: `infra/traefik/traefik.yml`
- Create: `infra/postgres/init/001-create-schemas.sql`
- Create: `infra/rabbitmq/rabbitmq.conf`
- Create: `infra/rabbitmq/definitions.json`
- Create: `infra/monitoring/prometheus.yml`

- [ ] Define the shared service names, ports, volumes, and healthchecks in `docker-compose.yml`.
- [ ] Add shared environment variable defaults in `.env.example`.
- [ ] Add root developer commands in `Makefile` for `up`, `down`, `build`, `test`, `lint`, and contract validation placeholders.
- [ ] Add NGINX route wiring for `/`, `/api/recipients/`, `/api/sources/`, `/api/notifications/`, and explicit `/internal/` deny behavior.
- [ ] Add Traefik static config for Docker provider and local dashboard entrypoint.
- [ ] Add PostgreSQL bootstrap SQL that creates only the frozen schemas: `recipients`, `sources`, `notifications`, `delivery`.
- [ ] Add RabbitMQ bootstrap config and definitions for main queue, retry queue, and dead-letter queue without changing payload semantics.
- [ ] Add minimal Prometheus scrape config for gateway and service health endpoints.

### Task 2: Add runnable backend service stubs

**Files:**
- Create: `recipients/go.mod`
- Create: `recipients/cmd/api/main.go`
- Create: `recipients/Dockerfile`
- Create: `sources/go.mod`
- Create: `sources/cmd/api/main.go`
- Create: `sources/Dockerfile`
- Create: `notifications-service/go.mod`
- Create: `notifications-service/cmd/api/main.go`
- Create: `notifications-service/Dockerfile`
- Create: `delivery/go.mod`
- Create: `delivery/cmd/consumer/main.go`
- Create: `delivery/Dockerfile`

- [ ] Add a tiny HTTP server for `recipients`, `sources`, and `notifications-service` with `/health` and `/ready` plus a placeholder root response.
- [ ] Add a tiny worker process for `delivery` with HTTP health endpoints and a long-running placeholder loop.
- [ ] Keep logging and messages structural only, with no business handlers or contract implementations.
- [ ] Add Dockerfiles that build and run each Go stub in containers.

### Task 3: Add runnable frontend placeholder and docs

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/nginx.conf`
- Create: `frontend/public/index.html`
- Create: `frontend/Dockerfile`
- Create: `README.md`
- Create: `docs/platform-baseline.md`

- [ ] Add a minimal Russian-only static frontend placeholder page describing Track 1 baseline status.
- [ ] Add a frontend Dockerfile using NGINX to serve the static app.
- [ ] Document environment conventions, health/readiness rules, and startup commands in root docs.

### Task 4: Add repo validation and CI skeleton

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `scripts/validate-contracts.sh`
- Create: `scripts/test-platform.sh`
- Create: `scripts/lint-platform.sh`

- [ ] Add a CI workflow that runs root lint, contract validation placeholder, and structural tests.
- [ ] Add shell scripts for contract validation placeholder, lint placeholder, and compose structure checks.
- [ ] Run the structural checks locally and fix any wiring errors.
