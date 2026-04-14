# Notification Center Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the full Notification Center system: recipient management, contact and preference management, source/operator management, templates, campaigns, one-time and recurring notifications, asynchronous multi-channel delivery through mock channels, notification history, unsubscribe flows, and per-sender analytics.

**Architecture:** The target system consists of a React frontend behind an API gateway layer implemented by NGINX and four Go services: `Recipients`, `Sources`, `Notifications`, and `Delivery`. Services communicate via REST, except the handoff from `Notifications` to `Delivery`, which goes through RabbitMQ. PostgreSQL is deployed as a single shared infrastructure component according to the diagrams, while service boundaries are preserved at the application level. `Notifications` owns the notification lifecycle and per-sender analytics read models; `Delivery` uses mock channel adapters, stores delivery attempts in the shared database, and reports results back to `Notifications` through an internal API.

**Tech Stack:** Go + Gin, React 18+, PostgreSQL 15+, RabbitMQ, NGINX, Traefik, Docker Compose, JWT, OpenAPI/Swagger, Prometheus/Grafana, structured logging, integration tests with Docker/Testcontainers.

---

## 1. Final System Scope

The final system must support the following actors and flows:

- Registered user: registration, login, contact management, preference management, quiet hours, channel blocking, sender unsubscribe, mailing unsubscribe, notification history.
- Guest recipient: receiving notifications, unsubscribing without a full account, and later becoming a registered user without losing history.
- Source operator: sender setup, template management, campaign management, recipient group management, recurring notification setup, per-sender analytics viewing.
- External source system: API-based notification creation through `POST /api/notifications/send`.

The final product must support:

- Email, SMS, and at least one messenger channel through mock adapters.
- Template-based content generation.
- Sender-scoped and mailing-scoped preferences and unsubscribe rules.
- One-time and recurring campaigns.
- Async delivery through RabbitMQ.
- Retry handling and dead-letter processing.
- Notification history for users and operators.
- Delivery analytics per sender.

This plan describes the full target implementation and is executed in three product increments:

- Increment 1: recipients, contacts, basic preferences, basic template-based notifications, queue-based delivery.
- Increment 2: source management, full template management, campaigns, recipient groups, recurring notifications.
- Increment 3: analytics, advanced preferences, delivery control, production-like behavior through mocks, and hardening.

## 2. Recommended Architecture Decisions

These decisions should be fixed before implementation starts, because they affect every service.

### 2.1 API Gateway Strategy

Implement the API gateway layer with `NGINX` and treat it as the gateway component shown in the diagrams.

Reasoning:

- The component diagram already aligns around NGINX reverse proxy behavior.
- The C4 diagram names an API Gateway container; in the implementation plan, that gateway role is fulfilled by NGINX rather than by a separate Go service.
- A separate Go gateway adds complexity without solving a clearly required problem.
- NGINX is sufficient for path-based routing, static frontend serving, CORS, compression, and basic rate limiting.

Path routing should be:

- `/` -> frontend
- `/api/recipients/*` -> `Recipients`
- `/api/sources/*` -> `Sources`
- `/api/notifications/*` -> `Notifications`
- `/internal/*` -> internal-only service endpoints, not exposed publicly

### 2.2 Data Ownership Strategy

Use one shared PostgreSQL instance for the whole system, following the diagrams.

Rules:

- PostgreSQL is a common infrastructure component.
- Each service owns its tables and migrations logically, even though the database is shared.
- Cross-service access should still happen through service APIs or queue contracts, not through direct table coupling in business logic.
- If convenient during implementation, tables may be grouped by schema, but the key architectural requirement is a single PostgreSQL component.

### 2.3 Authentication and Authorization

Use `Recipients` as the authentication owner for interactive users.

Recommended model:

- `Recipients` issues JWT access and refresh tokens.
- Roles: `recipient_user`, `source_operator`, `admin`.
- `Sources` owns sender/operator access relationships, because sender ownership belongs to source management.
- `Notifications` and `Sources` trust JWT claims and, where needed, verify sender access through `Sources` internal APIs.

