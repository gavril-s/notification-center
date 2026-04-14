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
2. After Track 0, start Track 1 and establish the platform baseline: root repository structure, root `docker-compose.yml`, Dockerfiles, CI skeleton, NGINX routing, RabbitMQ topology, and shared conventions.
3. When the Track 1 platform baseline is available, start Tracks `2-5` in parallel.
4. Tracks `2-5` may begin against frozen contracts, mocks, and stubs before all live integrations are ready.
5. Track 4 must not block on unfinished live implementations from Tracks 2 and 3; it should start with frozen interface mocks, then switch to real integrations when those APIs are available.
6. Keep `Notifications` and `Delivery` in the same track until the end-to-end pipeline is stable.
7. Frontend should work from frozen API contracts and mocks, not backend assumptions.

## Track Overview

| Track | Name                     | Starts When          | Main Ownership                                                |
| ----- | ------------------------ | -------------------- | ------------------------------------------------------------- |
| 0     | Foundation               | immediately                | contracts, architecture freeze, base repo bootstrap           |
| 1     | Platform and Integration | after Track 0 freeze       | Docker, compose, NGINX, RabbitMQ, CI, monitoring              |
| 2     | Recipients               | after Track 1 baseline     | auth, users, contacts, preferences, unsubscribe               |
| 3     | Sources                  | after Track 1 baseline     | senders, operators, templates, groups, campaigns, credentials |
| 4     | Notification Pipeline    | after Track 1 baseline     | `Notifications` and `Delivery` together                       |
| 5     | Frontend                 | after Track 1 baseline     | Russian-only UI for user and operator flows                   |

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
- Root repository structure specification

### Deliverables

- Finalized decisions for:
  - API gateway via NGINX
  - one shared PostgreSQL component
  - auth ownership in `Recipients`
  - administrator bootstrap model
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
- Approved repository structure specification matching `PLAN.md`

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
- Traefik configuration
- RabbitMQ topology
- PostgreSQL bootstrap
- Makefile targets
- CI checks
- health and readiness endpoints
- monitoring bootstrap
- shared observability stack
- shared security scaffolding
- backup and restore procedure
- integration and release test harnesses
- runbooks for local and staged operation
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
  - Traefik
  - supporting monitoring components if included
- Base Dockerfiles for every service and frontend
- Local environment runnable with one command
- Basic CI for build, lint, contract validation, and tests
- Shared healthcheck conventions
- Shared observability bootstrap, including metrics and logging wiring
- Shared security scaffolding, including rate limiting, secret-loading conventions, and tracing propagation defaults
- Backup, restore, smoke test, and release-check scripts

### Dependencies

- Needs Track 0 contract and architecture freeze.
- After the initial platform baseline is available, continues in parallel with Tracks 2-5.

### Coordination Contracts

- Must not invent service APIs.
- Must not change domain ownership.
- Must keep Docker and compose structure compatible with all service tracks.
- Owns cross-cutting operational work that has no single domain owner: monitoring bootstrap, runbooks, backup and restore flow, and release harnesses.
- Owns cross-cutting infrastructure-level security controls such as gateway rate limiting, secret-loading conventions, and shared tracing and logging bootstrap.

### Exit Criteria

- Full stack boots locally through one `docker-compose`.
- Every service can be wired into the environment without structural rework.
- Shared operational and validation tooling exists for the other tracks to plug into.

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
- Seeded administrator bootstrap in the authentication domain
- Runtime validation for channel-aware contact data, guest claim verification, and unsubscribe token verification
- Data model that keeps history anchored to `contact_id`
- Verified guest contact claim flow
- Tests for preference precedence and unsubscribe behavior

### Dependencies

- Needs Track 0 contracts and auth rules.
- Needs the Track 1 platform baseline and environment for full integration testing.

### Coordination Contracts

- `Recipients` is the source of truth for:
  - user identity
  - interactive auth
  - contacts
  - preferences
  - unsubscribe rules
- Track 2 owns service-level authorization for recipient-facing endpoints and audit trails for recipient-domain changes.
- Track 2 owns service-level logs, metrics, traces, and runtime validation for recipient-domain APIs and flows.
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
- Runtime validation for sender credentials, template variables, campaign and group input, and recurrence rule input
- Template and campaign management flows
- Recurrence definitions

### Dependencies

- Needs Track 0 contract freeze.
- Needs the Track 1 platform baseline and environment for integration.

