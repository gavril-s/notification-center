# Track 1 Specification: Platform and Integration

## Purpose

Track 1 builds the shared execution environment that all other tracks use. It owns Docker, compose, gateway wiring, broker wiring, CI skeleton, and shared observability and security scaffolding.

## Start Gate

Starts after Track 0 has frozen contracts and structure.

Tracks 2-5 start only after Track 1 has established the platform baseline.

## Owns

- Root `docker-compose.yml`
- Service Dockerfiles
- NGINX configuration
- Traefik configuration
- RabbitMQ topology and bootstrap
- PostgreSQL bootstrap config
- Root `Makefile`
- CI skeleton and validation entrypoints
- Shared healthcheck conventions
- Shared tracing, logging, and metrics bootstrap
- Shared secret-loading and environment conventions
- Smoke-test and release-check harnesses
- Backup and restore procedure
- Local environment runbooks

## Does Not Own

- Public business API payload design
- Internal business API payload design
- Business tables inside `recipients`, `sources`, `notifications`, or `delivery`
- Frontend business behavior

## Compose and Service Naming Freeze

The root compose file must start these services:

- `frontend`
- `recipients`
- `sources`
- `notifications-service`
- `delivery`
- `postgres`
- `rabbitmq`
- `nginx`
- `traefik`

Optional supporting services may be added only if they do not change business ownership boundaries.

## Gateway and Routing Rules

NGINX owns path-based routing.

- `/` -> `frontend`
- `/api/recipients/*` -> `recipients`
- `/api/sources/*` -> `sources`
- `/api/notifications/*` -> `notifications-service`
- `/internal/*` is not exposed publicly through the gateway

Track 1 may configure the gateway. It must not redefine business endpoints.

## Broker and Queue Rules

Track 1 owns broker topology, not payload semantics.

Required broker objects:

- primary notification dispatch queue
- retry queue or queues
- dead-letter queue

Track 1 must not alter the business fields of `notification.dispatch.v1`.

## PostgreSQL Rules

Track 1 owns the shared PostgreSQL container and bootstrap wiring.

Track 1 must not create or modify business tables except for infrastructure-only bootstrap helpers.

Track 1 must preserve the schema ownership map from Track 0:

- `recipients`
- `sources`
- `notifications`
- `delivery`

## Health and Runtime Rules

Every backend service must expose:

- `GET /health`
- `GET /ready`

Track 1 owns the requirement and platform verification. Service tracks own the handlers in their services.

## Shared Security and Observability Rules

Track 1 owns the shared scaffolding for:

- request ID propagation
- trace ID propagation
- gateway rate limiting
- environment and secret-loading conventions
- logging format conventions
- metrics export conventions

Track 1 does not own service-specific authorization logic or audit events.

## Files Owned by Track 1

- root `docker-compose.yml`
- root `Makefile`
- root CI workflows
- `infra/nginx/*`
- `infra/traefik/*`
- `infra/postgres/*`
- `infra/rabbitmq/*`
- `infra/monitoring/*`
- Dockerfiles in service roots, if not already created by service owners

## Database Ownership

Track 1 owns no business schema and no business table.

Track 1 may only:

- provision the PostgreSQL server
- provision logical schemas if needed
- validate migration execution flow

Track 1 must never define business columns for Track 2-4.

## Integration Contracts Consumed

Track 1 consumes only structural rules from Track 0.

It does not consume business data contracts except to validate them in CI.

## Deliverables

- One runnable root `docker-compose.yml`
- Shared local environment bootstrapping
- Base Dockerfiles
- CI skeleton for lint, test, build, and contract validation
- Shared gateway and broker wiring
- Shared operational scripts

## Change Control

Track 1 must not change:

- API payloads
- queue payload fields
- schema ownership
- domain model ownership

Any request to change those must be redirected to the owning track plus Track 0 conventions.

## Parallel Safety Criteria

Parallel work is safe when Track 1 has delivered:

- root compose and service names
- route wiring
- broker topology
- schema bootstrap
- common validation commands

At that point, Tracks 2-5 can implement against the same runtime environment.