This avoids introducing a fifth backend service while still supporting role separation.

### 2.4 Notification Lifecycle Ownership

`Notifications` owns the canonical notification status.

Recommended canonical states:

- `draft`
- `scheduled`
- `queued`
- `processing`
- `sent`
- `delivered`
- `failed`
- `cancelled`
- `skipped_by_preference`

`Delivery` owns delivery attempts and mock adapter metadata, and persists those attempt records in the shared PostgreSQL component. `Notifications` must remain the source of truth for business notification state. After each delivery attempt, `Delivery` stores the attempt result and calls an internal `Notifications` endpoint to update the canonical notification status.

### 2.5 Recurring Notification Ownership

Use this split:

- `Sources` stores campaign definitions, recipient groups, and recurrence rules.
- `Notifications` owns the scheduler that periodically asks `Sources` for due campaigns and materializes actual notifications.

This keeps business definitions in `Sources` and execution logic in `Notifications`.

### 2.6 Preference Precedence Rules

Preference evaluation must be explicit and deterministic. Recommended precedence:

1. Contact is disabled or invalid.
2. Mailing-level unsubscribe is active for the selected campaign or group.
3. Sender-level unsubscribe is active.
4. Channel-level block is active.
5. Quiet hours block delivery at the current time.
6. Campaign or send request limits the allowed channels.
7. Mock delivery fallback chooses an alternative adapter inside the same channel when supported.
8. If no eligible channel remains, notification is marked `skipped_by_preference`.

### 2.7 Channel Model

Define channel types as extensible string enums, not integer-only opaque codes.

Recommended initial values:

- `email`
- `sms`
- `telegram`

Adapter implementations remain pluggable behind the channel abstraction, but the project scope includes mocks only.

### 2.8 Guest Contact Claim Model

The plan must support turning a guest recipient into a registered user without losing any previously created data.

Rules:

- `contact_id` is the stable identifier used by notifications, preferences, unsubscribe records, and history.
- A guest contact may exist before registration and therefore may have no linked `user_id` initially.
- Registration must include a contact-claim flow that verifies ownership of an existing guest contact before attaching it to a newly created user.
- Contact claiming must use normalized contact values plus a verification token or one-time code to prevent accidental or malicious reassignment.
- History and unsubscribe state must remain attached to `contact_id`, so linking a contact to a user does not rewrite historical notification records.

### 2.9 Reliable Queue Handoff

The handoff from `Notifications` to RabbitMQ must be durable and retry-safe.

Rules:

- `Notifications` writes notification records and an outbox record in the same database transaction.
- A dedicated outbox publisher reads pending outbox rows and publishes them to RabbitMQ.
- Published outbox rows are marked as sent only after broker confirmation.
- `Delivery` consumers must be idempotent for duplicate messages using `notification_id` and `attempt`.
- A reconciliation job must detect stuck outbox rows and republish them safely.

## 3. Target Repository Structure

```text
notifications/
├── README.md
├── PLAN.md
├── docker-compose.yml
├── .env.example
├── Makefile
├── contracts/
│   ├── openapi/
│   │   ├── recipients.yaml
│   │   ├── sources.yaml
│   │   ├── notifications.yaml
│   │   └── internal.yaml
│   └── events/
│       └── notification-dispatch.v1.json
├── infra/
│   ├── nginx/
│   ├── traefik/
│   ├── postgres/
│   ├── rabbitmq/
│   └── monitoring/
├── recipients/
│   ├── go.mod
│   ├── cmd/api/
│   ├── internal/
│   ├── migrations/
│   └── Dockerfile
├── sources/
│   ├── go.mod
│   ├── cmd/api/
│   ├── internal/
│   ├── migrations/
│   └── Dockerfile
├── notifications-service/
│   ├── go.mod
│   ├── cmd/api/
│   ├── cmd/scheduler/
│   ├── internal/
│   ├── migrations/
│   └── Dockerfile
├── delivery/
│   ├── go.mod
│   ├── cmd/consumer/
│   ├── internal/
│   ├── migrations/
│   └── Dockerfile
├── frontend/
│   ├── package.json
│   ├── src/
│   ├── public/
│   └── Dockerfile
└── docs/
    ├── adr/
    ├── diagrams/
    └── testing/
```

