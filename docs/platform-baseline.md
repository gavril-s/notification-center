# Platform Baseline

This document describes the Track 1 shared runtime conventions.

## Health And Readiness

Every backend service exposes:

- `GET /health`
- `GET /ready`

NGINX and the frontend expose the same endpoints for container healthchecks.

## Public Routing

NGINX keeps the frozen public prefixes:

- `/` -> `frontend`
- `/api/recipients/*` -> `recipients`
- `/api/sources/*` -> `sources`
- `/api/notifications/*` -> `notifications-service`

`/internal/*` is not exposed publicly through the gateway.

## Environment Conventions

Shared runtime values live in `.env` and are documented in `.env.example`.

Current conventions:

- service port is passed by `PORT`
- service name is passed by `SERVICE_NAME`
- PostgreSQL uses one shared database container with logical schemas only
- RabbitMQ topology is provisioned centrally by Track 1

## Queue Topology

Track 1 provisions structural broker objects only:

- `notification.dispatch.v1`
- `notification.dispatch.v1.retry`
- `notification.dispatch.v1.dlq`

Payload semantics remain frozen by Track 0.

## Validation Commands

- `make compose-config`
- `make lint`
- `make validate-contracts`
- `make test`
