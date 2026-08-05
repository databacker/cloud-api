# Account audit contract and mutation convergence

The account audit endpoints expose an authorized account view of Cloud's shared audit system of
record. They do not create a second account-owned event store. Cloud evaluates the caller's current
account authorization on every request; the first version requires the current `owner` role because
the account permission model has no explicit `view_audit` permission. Independently, an event belongs
in an account stream only when its immutable, write-time `account_ids` snapshot contains that account.

This distinction preserves operator, automation, provider, service, migration, webhook, and
background-job changes in the affected owner's history. Event detail uses the same `404` response for
an unknown event and an event outside the requested account view, preventing ID probing from exposing
cross-account information.

Events are immutable and append-only. Event and actor fields contain only allowlisted, redacted data;
credentials, tokens, secrets, raw provider payloads, and unsafe upstream errors are never exposed.
Times are UTC RFC 3339 strings. List order is newest first and pagination cursors are opaque: clients
must pass cursors back unchanged and must not derive meaning from them.

## Backward-compatible mutation convergence

Existing account-facing mutations currently return several established shapes: a resource object
(`Account`, `Engine`, `Database`, billing and policy resources), an `ID`, a provider-session result,
or an empty `200`/`201`/`204`. Replacing these payloads with a new envelope would break generated Go
and TypeScript clients and is therefore out of scope for this change.

Convergence should be additive and staged:

1. Define shared `ActionStatus` values (`queued`, `running`, `succeeded`, `failed`, `cancelled`, and
   `already_complete`) and an `ActionResult` shape containing required `action_id` and `status`, plus
   optional `resource_id`.
2. For mutation responses that already return JSON objects, add optional `action_id` and `status`
   properties to the existing response schema first. This preserves old readers while regenerated
   clients can adopt the standard correlation fields.
3. For empty-body success responses, introduce a new success media-type/version or an explicitly
   negotiated response before changing the body. Do not change an existing `204` into a JSON response
   in place.
4. Keep legacy fields such as `action_audit_id`, where any exist in deployed contracts, as deprecated
   aliases during migration. New mutations must use `action_id`; implementations should return the
   same value in both fields while an alias remains.
5. Allocate `action_id` before work begins and return the original result on idempotent replay. The
   response `status` describes the logical action, while HTTP status continues to describe this
   request.

This proposal intentionally adds no system-wide query, export, stream, aggregation, or audit-record
mutation/deletion surface.