## 4. Service Responsibilities

| Component | Owns | Responsibilities |
| --- | --- | --- |
| `Recipients` | users, contacts, preferences, unsubscribe rules, auth tokens | registration, login, account profile, contact CRUD, quiet hours, channel blocking, sender unsubscribe, mailing unsubscribe, guest unsubscribe, guest-to-user linking |
| `Sources` | senders, templates, groups, campaigns, recurrence rules, operator access | operator cabinet APIs, sender scoping, template CRUD, recipient groups, campaign CRUD |
| `Notifications` | notification records, state machine, scheduling jobs, history, per-sender analytics read models | send API, template rendering, recipient resolution, preference evaluation, queue publishing, internal status updates, history APIs, analytics APIs |
| `Delivery` | delivery attempts, mock adapter results, retry state, DLQ records | queue consumption, mock delivery execution, retries, dead-letter handling, shared-DB attempt persistence, status callback to `Notifications` |
| `Frontend` | UI only | recipient cabinet, operator cabinet, history, analytics, unsubscribe UX |
| `API Gateway (NGINX)` | routing config | path routing, static assets, public entrypoint |
| `RabbitMQ` | queue topology | `notification.dispatch`, retry queues, dead-letter queues |

## 5. Core Data Contracts

The following aggregates must exist in the final system:

- User
- Contact
- Preference
- Sender
- Template
- ContactGroup
- Campaign
- RecurrenceRule
- Notification
- UnsubscribeRule
- DeliveryAttempt
- AnalyticsSnapshot or analytics read model tables

Additional compatibility rule:

- The data model must allow a contact that was originally used as a guest recipient to be linked to a registered user later without breaking history, unsubscribe records, or preferences.
- `Notification` history must remain anchored to `contact_id`, not only to `user_id`.
- `UnsubscribeRule` must support at least `sender` and `campaign_or_group` scope so both unsubscribe flows from the diagrams are implemented.

The following contracts must be versioned:

- Public REST contracts in `contracts/openapi/*.yaml`
- Queue message contract in `contracts/events/notification-dispatch.v1.json`
- Internal REST contracts for:
  - recipient resolution
  - template lookup
  - due campaign lookup
  - notification status update

The queue payload should include:

- `notification_id`
- `sender_id`
- `contact_id`
- `campaign_id`
- `group_id`
- `channel`
- `rendered_content`
- `subject`
- `unsubscribe_url`
- `metadata`
- `event_id`
- `attempt`
- `trace_id`
- `created_at`

## 6. Implementation Phases

### Task 1: Architecture Baseline and Contract Freeze

**Objective:** Remove ambiguities in the current diagrams and create a stable foundation for parallel implementation.

- [ ] Approve the gateway decision: NGINX-only, no separate Go gateway.
- [ ] Approve the PostgreSQL strategy: one shared PostgreSQL component following the diagrams.
- [ ] Approve auth ownership in `Recipients`.
- [ ] Approve the canonical notification state machine.
- [ ] Approve preference precedence rules.
- [ ] Approve the guest contact claim model.
- [ ] Approve the split between sender-level and mailing-level unsubscribe.
- [ ] Approve the split between `Sources` recurrence ownership and `Notifications` scheduling ownership.
- [ ] Approve the outbox-based RabbitMQ publishing flow.
- [ ] Write Architecture Decision Records for the decisions above.
- [ ] Define public REST contracts for `Recipients`, `Sources`, and `Notifications`.
- [ ] Define internal REST contracts between services.
- [ ] Define the RabbitMQ dispatch event schema.
- [ ] Define standard service conventions: config layout, env variables, health checks, logging format, error response format, trace propagation.

