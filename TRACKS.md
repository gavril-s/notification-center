# Notification Center Parallel Work Tracks

## Purpose

This document defines how to split Notification Center implementation work across parallel tracks with minimal conflicts, clear ownership, and stable integration boundaries.

The recommended model is:

1. Complete a small shared foundation first.
2. Freeze contracts and architectural decisions.
3. Start parallel implementation tracks only after the foundation is stable.

Do not split the project purely by service from day one. The highest-risk dependencies are cross-cutting: contracts, auth, queue payloads, notification state ownership, unsubscribe semantics, and the Docker-based local environment.

## Mandatory Global Rules

These rules apply to every track without exception.

- The entire user interface must be in Russian only.
- Every backend service, the frontend, and infrastructure components must be deployed through Docker.
- The entire system must be started through one root `docker-compose.yml` that includes application containers and infrastructure components, including PostgreSQL and RabbitMQ.
- If there is any uncertainty, check `CLAUDE.md`, `notifications/PLAN.md`, and the diagrams in `ПП/Диаграммы/` before making a decision.
- Delivery uses mocks only. No real external providers are part of the project scope.
- Analytics are limited to per-sender analytics.
- PostgreSQL is one shared infrastructure component for the whole system.
- Service boundaries must still be preserved at the application level even with one shared PostgreSQL instance.
- Cross-service communication must happen through REST or RabbitMQ contracts, not ad hoc database coupling in business logic.
- `Notifications` owns canonical notification lifecycle state.
- `Delivery` owns transport-level attempts and retry behavior only.
- Guest recipients must be able to become registered users later without losing history, preferences, or unsubscribe state.

## Work Model

### Phase 0: Foundation First

Heavy parallelization must not start until the following foundation items are frozen.

- Architecture decisions from `PLAN.md`
- Public REST contracts
- Internal REST contracts
- RabbitMQ event schema
- JWT and machine-to-machine auth rules
- Notification state model
- Preference and unsubscribe precedence rules
- Guest contact claim model
- Outbox-based queue publishing model
- Root Docker and `docker-compose` bootstrap
- NGINX routing
- Basic CI and healthcheck conventions

This is the minimum shared base required to prevent track drift.

## Start Order

1. Start `Track 0: Foundation` first.
2. When the foundation exit criteria are met, start Tracks `1-5` in parallel.
3. Keep `Notifications` and `Delivery` in the same track until the end-to-end pipeline is stable.
4. Frontend should work from frozen API contracts and mocks, not backend assumptions.

## Track Overview

| Track | Name                     | Starts When          | Main Ownership                                                |
| ----- | ------------------------ | -------------------- | ------------------------------------------------------------- |
| 0     | Foundation               | immediately          | contracts, architecture freeze, base repo bootstrap           |
| 1     | Platform and Integration | after Track 0 freeze | Docker, compose, NGINX, RabbitMQ, CI, monitoring              |
| 2     | Recipients               | after Track 0 freeze | auth, users, contacts, preferences, unsubscribe               |
| 3     | Sources                  | after Track 0 freeze | senders, operators, templates, groups, campaigns, credentials |
| 4     | Notification Pipeline    | after Track 0 freeze | `Notifications` and `Delivery` together                       |
| 5     | Frontend                 | after Track 0 freeze | Russian-only UI for user and operator flows                   |

## Track 0: Foundation

### Goal

Create the shared base that all other tracks depend on.

### Owns

- Architecture decision freeze
- Public API contract definitions
- Internal API contract definitions
- RabbitMQ event schema
- Common error envelope
- Trace and correlation conventions
- Role and auth conventions
- Shared naming conventions
- Root repository skeleton

### Deliverables

- Finalized decisions for:
  - API gateway via NGINX
  - one shared PostgreSQL component
  - auth ownership in `Recipients`
  - machine-to-machine auth for external source systems
  - canonical notification state ownership in `Notifications`
  - transport-level attempt ownership in `Delivery`
  - guest contact claim model
  - sender-level and mailing-level unsubscribe model
  - recurrence split between `Sources` and `Notifications`
  - outbox-based publish flow
- Versioned contracts under:
  - `contracts/openapi/`
  - `contracts/events/`
