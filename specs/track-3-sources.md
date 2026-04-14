# Track 3 Specification: Sources

## Purpose

Track 3 owns sender-side business entities and permissions: senders, operators, integration credentials, templates, groups, campaigns, and recurrence rules.

## Start Gate

Starts after:

- Track 0 contract freeze
- Track 1 platform baseline

## Owns

- Sender entities
- Source operators
- Operator-to-sender assignment
- Sender integration credentials
- Templates
- Contact groups and membership
- Campaigns
- Recurrence rules
- Template lookup APIs
- Due campaign lookup APIs
- Credential resolution API for source-system auth

## Does Not Own

- Interactive auth token issuance
- Recipient preferences
- Canonical notification lifecycle
- Delivery attempts

## Schema Ownership

Owned schema: `sources`

Owned tables:

- `senders`
- `sender_operators`
- `sender_credentials`
- `templates`
- `contact_groups`
- `contact_group_members`
- `campaigns`
- `recurrence_rules`
- `audit_log`

Track 3 must not write to:

- `recipients.*`
- `notifications.*`
- `delivery.*`

## Public API Ownership

Owned public endpoints:

- `GET /api/sources/senders`
- `POST /api/sources/senders`
- `POST /api/sources/senders/{sender_id}/operators`
- `POST /api/sources/senders/{sender_id}/credentials`
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

### Minimum Request/Response Freeze

`POST /api/sources/templates`

Request:

```json
{
  "sender_id": "uuid",
  "name": "string",
  "channel": "email|sms|telegram",
  "subject": "string or null",
  "body": "string",
  "variables": ["name", "code"]
}
```

`POST /api/sources/campaigns`

Request:

```json
{
  "sender_id": "uuid",
  "group_id": "uuid",
  "template_id": "uuid",
  "name": "string",
  "scheduled_at": "RFC3339 UTC or null",
  "channels": ["email"],
  "recurrence_rule": {
    "kind": "once|daily|weekly|cron",
    "value": "string or null"
  }
}
```

Channel-template rule:

- A template is single-channel and owns exactly one `channel` value.
- The current frozen contract allows only one effective channel per campaign or direct send request.
- Therefore the `channels` array must currently contain exactly one value, and it must match the selected template `channel`.
- Multi-channel delivery is achieved by materializing separate notification records per channel/template pair, not by treating one template as channel-agnostic.

## Internal API Ownership

Owned internal endpoints:

- `GET /internal/sources/templates/{template_id}`
- `GET /internal/sources/campaigns/{campaign_id}`
- `GET /internal/sources/campaigns/due`
- `GET /internal/sources/groups/{group_id}/members`
- `GET /internal/sources/senders/{sender_id}/operators/{user_id}`
- `POST /internal/sources/senders/resolve-credential`

`GET /internal/sources/templates/{template_id}`

Response:

```json
{
  "template_id": "uuid",
  "sender_id": "uuid",
  "channel": "email|sms|telegram",
  "subject": "string or null",
  "body": "string",
  "variables": ["name", "code"],
  "active": true
}
```

`GET /internal/sources/campaigns/{campaign_id}`

Response:

```json
{
  "campaign_id": "uuid",
  "sender_id": "uuid",
  "group_id": "uuid",
  "template_id": "uuid",
  "channels": ["email"],
  "scheduled_at": "RFC3339 UTC or null",
  "active": true
}
```

`GET /internal/sources/campaigns/due`

Request:

```text
GET /internal/sources/campaigns/due?as_of=<RFC3339 UTC>&limit=<n>&cursor=<string or null>
```

Response:

```json
{
  "campaigns": [
    {
      "occurrence_id": "string",
      "campaign_id": "uuid",
      "sender_id": "uuid",
      "group_id": "uuid",
      "template_id": "uuid",
      "channels": ["email"],
      "scheduled_at": "RFC3339 UTC"
    }
  ],
  "next_cursor": "string or null"
}
```

Due-campaign polling rule:

- This endpoint is read-only and non-claiming.
- `as_of` is required.
- `limit` is required.
- `cursor` is optional.
- `next_cursor` is required in the response and is null when there are no more results.
- `occurrence_id` uniquely identifies one due occurrence of one campaign.
- The same due occurrence must return the same `occurrence_id` across repeated polls until it is no longer due.
- `scheduled_at` is the occurrence time used for materialization.
- Deduplication and materialization claiming are owned by Track 4 in the `notifications` schema, not by this endpoint.

`GET /internal/sources/senders/{sender_id}/operators/{user_id}`

Response:

```json
{
  "allowed": true,
  "role": "source_operator|admin"
}
```

Authorization rule:

- `allowed=true` means the user may access sender-scoped operator history, analytics, templates, groups, and campaigns for that sender.
- Track 3 owns the semantics of this permission decision.

`GET /internal/sources/groups/{group_id}/members`

Response:

```json
{
  "group_id": "uuid",
  "contact_ids": ["uuid"]
}
```

`POST /internal/sources/senders/resolve-credential`

Request:

```json
{
  "integration_key": "string"
}
```

Response:

```json
{
  "sender_id": "uuid",
  "credential_id": "uuid",
  "active": true
}
```

Group membership exposure rule:

- Track 3 owns group membership as source-side truth.
- For pipeline use, Track 3 exposes recipient targeting through `GET /internal/sources/groups/{group_id}/members`, returning stable `contact_id` lists associated with a `group_id`.
- Track 3 must not expose internal group membership in a shape that requires Track 4 to infer recipient identifiers from non-frozen fields.
- Group membership is resolved live at execution time. Scheduled and recurring campaigns use current group membership when Track 4 asks for members, not a membership snapshot captured at campaign creation time.

## Validation Ownership

Track 3 owns runtime validation for:

- sender credential state
- template variables
- campaign input
- group membership input
- recurrence rule input

Track 3 owns what a valid template and campaign mean.

## Auth and Audit Ownership

Track 3 owns:

- source-side endpoint authorization
- operator-to-sender assignment enforcement
- audit records for sender, template, group, campaign, and credential changes

Track 3 must not issue JWTs.

## Change Control

Track 3 cannot change without coordination:

- credential resolution contract
- template lookup contract
- due campaign lookup contract
- sender-to-operator permission semantics

## Parallel Safety Criteria

Parallel work is safe when:

- Track 3 only writes `sources.*`
- Track 4 uses only frozen lookup and credential resolution contracts
- no other track invents template or campaign semantics