**Exit criteria:**

- All critical architecture choices are written down.
- Service teams can work independently without guessing ownership.
- REST and queue contracts are stable enough to generate mocks and tests.

### Task 2: Repository Bootstrap and Platform Foundation

**Objective:** Create the implementation repository skeleton and local development environment.

- [ ] Create the directory structure for all services, frontend, contracts, docs, and infrastructure.
- [ ] Add root `docker-compose.yml` for PostgreSQL, RabbitMQ, NGINX, Traefik, all services, frontend, and monitoring tools.
- [ ] Add a root `Makefile` with targets for `lint`, `test`, `build`, `up`, `down`, and `migrate`.
- [ ] Configure local PostgreSQL as one shared component for all services.
- [ ] Configure RabbitMQ exchanges, queues, retry queues, and dead-letter queues.
- [ ] Configure NGINX for frontend serving and API path routing.
- [ ] Add base Dockerfiles for Go services and frontend.
- [ ] Add CI workflow for build, lint, unit tests, contract checks, and frontend checks.
- [ ] Add OpenAPI validation and queue schema validation to CI.
- [ ] Add shared observability stack: Prometheus, Grafana, and log aggregation for local development.

**Exit criteria:**

- A developer can run the full system locally with one command.
- Every service exposes `/health` and `/ready`.
- CI can validate code and contracts even before business features are complete.

### Task 3: Recipients Service

**Objective:** Implement recipient identity, contacts, and delivery preferences.

- [ ] Implement user registration and login.
- [ ] Implement JWT access and refresh token flow.
- [ ] Implement account profile retrieval.
- [ ] Implement contact CRUD with channel-aware validation.
- [ ] Implement contact verification flags and duplicate prevention.
- [ ] Implement preferences per contact and per sender.
- [ ] Implement quiet hours and channel enable/disable rules.
- [ ] Implement sender-level unsubscribe.
- [ ] Implement mailing-level unsubscribe for campaigns or groups.
- [ ] Implement guest unsubscribe flow using secure unsubscribe tokens.
- [ ] Implement guest-to-user contact linking through verified contact claim.
- [ ] Implement internal recipient resolution API for `Notifications`.
- [ ] Implement audit logging for preference changes.
- [ ] Add unit and integration tests for auth, validation, quiet hours, sender and mailing unsubscribe precedence, guest linking, and unauthorized access.

**Recommended public endpoints:**

- `POST /api/recipients/register`
- `POST /api/recipients/login`
- `POST /api/recipients/refresh`
- `GET /api/recipients/me`
- `GET /api/recipients/contacts`
- `POST /api/recipients/contacts`
- `PUT /api/recipients/contacts/{contact_id}`
- `DELETE /api/recipients/contacts/{contact_id}`
- `GET /api/recipients/preferences`
- `PUT /api/recipients/preferences`
- `POST /api/recipients/unsubscribe`

**Recommended internal endpoints:**

- `POST /internal/recipients/resolve`
- `GET /internal/recipients/users/{user_id}/contacts`

**Exit criteria:**

- The system can authenticate users.
- Contacts and preferences can be managed through the API.
- Guest recipients can later be attached to a registered account.
- `Notifications` can resolve eligible recipients without direct DB access.

### Task 4: Sources Service

**Objective:** Implement sender-side management of templates, groups, campaigns, and recurrence rules.

- [ ] Implement sender entity and sender-scoped operator access.
- [ ] Implement template CRUD with versioning support.
- [ ] Implement template preview and variable validation.
- [ ] Implement contact groups for mailing lists.
- [ ] Implement campaign CRUD for one-time sends.
- [ ] Implement recurrence rules for repeating campaigns.
- [ ] Implement campaign scheduling metadata.
- [ ] Implement internal APIs for template lookup and due campaign retrieval.
- [ ] Add tests for sender isolation, template variable validation, and recurrence rule calculation.

