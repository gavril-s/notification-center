# Track 0 Specification: Foundation

## Purpose

Track 0 freezes the shared rules that all other tracks must follow. It is the source of truth for interface conventions, cross-track ownership boundaries, and schema ownership rules. It defines the contracts. It does not implement business logic.

Source documents:

- `notifications/PLAN.md`
- `notifications/TRACKS.md`
- `CLAUDE.md`
- `ПП/Диаграммы/*`

## Start Gate

This track starts immediately.

All other tracks may start only after Track 0 has produced and published its decisions.

## Owns

- Contract freeze for public REST APIs
- Contract freeze for internal REST APIs
- Contract freeze for RabbitMQ event payloads
- Common API conventions
- Common auth and authorization conventions
- Common ID, time, enum, and error formats
- Common schema ownership map
- Common migration ownership rules
- Repository structure specification

## Does Not Own

- Docker or infrastructure implementation
- Any business service implementation
- Any business database table implementation
- Frontend implementation

## Global Interface Conventions

These conventions are mandatory for all tracks.

### Paths

- Public recipient API prefix: `/api/recipients`
- Public source API prefix: `/api/sources`
- Public notification API prefix: `/api/notifications`
- Internal recipient API prefix: `/internal/recipients`
- Internal source API prefix: `/internal/sources`
- Internal notification API prefix: `/internal/notifications`

### Identifiers and Time

- All business identifiers are UUID strings.
- All timestamps in APIs and events use RFC3339 in UTC.
- All enums are lowercase strings.
- All status and channel values are string enums, not integer codes.

### Common Headers

- `X-Request-ID` is required on all internal service-to-service requests.
- `X-Trace-ID` is required on all internal service-to-service requests and queue metadata.
- Interactive requests use `Authorization: Bearer <jwt>`.
- External source-system send requests use `X-Integration-Key: <secret>`.

### Error Envelope

All public and internal APIs return errors in this shape:

```json
{
  "error": {
    "code": "string_code",
    "message": "human readable message",
    "details": {},
    "request_id": "uuid"
  }
}
```

### Pagination

- List endpoints use `page` and `size` query parameters.
- Default `page=1`.
- Default `size=20`.
- Maximum `size=100`.

Paginated response envelope:

```json
{
  "items": [],
  "page": 1,
  "size": 20,
  "total": 0
}
```

Unless a spec says otherwise, paginated history and list endpoints must use deterministic ordering.

## Auth and Authorization Freeze

### Interactive Auth

- `Recipients` owns interactive authentication.
- JWT claims are:
  - `sub`: user UUID
  - `role`: `recipient_user`, `source_operator`, or `admin`
  - `iat`
  - `exp`
- Sender access is not stored inside JWT claims. `Sources` remains authoritative for sender-to-operator permissions.

### Source-System Auth

- External source systems do not use JWT.
- `X-Integration-Key` identifies a sender integration credential.
- `Sources` is the source of truth for integration credentials.
- `Notifications` must resolve the sender by calling `POST /internal/sources/senders/resolve-credential`.

### Admin Bootstrap

- The bootstrap model is seeded administrator bootstrap in the authentication domain.
- `Recipients` implements bootstrap.
- `Sources` implements operator-to-sender assignment after an administrator exists.

## Notification Lifecycle Freeze

Canonical notification states owned by `Notifications`:

- `draft`
- `scheduled`
- `queued`
- `processing`
- `sent`
- `delivered`
- `failed`
- `cancelled`
- `skipped_by_preference`

Transport-level attempt state belongs to `Delivery` only and must not replace canonical notification state.

## Preference and Unsubscribe Freeze

Evaluation order:

1. invalid or disabled contact
2. mailing-level unsubscribe
3. sender-level unsubscribe
4. channel-level block
5. quiet hours
6. channel restriction from campaign or send request
7. no eligible channel -> `skipped_by_preference`

Supported unsubscribe scopes:

- `sender`
- `campaign_or_group`

### Unsubscribe URL Contract

Track 4 generates unsubscribe URLs. Track 2 verifies them. Track 5 consumes them in the UI.

Frozen public format:

```text
/unsubscribe?token=<opaque_string>&available_scopes=<sender|campaign_or_group|sender,campaign_or_group>
```

Frozen token semantics:

- The token is opaque to the frontend.
- The signed token payload must contain:
  - `contact_id`
  - `sender_id`
  - `available_scopes`
  - `campaign_id or null`
  - `group_id or null`
  - `exp`
- `available_scopes` may contain one or both of `sender` and `campaign_or_group`.
- If `campaign_or_group` is present, exactly one of `campaign_id` or `group_id` must be present.
- Canonical mailing target rule:
  - if a notification was created from a campaign, `campaign_id` is the canonical mailing unsubscribe target
  - `group_id` is used as the mailing target only when there is no `campaign_id`
- Token expiry is 30 days from generation.

Frozen ownership:

- Track 4 generates the token and URL.
- Track 2 verifies token validity and applies the chosen unsubscribe state.
- Track 5 must not decode or reinterpret token payload fields.
- `available_scopes` in the query string is a UI hint only. Track 2 must rely on the signed token payload as the source of truth.

## Queue Contract Freeze

RabbitMQ dispatch event name:

- `notification.dispatch.v1`

Required fields:

```json
{
  "event_id": "uuid",
  "notification_id": "uuid",
  "sender_id": "uuid",
  "contact_id": "uuid",
  "campaign_id": "uuid or null",
  "group_id": "uuid or null",
  "channel": "email|sms|telegram",
  "subject": "string or null",
  "rendered_content": "string",
  "unsubscribe_url": "string or null",
  "metadata": {},
  "attempt": 1,
  "trace_id": "uuid",
  "created_at": "RFC3339 UTC"
}
```

Rules:

- `Notifications` publishes this event.
- `Delivery` consumes this event.
- The payload is versioned and cannot be changed without explicit coordination.

## Schema Ownership Freeze

One PostgreSQL component is used for the whole system, but logical ownership is strict.

- Schema `recipients` is owned by Track 2.
- Schema `sources` is owned by Track 3.
- Schema `notifications` is owned by Track 4.
- Schema `delivery` is owned by Track 4.

Rules:

- A track may write only to schemas it owns.
- Cross-service business reads from another service schema are forbidden.
- Migrations are owned by the track that owns the schema.
- Shared PostgreSQL does not mean shared write access.

## Repository Structure Freeze

Track 0 defines the structure. Track 1 creates it.

- `contracts/openapi/`
- `contracts/events/`
- `infra/nginx/`
- `infra/traefik/`
- `infra/postgres/`
- `infra/rabbitmq/`
- `infra/monitoring/`
- `recipients/`
- `sources/`
- `notifications-service/`
- `delivery/`
- `frontend/`
- `notifications/specs/`

## Required Deliverables

- Frozen public API specs
- Frozen internal API specs
- Frozen queue payload spec
- Frozen ownership map for auth, state, unsubscribe, recurrence, and DB schemas
- Shared conventions document references inside the repo

## Change Control

Any change to the following requires explicit cross-track coordination:

- API path prefixes
- error envelope
- JWT claim structure
- `X-Integration-Key` auth model
- queue payload fields or semantics
- schema ownership map
- canonical notification states
- unsubscribe scopes or precedence

## Parallel Safety Criteria

Parallel work is safe only when:

- Track 0 outputs are written down and accepted.
- Downstream tracks can implement using frozen contracts without inventing payloads.
- No downstream track writes into another track's schema.
