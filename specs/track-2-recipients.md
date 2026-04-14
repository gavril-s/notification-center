# Track 2 Specification: Recipients

## Purpose

Track 2 owns recipient identity, contacts, preferences, unsubscribe rules, guest-to-user claim flow, and interactive authentication.

## Start Gate

Starts after:

- Track 0 contract freeze
- Track 1 platform baseline

## Owns

- User registration and login
- JWT issuance and refresh
- Interactive auth enforcement for recipient-facing routes
- Contacts and contact verification state
- Preferences
- Quiet hours
- Sender-level unsubscribe
- Mailing-level unsubscribe
- Guest unsubscribe token verification
- Guest contact claim flow
- Internal recipient resolution API
- Seeded administrator bootstrap in the authentication domain

## Does Not Own

- Sender entities
- Templates
- Campaigns or groups as source-side entities
- Canonical notification status
- Delivery attempts

## Schema Ownership

Owned schema: `recipients`

Owned tables:

- `users`
- `refresh_tokens`
- `contacts`
- `preferences`
- `unsubscribe_rules`
- `guest_claim_tokens`
- `unsubscribe_tokens`
- `audit_log`

Track 2 must not write to:

- `sources.*`
- `notifications.*`
- `delivery.*`

## Public API Ownership

Owned public endpoints:

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

### Minimum Request/Response Freeze

`POST /api/recipients/register`

Request:

```json
{
  "login": "string",
  "password": "string",
  "claim_contact_token": "string or null"
}
```

Response:

```json
{
  "user_id": "uuid",
  "access_token": "string",
  "refresh_token": "string"
}
```

`PUT /api/recipients/preferences`

Request:

```json
{
  "contact_id": "uuid",
  "sender_id": "uuid or null",
  "scope_type": "global|sender|campaign|group",
  "scope_id": "uuid or null",
  "enabled": true,
  "quiet_from": "HH:MM or null",
  "quiet_to": "HH:MM or null",
  "blocked_channels": ["email"]
}
```

Rules:

- Mailing-scoped preference and unsubscribe state must support both campaign scope and group scope.
- `scope_type=campaign` means `scope_id` contains a campaign UUID.
- `scope_type=group` means `scope_id` contains a group UUID.

Contact deletion rule:

- `DELETE /api/recipients/contacts/{contact_id}` is a soft delete only.
- Historical `contact_id` values must remain resolvable for notification history purposes.
- Soft-deleted contacts may be inactive for future sends, but must not be physically removed in a way that breaks history lookup.

`POST /api/recipients/unsubscribe`

Request:

```json
{
  "token": "opaque_string",
  "selected_scope": "sender|campaign_or_group"
}
```

Rules:

- Track 2 verifies that `selected_scope` is allowed by the signed token payload.
- Track 2 applies unsubscribe state at sender scope or campaign-or-group scope according to `selected_scope`.
- Track 2 must not trust frontend-provided scope data unless the token permits it.

## Internal API Ownership

Owned internal endpoints:

- `POST /internal/recipients/resolve`
- `GET /internal/recipients/users/{user_id}/contacts`

`GET /internal/recipients/users/{user_id}/contacts`

Response:

```json
{
  "contacts": [
    {
      "contact_id": "uuid",
      "channel": "email|sms|telegram",
      "value": "string",
      "enabled": true
    }
  ]
}
```

Rules:

- This endpoint returns all contacts currently linked to the user, including contacts that were originally claimed from guest history.
- Disabled contacts are still returned so Track 4 can resolve historical records correctly.
- This endpoint is the only supported way for Track 4 to resolve a user into current contact IDs for history lookup.

`POST /internal/recipients/resolve`

Request:

```json
{
  "sender_id": "uuid",
  "campaign_id": "uuid or null",
  "group_id": "uuid or null",
  "contact_ids": ["uuid"],
  "requested_channels": ["email", "sms", "telegram"],
  "evaluate_at": "RFC3339 UTC"
}
```

Response:

```json
{
  "contacts": [
    {
      "contact_id": "uuid",
      "user_id": "uuid or null",
      "candidates": [
        {
          "channel": "email",
          "value": "string",
          "allowed": true,
          "blocked_reason": "string or null"
        }
      ]
    }
  ]
}
```

Rules:

- Track 2 evaluates preference and unsubscribe rules per requested channel.
- Track 2 does not choose a single final channel for Track 4.
- Track 4 creates one concrete notification per allowed contact-channel candidate.
- Track 2 owns the semantics of `allowed` and `blocked_reason`.

## Validation Ownership

Track 2 owns runtime validation for:

- channel-aware contact format
- duplicate contacts
- quiet hour values
- unsubscribe token validity
- guest contact claim validity

Unsubscribe verification must follow the frozen Track 0 token contract exactly.

Unsubscribe verification must follow the frozen Track 0 token contract exactly.

Track 2 must not push validation decisions for these rules into Track 4.

## Auth and Audit Ownership

Track 2 owns:

- recipient-facing authorization
- administrator bootstrap in auth domain
- audit records for user, contact, preference, unsubscribe, and claim actions

## Change Control

Track 2 cannot change without coordination:

- JWT claim shape
- `POST /internal/recipients/resolve` contract
- meaning of unsubscribe scopes
- guest claim semantics

## Parallel Safety Criteria

Parallel work is safe when:

- Track 2 only writes `recipients.*`
- Track 4 consumes only the frozen internal resolve contract
- no other track invents recipient preference logic