**Recommended public endpoints:**

- `GET /api/sources/senders`
- `POST /api/sources/senders`
- `GET /api/sources/templates`
- `POST /api/sources/templates`
- `PUT /api/sources/templates/{template_id}`
- `DELETE /api/sources/templates/{template_id}`
- `GET /api/sources/groups`
- `POST /api/sources/groups`
- `PUT /api/sources/groups/{group_id}`
- `POST /api/sources/groups/{group_id}/members`
- `GET /api/sources/campaigns`
- `POST /api/sources/campaigns`
- `PUT /api/sources/campaigns/{campaign_id}`

**Recommended internal endpoints:**

- `GET /internal/sources/templates/{template_id}`
- `GET /internal/sources/campaigns/{campaign_id}`
- `GET /internal/sources/campaigns/due`
- `GET /internal/sources/senders/{sender_id}/operators/{user_id}`

**Exit criteria:**

- Operators can create templates, groups, and campaigns.
- `Notifications` can retrieve all data needed to create concrete notifications.
- Recurring campaigns are defined and queryable.

### Task 5: Notifications Service

**Objective:** Implement orchestration, scheduling, notification history, and per-sender analytics ownership.

- [ ] Implement `POST /api/notifications/send` for external source systems.
- [ ] Add idempotency support for send requests.
- [ ] Validate sender, template, and payload variables.
- [ ] Fetch template and sender data from `Sources`.
- [ ] Fetch recipient eligibility data from `Recipients`.
- [ ] Render notification content per target contact and channel.
- [ ] Generate signed unsubscribe URLs for sender-level and mailing-level unsubscribe flows and embed them into rendered content when the channel supports links.
- [ ] Evaluate preferences and mark blocked records as `skipped_by_preference`.
- [ ] Persist notification records and their initial states.
- [ ] Persist outbox rows in the same transaction as notification creation.
- [ ] Publish dispatch messages to RabbitMQ through an outbox publisher.
- [ ] Implement scheduler process for delayed and recurring notifications.
- [ ] Implement internal endpoint for delivery status updates from `Delivery`.
- [ ] Implement user notification history APIs.
- [ ] Implement per-sender analytics APIs and read models.
- [ ] Add tests for idempotency, rendering failures, unsubscribe URL generation, preference filtering, outbox recovery, retry-safe queue publishing, and status transitions.

**Recommended public endpoints:**

- `POST /api/notifications/send`
- `GET /api/notifications/{notification_id}`
- `GET /api/notifications/history`
- `GET /api/notifications/analytics`

**Recommended internal endpoints:**

- `POST /internal/notifications/{notification_id}/delivery-status`
- `POST /internal/notifications/scheduler/run`
- `GET /internal/notifications/{notification_id}`

**Exit criteria:**

- The system can create, store, and queue notifications end-to-end.
- Scheduled and recurring notifications can be materialized automatically.
- Users and operators can query history and per-sender analytics without direct DB joins across services.

### Task 6: Delivery Service

**Objective:** Implement reliable mock delivery and feedback into the notification lifecycle.

- [ ] Implement RabbitMQ consumer for dispatch messages.
- [ ] Add worker pool and prefetch tuning.
- [ ] Implement mock adapters for `email`, `sms`, and one messenger channel.
- [ ] Keep the adapter interface extensible, but do not integrate real providers.
- [ ] Persist each delivery attempt with mock request and result metadata.
- [ ] Add retry policy with exponential backoff.
- [ ] Add dead-letter handling for exhausted retries.
- [ ] Make consumer processing idempotent for duplicate queue deliveries.
- [ ] Simulate final delivery statuses inside the mock delivery layer.
- [ ] Report delivery outcomes back to `Notifications`.
- [ ] Add tests for retries, poison messages, duplicate deliveries, persisted attempts, and simulated delivery failures.

