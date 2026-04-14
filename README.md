# Notification Center

Track 1 provides the shared local runtime baseline for the Notification Center project.

## Included Components

- `frontend`
- `recipients`
- `sources`
- `notifications-service`
- `delivery`
- `postgres`
- `rabbitmq`
- `nginx`
- `traefik`

## Quick Start

1. Copy `.env.example` to `.env` if custom values are needed.
2. Run `make up`.
3. Open `http://localhost` for the Russian frontend stub.
4. Open `http://localhost:8088` for the Traefik dashboard.

## Scope

This baseline is structural only:

- shared Docker runtime
- gateway routing
- shared broker and database bootstrap
- health and readiness conventions
- CI and validation entrypoints

Business handlers and domain logic are intentionally not implemented in Track 1.
