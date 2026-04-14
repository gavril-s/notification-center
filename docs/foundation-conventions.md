# Foundation Conventions

This document freezes the Track 0 cross-service conventions that downstream tracks must implement without reinterpretation.

## Error Envelope

All public and internal REST APIs return errors in this shape:

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

Rules:

- `code` is a stable lowercase string intended for client logic.
- `message` is human-readable and may be shown in logs or UI handling layers.
- `details` is an open object for validation or domain-specific metadata.
- `request_id` identifies the request being handled.

## Common Headers

Public interactive requests:

- `Authorization: Bearer <jwt>`

Public source-system send requests:

- `X-Integration-Key: <secret>`

Internal service-to-service requests:

- `X-Request-ID: <uuid>` required
- `X-Trace-ID: <uuid>` required

Queue metadata:

- `X-Trace-ID` semantics are carried by the event field `trace_id`.

## Pagination Envelope

List endpoints use query parameters:

- `page`, default `1`
- `size`, default `20`, maximum `100`

Paginated response shape:

```json
{
  "items": [],
  "page": 1,
  "size": 20,
  "total": 0
}
```

Rules:

- Ordering must be deterministic unless an endpoint freezes a stronger ordering rule.
- Notification history ordering is `created_at desc`, then `notification_id desc`.

## Auth Model

Interactive auth:

- `Recipients` owns authentication for users.
- JWT claims are frozen to `sub`, `role`, `iat`, `exp`.
- Allowed roles: `recipient_user`, `source_operator`, `admin`.
- Sender permissions are not embedded into JWT claims.

Machine-to-machine auth:

- External source systems do not use JWT.
- `X-Integration-Key` identifies a sender integration credential.
- `Sources` owns credential truth.
- `Notifications` validates credentials through `POST /internal/sources/senders/resolve-credential`.

## Unsubscribe URL And Token Contract

Frozen URL format:

```text
/unsubscribe?token=<opaque_string>&available_scopes=<sender|campaign_or_group|sender,campaign_or_group>
```

Rules:

- `token` is opaque outside backend verification.
- `available_scopes` is a UI hint only and is not authoritative.
- Track 4 generates the signed token and URL.
- Track 2 verifies the token and applies the selected unsubscribe scope.
- Track 5 must not decode or reinterpret token fields.

Signed token payload must contain:

- `contact_id`
- `sender_id`
- `available_scopes`
- `campaign_id` or `null`
- `group_id` or `null`
- `exp`

Additional rules:

- `available_scopes` may contain one or both of `sender` and `campaign_or_group`.
- If `campaign_or_group` is present, exactly one of `campaign_id` or `group_id` must be present.
- If a notification comes from a campaign, `campaign_id` is the canonical mailing unsubscribe target.
- `group_id` is the mailing target only when there is no `campaign_id`.
- Token expiry is frozen to 30 days from generation.

## Schema Ownership Map

One PostgreSQL component is shared at infrastructure level, but logical ownership is strict:

| Schema | Owner |
| --- | --- |
| `recipients` | Track 2 / Recipients |
| `sources` | Track 3 / Sources |
| `notifications` | Track 4 / Notifications |
| `delivery` | Track 4 / Delivery |

Rules:

- A track writes only to schemas it owns.
- Cross-service business reads through database coupling are forbidden.
- Migrations are owned by the track that owns the schema.
- Shared PostgreSQL does not imply shared write access.

## Canonical Notification States

`Notifications` owns the canonical state machine. Frozen values:

- `draft`
- `scheduled`
- `queued`
- `processing`
- `sent`
- `delivered`
- `failed`
- `cancelled`
- `skipped_by_preference`

Rules:

- `Delivery` owns transport attempts and retry behavior only.
- Transport attempt outcomes must not redefine canonical notification state ownership.