### Coordination Contracts

- `Sources` is the source of truth for:
  - sender entities
  - operator access
  - source-system credentials
  - templates
  - groups
  - campaigns
  - recurrence definitions
- Track 3 owns operator assignment flows, service-level authorization for source-side endpoints, and audit trails for sender, template, group, campaign, and credential changes.
- Track 3 owns service-level logs, metrics, traces, and runtime validation for source-domain APIs and flows.
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
- Runtime validation for send payloads, sender and template compatibility, rendering prerequisites, and signed unsubscribe URL generation inputs
- Mock channel adapters for `email`, `sms`, and one messenger

### Dependencies

- Needs Track 0 contracts before implementation starts.
- May start after the Track 1 platform baseline is available, using frozen mocks for recipient resolution, template lookup, due campaign lookup, and credential resolution while live integrations are still in progress.
- Needs Track 2 recipient resolution API for final live integration.
- Needs Track 3 template lookup, due campaign lookup, and credential resolution APIs for final live integration.
- Needs the Track 1 platform baseline and infrastructure.

### Coordination Contracts

- `Notifications` owns canonical notification state.
- `Delivery` owns attempt-level transport data only.
- Queue payload format is frozen by Track 0.
- Retry semantics and idempotency rules must not drift.
- Do not split `Notifications` and `Delivery` into separate tracks early.
- While upstream services are still in progress, Track 4 must use contract-faithful mocks instead of redefining upstream behavior.
- Track 4 owns service-level authorization for send, history, and analytics endpoints, plus auditability of pipeline-critical state transitions.
- Track 4 owns service-level logs, metrics, traces, and runtime validation for the notification pipeline.

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
- Needs the Track 1 platform baseline and strongly benefits from the Track 1 local environment.
- Consumes APIs from Tracks 2-4.

### Coordination Contracts

- Frontend must work from published contracts and mocks.
- Frontend must not infer undocumented backend behavior.
- UI copy must remain Russian only.
- Track 5 owns frontend-side access control behavior, protected route handling, and Russian-only validation in the UI layer.

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

## Cross-Track Hardening and Validation Ownership

Some work is cross-cutting and must be owned explicitly to avoid gaps.

- Track 1 owns shared operational work:
  - monitoring bootstrap
  - logging and metrics wiring conventions
  - tracing propagation conventions
  - gateway-level rate limiting
  - secret-loading and environment conventions
  - backup and restore procedure
  - smoke-test harness
  - release and environment runbooks
  - root CI pipelines and compose-based verification scripts
- Tracks 2-5 own service and feature correctness inside their domains:
  - unit tests
  - integration tests for their APIs and data model
  - contract compliance
  - feature-specific error handling
  - service-level authorization
  - service-level audit trails where required
  - service-level runtime validation logic
  - service-level logs, metrics, and traces following Track 1 conventions
- Track 4 additionally owns queue and pipeline validation:
  - outbox validation
  - queue contract behavior
  - idempotency
  - retries and DLQ behavior
  - end-to-end notification execution checks
- Track 5 owns frontend behavior validation:
  - Russian-only UI verification
  - page-level tests
  - API-client integration checks

Cross-system validation ownership is split as follows:

- Track 1 owns common verification infrastructure and the top-level execution of:
  - compose-based smoke tests
  - end-to-end environment startup checks
  - shared contract-test runners
  - release-readiness scripts
- Track 4 owns end-to-end notification flow scenarios that span multiple services.
- Track 1 and Track 4 jointly own load testing and failure-injection scenarios for queue backlog, retry pressure, and degraded delivery behavior.
- Track 5 owns end-to-end UI verification against the integrated environment.

Final system validation is shared, but coordinated by Track 1 using the common test and compose environment.

## Explicit Ownership Clarifications

- Track 0 freezes the administrator bootstrap model, but does not implement it.
- Track 1 provides the environment and seed-config mechanism used for bootstrap.
- Track 2 is the implementation owner of administrator bootstrap in the authentication domain.
- Track 3 is the implementation owner of operator-to-sender assignment after the initial administrator exists.
- Track 2 owns recipient-domain runtime validation.
- Track 3 owns source-domain runtime validation.
- Track 4 owns notification-pipeline runtime validation.

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

- Track 1: Platform and Integration

### Wave 3

After Track 1 platform baseline is available:

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