**Exit criteria:**

- Every queued notification either succeeds, retries, or lands in a dead-letter path with traceable metadata.
- `Notifications` remains the source of truth for business status.
- Delivery behavior is fully demonstrated through mocks without external providers.

### Task 7: Frontend

**Objective:** Build a complete user and operator web interface.

- [ ] Implement authentication screens.
- [ ] Implement recipient cabinet for contacts and preferences.
- [ ] Implement quiet hours and channel control UI.
- [ ] Implement guest unsubscribe page with sender-level and mailing-level choices when supported by the notification.
- [ ] Implement guest-to-user upgrade UX.
- [ ] Implement operator cabinet for templates, groups, and campaigns.
- [ ] Implement campaign scheduling and recurring setup UI.
- [ ] Implement notification history screens for users.
- [ ] Implement per-sender analytics dashboard for operators.
- [ ] Implement empty states, loading states, and error handling for all major flows.
- [ ] Ensure responsive behavior for desktop and mobile.
- [ ] Add frontend tests for forms, API integration, and critical user flows.

**Recommended route groups:**

- `/login`
- `/app/profile`
- `/app/contacts`
- `/app/preferences`
- `/app/history`
- `/operator/templates`
- `/operator/groups`
- `/operator/campaigns`
- `/operator/analytics`
- `/unsubscribe`

**Exit criteria:**

- All user-visible use cases from the current diagrams are supported by the UI.
- The frontend works through NGINX without direct service exposure.
- Core flows remain usable on mobile screens.

### Task 8: Security, Observability, and Operations

**Objective:** Make the system supportable, testable, and safe.

- [ ] Add role-based authorization checks to all public APIs.
- [ ] Add audit trails for template changes, campaign changes, and preference changes.
- [ ] Add structured logs with correlation IDs.
- [ ] Add metrics for API latency, queue depth, retries, failure ratios, mock adapter latency, and scheduler throughput.
- [ ] Add traces across gateway, services, and queue handoff.
- [ ] Add rate limiting for public endpoints.
- [ ] Add secret handling strategy even for local mock-only development.
- [ ] Add database backup and restore procedure.
- [ ] Add operational runbooks for queue backlog, failed delivery spikes, and mock adapter failures.
- [ ] Add smoke tests for local and staged environments.

**Exit criteria:**

- Failures can be diagnosed from logs, metrics, and traces.
- Sensitive endpoints and credentials are protected.
- The system can be demonstrated as a production-like capstone, not just a code prototype.

### Task 9: System Validation and Release Readiness

**Objective:** Validate the full architecture and ship the final system incrementally.

- [ ] Add unit tests to each service for domain logic and validation.
- [ ] Add contract tests for public REST APIs and internal REST APIs.
- [ ] Add event contract tests for the RabbitMQ payload.
- [ ] Add integration tests against PostgreSQL and RabbitMQ.
- [ ] Add end-to-end tests through NGINX for critical flows.
- [ ] Verify user registration, login, contact creation, and preference updates.
- [ ] Verify template and campaign creation by an operator.
- [ ] Verify notification creation by an external source system.
- [ ] Verify queue publishing, outbox recovery, consumption, and mock delivery result propagation.
- [ ] Verify history visibility for the user.
- [ ] Verify guest unsubscribe from both sender and mailing scopes and later guest-to-user linking.
- [ ] Verify per-sender analytics after completed deliveries.
- [ ] Add load tests for burst notification creation and queue drain.
- [ ] Add failure injection tests for queue backlog and mock channel failures.

**Standard validation commands to support in the repository:**

- `docker compose up -d`
- `make lint`
- `make test`
- `make build`
- `make integration-test`
- `make e2e-test`
- `cd frontend && npm test`
- `cd frontend && npm run build`