- Shared conventions documented in the repo
- Initial repository structure matching `PLAN.md`

### Exit Criteria

- Another agent can implement against frozen contracts without inventing missing rules.
- Public, internal, and event contracts are stable enough for mocks and tests.
- Architecture ambiguities are resolved in writing.

### Must Not Do

- Do not implement deep business logic.
- Do not allow other tracks to redefine contracts after freeze without explicit coordination.

## Track 1: Platform and Integration

### Goal

Own the platform spine that all other tracks rely on.

### Owns

- Root `docker-compose.yml`
- Service Dockerfiles
- NGINX routing
- RabbitMQ topology
- PostgreSQL bootstrap
- Makefile targets
- CI checks
- health and readiness endpoints
- monitoring bootstrap
- local integration environment

### Deliverables

- One root `docker-compose.yml` that starts:
  - frontend
  - recipients
  - sources
  - notifications-service
  - delivery
  - PostgreSQL
  - RabbitMQ
  - NGINX
  - supporting monitoring components if included
- Base Dockerfiles for every service and frontend
- Local environment runnable with one command
- Basic CI for build, lint, contract validation, and tests
- Shared healthcheck conventions

### Dependencies

- Needs Track 0 contract and architecture freeze.
- Can run in parallel with Tracks 2-5 after the freeze.

### Coordination Contracts

- Must not invent service APIs.
- Must not change domain ownership.
- Must keep Docker and compose structure compatible with all service tracks.

### Exit Criteria

- Full stack boots locally through one `docker-compose`.
- Every service can be wired into the environment without structural rework.

## Track 2: Recipients

### Goal

Implement recipient identity and preference truth.

### Owns

- registration
- login
- JWT issuance
- users
- contacts
- preferences
- quiet hours
- sender unsubscribe
- mailing unsubscribe
- guest unsubscribe
- guest-to-user contact claim flow
- internal recipient resolution API

### Deliverables

- Public recipient APIs
- Internal recipient resolution API for `Notifications`
- Auth and role handling for interactive users
- Data model that keeps history anchored to `contact_id`
- Verified guest contact claim flow
- Tests for preference precedence and unsubscribe behavior

### Dependencies

- Needs Track 0 contracts and auth rules.
- Needs Track 1 environment for full integration testing.

### Coordination Contracts

- `Recipients` is the source of truth for:
  - user identity
  - interactive auth
  - contacts
  - preferences
  - unsubscribe rules
- Do not define sender templates, campaigns, or notification lifecycle logic here.
- Do not change JWT shape unilaterally.

### Exit Criteria

- `Notifications` can resolve eligible recipients without direct DB access.
- Guest recipients can later become registered users without losing relevant state.

## Track 3: Sources

### Goal

Implement sender-side management and source-system ownership.

### Owns

- senders
- source operators
- operator-to-sender assignment
- sender integration credentials
- templates
- contact groups
- campaigns
- recurrence rules
- template lookup APIs
- due campaign lookup APIs

### Deliverables

- Public operator APIs
- Internal lookup APIs for `Notifications`
- Machine credential ownership for external source systems
- Sender-scoped access model
- Template and campaign management flows
- Recurrence definitions

### Dependencies

- Needs Track 0 contract freeze.
- Needs Track 1 environment for integration.

### Coordination Contracts

- `Sources` is the source of truth for:
  - sender entities
  - operator access
  - source-system credentials
  - templates
  - groups
  - campaigns
  - recurrence definitions
- Do not implement actual notification sending here.
- Do not redefine notification statuses.

### Exit Criteria

- `Notifications` can validate source-system credentials and fetch all sender, template, and campaign data it needs.
- Operators can manage sender-side entities without cross-service hacks.

## Track 4: Notification Pipeline

### Goal

Implement the end-to-end notification execution pipeline.

### Scope Rule

This track owns both `Notifications` and `Delivery` together until the pipeline is stable.

### Owns

- `POST /api/notifications/send`
- sender credential validation flow
- template rendering
- recipient resolution integration
- unsubscribe URL generation
- preference evaluation
- notification persistence
- outbox creation
- RabbitMQ publish flow
- scheduler
- queue consumer
- retries
- DLQ handling
- transport-level attempt recording
- callback from `Delivery` to `Notifications`
- canonical status transitions
- history APIs
- per-sender analytics read models

