# Track 5 Specification: Frontend

## Purpose

Track 5 owns the full web UI for recipients and source operators. It must consume frozen API contracts and keep all user-facing text in Russian only.

## Start Gate

Starts after:

- Track 0 contract freeze
- Track 1 platform baseline

The frontend may use mocks and stubs first, but it must not invent backend payloads that contradict frozen contracts.

## Owns

- Login UI
- Recipient cabinet UI
- Contacts UI
- Preferences UI
- Quiet hours UI
- Guest unsubscribe UI
- Guest-to-user upgrade UX
- Operator sender, template, group, and campaign UI
- History UI
- Operator history UI
- Per-sender analytics UI
- Frontend route protection
- Frontend-side API client layer
- Russian-only UI validation

## Does Not Own

- Business API definitions
- Any backend database schema
- Any business queue payload

## Route Ownership

Owned routes:

- `/login`
- `/app/profile`
- `/app/contacts`
- `/app/preferences`
- `/app/history`
- `/operator/templates`
- `/operator/groups`
- `/operator/campaigns`
- `/operator/history`
- `/operator/analytics`
- `/unsubscribe`

## API Contracts Consumed

Track 5 consumes and must not redefine:

- recipient public APIs from Track 2
- source public APIs from Track 3
- notification public APIs from Track 4

Rules:

- API clients must match frozen request and response fields.
- If a required backend field is missing, Track 5 must report a contract gap instead of inventing it.

## UI Language Rules

- All user-facing UI text must be Russian only.
- No English labels, buttons, form messages, empty states, or route titles are allowed in the final product.
- Technical identifiers in code may be English, but rendered UI may not.

## Auth Rules

- Interactive API calls use JWT from Track 2.
- Protected routes must enforce role-aware access in the UI layer.
- Sender access decisions still belong to backend services; frontend must not hardcode sender permissions.

## Data Ownership

Track 5 owns no backend schema.

Local frontend state is allowed but must not become a source of truth for:

- recipient preferences
- sender permissions
- notification statuses
- analytics data

## Validation Ownership

Track 5 owns:

- form validation aligned with backend contracts
- frontend-side route protection
- Russian-only UI checks

Track 5 does not own final business validation. Backend remains authoritative.

## Integration Rules

Track 5 must integrate with:

- Track 2 for auth, contacts, preferences, and unsubscribe flows
- Track 3 for operator setup, templates, groups, and campaigns
- Track 4 for history, send results, and analytics

Operator history contract:

- Track 5 must use `GET /api/notifications/history?mode=operator&sender_id=<uuid>&campaign_id=<uuid or null>&status=<status or null>&page=<n>&size=<n>` for sender-scoped operator history.

Unsubscribe route contract:

- `/unsubscribe` consumes `token` and `available_scopes` from query parameters.
- Track 5 treats `token` as opaque.
- Track 5 must not generate, decode, or reinterpret unsubscribe token payloads.
- If `available_scopes` indicates both sender and campaign-or-group choices, the UI must offer both choices in Russian and submit the chosen one to `POST /api/recipients/unsubscribe`.

## Change Control

Track 5 cannot change without coordination:

- public API payloads
- JWT claim expectations
- role names
- meaning of history and analytics fields

## Parallel Safety Criteria

Parallel work is safe when:

- Track 5 works only from frozen contracts and mocks
- Track 5 owns no backend schema or queue fields
- Track 5 reports contract mismatches instead of patching around them locally
