# Track 4 Specification: Notification Pipeline

## Purpose

Track 4 owns the combined execution pipeline across `Notifications` and `Delivery` until the end-to-end flow is stable. It is responsible for taking an authenticated send request, materializing notifications, publishing queue events, consuming them, simulating delivery, and feeding results back into canonical notification state.

## Start Gate

Starts after:

- Track 0 contract freeze
- Track 1 platform baseline

It may start with mocks for Track 2 and Track 3 APIs, but it must switch to live integrations using the frozen contracts.

## Owns

- `POST /api/notifications/send`
- sender credential validation flow usage
- template rendering
- recipient resolution integration
- unsubscribe URL generation
- preference evaluation usage
- notification persistence
- outbox persistence and publishing
- scheduler
- RabbitMQ consumer
- retry and DLQ behavior
- transport-level attempt recording
- callback from `Delivery` to canonical `Notifications` state
- history APIs
- per-sender analytics read models

## Does Not Own

- Template correctness rules
- Sender credential issuance
- Recipient preference truth
- Interactive auth token issuance

## Schema Ownership

Owned schemas:

- `notifications`
- `delivery`

Owned tables in `notifications`:

- `notifications`
- `notification_outbox`
- `notification_history`
- `sender_analytics_daily`

Owned tables in `delivery`:

- `delivery_attempts`
- `delivery_dead_letters`

Track 4 must not write to:

- `recipients.*`
- `sources.*`

History anchor rule:

- `notification_history` is anchored to `contact_id` as the stable recipient identity.
- `user_id` may be stored as an optional convenience field, but it must not replace `contact_id` as the primary historical anchor.
- Guest-to-user linking must not require rewriting history rows.

## Public API Ownership

Owned public endpoints:

- `POST /api/notifications/send`
- `GET /api/notifications/{notification_id}`
- `GET /api/notifications/history`
- `GET /api/notifications/analytics`

`POST /api/notifications/send`

Request:

```json
{
  "template_id": "uuid",
  "campaign_id": "uuid or null",
  "contact_ids": ["uuid"],
  "channels": ["email", "sms", "telegram"],
  "variables": {},
  "scheduled_at": "RFC3339 UTC or null",
  "idempotency_key": "string"
}
```

Rules:

- Sender identity comes from `X-Integration-Key` resolution, not from request body.
- `idempotency_key` is required.
- The current frozen contract allows only one effective channel per request.
- `channels` must contain exactly one value.
- The selected template must have the same channel value.

When both sender-level and campaign-or-group unsubscribe are valid for a notification, Track 4 must generate a token whose allowed scope set contains both values and must emit the matching `available_scopes` UI hint in the unsubscribe URL.

For mailing-level unsubscribe targeting:

- If `campaign_id` exists, Track 4 must encode mailing scope using `campaign_id`.
- If there is no `campaign_id`, Track 4 may encode mailing scope using `group_id`.

`GET /api/notifications/history`

Request and response contract rule:

- Recipient mode request: `GET /api/notifications/history?page=<n>&size=<n>`
- Operator mode request: `GET /api/notifications/history?mode=operator&sender_id=<uuid>&campaign_id=<uuid or null>&status=<status or null>&page=<n>&size=<n>`

- History supports two modes:
  - recipient mode: resolved by authenticated user through linked `contact_id` values
  - operator mode: filtered by `sender_id` after operator permission is verified through `GET /internal/sources/senders/{sender_id}/operators/{user_id}`
- History response must use the frozen paginated response envelope from Track 0.
- History response rows must include `notification_id`, `contact_id`, `channel`, `status`, `sender_id`, `campaign_id`, `created_at`, and `delivered_at or null`.
- Operator mode may additionally filter by `status` and `campaign_id`.
- Ordering is `created_at desc`, then `notification_id desc` for tie-breaking.

## Internal API Ownership

Owned internal endpoints:

- `POST /internal/notifications/{notification_id}/delivery-status`
- `POST /internal/notifications/scheduler/run`
- `GET /internal/notifications/{notification_id}`

Consumed internal endpoints:

- `POST /internal/recipients/resolve`
- `GET /internal/recipients/users/{user_id}/contacts`
- `GET /internal/sources/templates/{template_id}`
- `GET /internal/sources/campaigns/{campaign_id}`
- `GET /internal/sources/campaigns/due`
- `GET /internal/sources/groups/{group_id}/members`
- `GET /internal/sources/senders/{sender_id}/operators/{user_id}`
- `POST /internal/sources/senders/resolve-credential`

`GET /internal/sources/campaigns/due` usage rule:

- Track 4 must call this endpoint with `as_of`, `limit`, and optional `cursor`.
- The endpoint is read-only and does not reserve work.
- Track 4 must treat `occurrence_id` as the stable deduplication key for recurring materialization.
- Track 4 must follow `next_cursor` until it becomes null when polling all due work.
- Track 4 owns deduplication, claim, and idempotent materialization in the `notifications` schema.

Recipient history resolution rule:

- Track 4 must resolve authenticated user history through `GET /internal/recipients/users/{user_id}/contacts`.
- Track 4 must not infer user-to-contact links from any local schema copy.

`POST /internal/notifications/{notification_id}/delivery-status`

Request:

```json
{
  "attempt_id": "uuid",
  "transport_status": "accepted|failed|delivered",
  "provider_code": "mock_provider",
  "error_code": "string or null",
  "occurred_at": "RFC3339 UTC"
}
```

Track 4 owns mapping transport status into canonical notification status.

Group membership rule:

- Track 4 does not infer recipient membership from campaign payloads.
- Track 4 must fetch campaign group members only through `GET /internal/sources/groups/{group_id}/members`.
- Scheduled and recurring campaigns use live group membership at execution time, not a snapshot captured earlier.

## Queue Ownership

Track 4 owns production and consumption of `notification.dispatch.v1`.

Rules:

- `Notifications` writes notification and outbox rows in one transaction.
- Outbox publisher sends events to RabbitMQ.
- `Delivery` consumer is idempotent by `notification_id` plus `attempt`.

## Validation Ownership

Track 4 owns runtime validation for:

- send payload shape and required fields
- sender/template compatibility
- rendering prerequisites
- signed unsubscribe URL generation inputs
- idempotency behavior

Signed unsubscribe URL generation must follow the frozen Track 0 unsubscribe URL contract exactly.

Track 4 does not own contact or template semantic validation rules outside its inputs.

## Auth, Audit, and Observability Ownership

Track 4 owns:

- authorization for send, history, and analytics endpoints
- auditability of pipeline-critical state transitions
- service-level logs, metrics, and traces for `Notifications` and `Delivery`

## Change Control

Track 4 cannot change without coordination:

- `notification.dispatch.v1`
- recipient resolution contract
- source credential resolution contract
- canonical notification status list
- unsubscribe URL meaning

## Parallel Safety Criteria

Parallel work is safe when:

- Track 4 writes only `notifications.*` and `delivery.*`
- Track 2 owns recipient truth
- Track 3 owns sender, template, and campaign truth
- Track 4 consumes upstream contracts exactly as frozen