### Deliverables

- Fully defined send pipeline from request to final mock delivery result
- Durable outbox-based publish path
- Idempotent consumer behavior
- Status callback loop
- User history APIs
- Per-sender analytics APIs
- Mock channel adapters for `email`, `sms`, and one messenger

### Dependencies

- Needs Track 0 contracts before implementation starts.
- Needs Track 2 recipient resolution API.
- Needs Track 3 template lookup, due campaign lookup, and credential resolution APIs.
- Needs Track 1 infrastructure.

### Coordination Contracts

- `Notifications` owns canonical notification state.
- `Delivery` owns attempt-level transport data only.
- Queue payload format is frozen by Track 0.
- Retry semantics and idempotency rules must not drift.
- Do not split `Notifications` and `Delivery` into separate tracks early.

### Exit Criteria

- A source-system request can become a queued notification.
- Queue messages are delivered to mock channels.
- Transport outcomes are recorded.
- Canonical notification statuses are updated correctly.
- History and per-sender analytics are queryable.

## Track 5: Frontend

### Goal

Implement the full Russian-only user and operator interface.

### Owns

- auth screens
- recipient cabinet
- contact management UI
- preference management UI
- quiet hours UI
- guest unsubscribe UI
- guest-to-user upgrade UX
- operator cabinet
- templates UI
- groups UI
- campaigns UI
- recurring notification UI
- history UI
- per-sender analytics UI

### Deliverables

- All user-facing UI text in Russian only
- API client layer based on frozen contracts
- User flows for recipient features
- Operator flows for source-side features
- Notification history views
- Per-sender analytics views
- Responsive pages for desktop and mobile

### Dependencies

- Needs Track 0 public API contracts.
- Strongly benefits from Track 1 local environment.
- Consumes APIs from Tracks 2-4.

### Coordination Contracts

- Frontend must work from published contracts and mocks.
- Frontend must not infer undocumented backend behavior.
- UI copy must remain Russian only.

### Exit Criteria

- User and operator flows from the diagrams are represented in the UI.
- Frontend can run through NGINX inside the root Docker environment.
- No English UI text appears in the product.

## Cross-Track Contracts

These rules are mandatory for all tracks.

- No breaking change to public REST, internal REST, or RabbitMQ contracts without explicit cross-track coordination.
- No unilateral changes to JWT claims, roles, machine credential flow, or unsubscribe token format.
- No cross-service business logic through direct DB coupling.
- No redefinition of canonical notification statuses outside the agreed model.
- No redefinition of sender-level versus mailing-level unsubscribe behavior.
- No real provider integration. Mocks only.
- All work must remain compatible with one shared PostgreSQL component and one root `docker-compose`.

## Integration Gates

A track is not ready to merge only because its local code works. The following gates matter:

- Contract checks pass
- Service-level tests pass
- Docker build works
- Integration wiring still works inside root `docker-compose`
- No violation of service ownership boundaries
- For frontend: Russian-only UI check passes

## Recommended Spawn Order

### Wave 1

- Track 0: Foundation

### Wave 2

After Track 0 freeze:

- Track 1: Platform and Integration
- Track 2: Recipients
- Track 3: Sources
- Track 4: Notification Pipeline
- Track 5: Frontend

## Recommended Agent Prompt Template

Use this template when spawning an agent for a track:

```text
You are working on Notification Center.

Read first:
- notifications/TRACKS.md
- notifications/PLAN.md
- CLAUDE.md
- ПП/Диаграммы/*

Your assigned track is: <TRACK NAME>

Follow all global rules from TRACKS.md.
Do not take ownership outside your track.
If any requirement is unclear, check the diagrams and project files before making a decision.
Respect frozen contracts and service boundaries.
If a change affects another track's contract, stop and report it explicitly instead of silently changing it.

Your goal in this session:
<PASTE THE SPECIFIC SUBTASKS FOR THIS TRACK>
```

## Short Recommendation

The best split is:

- first: one small foundation track
- then: five parallel tracks
- with `Notifications` and `Delivery` kept together

This gives the lowest coordination cost and the lowest risk of contract drift.