**Exit criteria:**

- The entire system can be started locally.
- All critical flows pass end-to-end.
- The project is ready for a demo, report, and final defense.

## 7. Increment Roadmap

### Increment 1

Scope:

- Repository bootstrap and local infrastructure.
- `Recipients` with registration, contacts, and basic preferences.
- Minimal `Sources` support for sender and template lookup.
- `Notifications` send API and queue publishing.
- `Delivery` consumer with mock adapters.
- Basic frontend for recipient preferences and simple operator actions.

Success criteria:

- A source system can request a notification.
- The system respects basic recipient preferences.
- A message is queued, consumed, and marked with final status.
- A user can see or infer delivery history.

### Increment 2

Scope:

- Full `Sources` implementation.
- Contact groups and campaigns.
- Recurrence rules.
- `Notifications` scheduler for due campaigns.
- Extended operator frontend.

Success criteria:

- Operators can create groups, campaigns, and recurring notifications.
- Scheduled notifications are generated automatically.
- Source-side workflows no longer depend on manual data preparation.

### Increment 3

Scope:

- Advanced preferences and unsubscribe logic.
- Per-sender analytics and dashboards.
- Delivery control, retries, dead-letter handling, and operational observability.
- Hardening of mock delivery scenarios for demo and defense.

Success criteria:

- The full product vision from the diagrams is implemented.
- Per-sender analytics work against real notification lifecycle data.
- The system behaves predictably under failure conditions using mock delivery channels.

## 8. Final Acceptance Criteria

The implementation is complete when all conditions below are true:

- A registered user can manage contacts and preferences.
- A guest recipient can unsubscribe without a full account at both sender and mailing scope.
- A guest recipient can later become a registered user without losing existing history and preferences.
- A source operator can manage senders, templates, groups, and campaigns.
- An external source system can create notifications through the public API.
- Notifications are rendered, include unsubscribe links where required, are filtered by preferences, queued durably, and delivered asynchronously.
- Delivery failures are retried and traceable.
- Notification history is available to users.
- Per-sender analytics are available to operators.
- The full system runs in Docker Compose behind a single public gateway.
- The architecture matches the final documented boundaries and follows the diagrams, including a single PostgreSQL component.

## 9. Main Risks and Mitigations

- **Risk:** Current diagrams label the gateway differently across artifacts.
  **Mitigation:** Treat the API Gateway role as an NGINX-based gateway layer and record that mapping explicitly in ADRs.

- **Risk:** Shared PostgreSQL can erode service boundaries.
  **Mitigation:** Keep ownership at the application layer, use separate migrations per service, and avoid direct business coupling through foreign reads/writes across services.

- **Risk:** Auth scope can grow beyond the capstone’s needs.
  **Mitigation:** Keep auth inside `Recipients` and avoid a separate identity service unless a real blocker appears.

- **Risk:** Scheduling and recurrence logic can become duplicated across `Sources` and `Notifications`.
  **Mitigation:** Store definitions in `Sources`, execute schedules only in `Notifications`.

- **Risk:** Mock delivery can become too simplistic for demonstration.
  **Mitigation:** Implement realistic success, delay, retry, and failure scenarios in the mock adapters.

- **Risk:** Analytics can become expensive if built directly on transactional tables.
  **Mitigation:** Build per-sender analytics read models inside `Notifications`.

## 10. Resolved Project Assumptions

- External providers are not part of the project scope; delivery is implemented through mocks only.
- Real provider credentials and real delivery webhooks are not required.
- Guest recipients may later become registered users, so the data model must support contact ownership reassignment without losing history.
- Unsubscribe must work at both sender scope and mailing scope.
- Analytics are limited to per-sender reporting.
- Final delivery statuses are simulated by the mock delivery layer.
- Infrastructure follows the diagrams and uses one PostgreSQL component for the whole system.

This plan should be treated as the implementation source of truth for the final Notification Center system.
