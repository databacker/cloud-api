# cloud-api

This repository describes the northbound API for the cloud service. It provides a set of endpoints that allow users to interact with the cloud service.

This does not cover the open source API between the cloud service and individual
backup engines. Those APIs are documented in
[databacker api](https://github.com/databacker/api).

## API Specification

The API is a RESTful API, with JSON payloads.
The API is defined in this repository in [OpenAPI 3.0 spec](https://github.com/OAI/OpenAPI-Specification) format.
It is used to generate bindings for a controller implementation and any
clients.

The API specification includes the following sections:

* administration (by end-users)

### Protocol

The protocol is RESTful, with json payloads over HTTP, secured by TLS.
json payloads are used everywhere, except for streaming large amounts of data,
for example logs.

### Authentication & Authorization

API requests must be authenticated.
[Json Web Tokens (JWT)](https://jwt.io/), defined in [RFC7519](https://tools.ietf.org/html/rfc7519)
are used for validating and authorizing requests.

The JWTs are issued via one of two methods:

* OAuth2 Authorization Code Grant, for end-users (administration, this repository)
* OAuth2 Client Flow, for backup engines (reporting, orchestration, [api repository](https://github.com/databacker/api))

#### OAuth2 Client Flow

Backup engines use the [OAuth2 Client Flow](https://datatracker.ietf.org/doc/html/rfc6749#section-4.4).

The backup engine authenticates using its affiliated ECDSA public key, using it to generate a JWT,
which is submitted for all future requests. Once the JWT expires, the backup engine must re-authenticate.

## Product Model

The northbound administration API now exposes two explicit concepts:

* Backup engines manage registration, setup, config, lifecycle, and deletion.
* Databases are the protected resources users review for backup state, failures, logs, and traces.

The API uses `databases` rather than `protected-resources` because the current product and telemetry
schemas are database-specific. If non-database protected resources are added later, introduce a new
resource family or a typed protected-resource abstraction then.

Configs belong to backup engines. Backups, logs, and traces belong to databases, with `engine_id`
available on event payloads for correlation.

Billing uses a separate payer boundary:

* Accounts/workspaces own operational resources and access control.
* A `billing_customer` is the payer/payment-profile boundary and may eventually pay for more than one account.
* Signup/default-account creation should associate the account with a dormant billing customer so users can explore before provider payment setup.
* The base billable product is an active protected database. Engines, backup attempts, logs, traces, failures, stale health, and backup success/failure are operational records and do not directly determine base billing.
* Payment methods and provider/MoR commercial details are managed only through `/admin/billing/customers/{billing_customer}/portal-session`. The API never accepts raw card or bank credentials.
* Payment-based billing readiness is established only by verified provider/MoR confirmation through webhook/callback processing or server-side provider lookup, not by browser redirect results.
* Entitlement intervals retain the consuming account, billed customer, database identity, product/price version, lifecycle reasons, and active interval timestamps. Historical payer attribution must not be rewritten by any future account reassignment.
* Databases remain billable until explicitly deactivated. Reactivation is explicit, paid, requires billing readiness, requires charge acknowledgement, and opens a new entitlement interval.

### Endpoints

This API spec does not delineate URL endpoints, or distinguish between URL endpoints for various
purposes. The entire spec can be implemented in a single endpoint, or split among multiple.

Routes in the API are global; a single API scheme is defined for the entire API, independent of
endpoints. Separate endpoints may choose to implement different subsets of the API.

An endpoint that implements only a subset of the API, upon being queried for a route
that is not implemented may return one of the following:

* `301 Moved Permanently` with a `Location` header to the correct endpoint, if it knows of the appropriate location.
* `410 Gone` if it does not know of the appropriate location. This indicates that it is a valid part of the API, but this server does not implement it.
* `404 Not Found` if this is not a known valid part of the API.

### Versioning

The API _as a whole_ is not versioned, e.g. `https://endpoint/v1/` and `https://endpoint/v2/`.
While this may not be possible permanently, this specification shall attempt to avoid it as long
as possible.

As this API is as closely REST compatible as possible, all resources are permanent endpoints,
e.g. `/config/{engine}` and `/report/{engine}`. New resources will be released at new endpoints.

Specific versions of individual resources are versioned via HTTP headers.
The `Accept` header is used to specify the version of the resource requested.
Each endpoint that represents a resource has one or more specific media-type which it returns.

Generic media-types, such as `application/json` are not used. Instead, each resource has a specific
media-type, e.g. `application/vnd.databack.device-config.v1+json`.

The version is specified in the media-type, and the format is specified in the extension, e.g. `+json`.

* If the format is not specified, the default is `json`.
* If the version is not specified, the default is the highest one available.
* If no media-type is provided, the default is the highest version of the default format.

New endpoints or new versions of individual resources require new media types and knowledge of the
new paths, and therefore require new databacker versions. However:

* all new endpoints should be backwards compatible with previous versions of databacker, which will be unaware of them
* some new versions of existing resources should be backwards compatible with previous versions of databacker, when they simply add new fields, that should be ignored by older versions of databacker
* other new versions of existing resources may not be backwards compatible with previous versions of databacker, when they change the meaning of existing fields, or remove fields; new media-types should be used for these

## API philosophy

The API endpoints are all designed to be as RESTful as possible, with the following principles.

All endpoints begin with `/admin/`, to distinguish from other endpoint domains that may be used for other purposes.

All resources include two paths to the same resource: one that does not include the account name, and one that does.
Using the path without the account name is optional, and will default to the equivalent path with your login account.
Users with multiple accounts may use the account-specific paths to access resources in other accounts they have access to.

In all cases, your authenticated user will be checked for access to the account before giving access to resources.

For example, if my default account is `123`, then the following account-scoped resources are available:

* `/admin/accounts/123/engines`
* `/admin/accounts/123/databases`
* `/admin/accounts/123/databases/{database}/backups`
* `/admin/accounts/123/databases/{database}/logs`
* `/admin/accounts/123/databases/{database}/traces`
* `/admin/accounts/123/engines/{engine}/configs`

The old northbound `/admin/accounts/{account}/instances` resource family has been removed as a
breaking API change. Frontends should replace instance list/detail/config routes with engine routes,
and replace instance backup/log/trace routes with database routes.

## Frontend Migration Notes

* Replace `/admin/accounts/{account}/instances` with `/admin/accounts/{account}/engines`.
* Render engine rows from `EngineSummary`; the list response now returns summary objects instead of IDs.
* Replace instance-scoped backups/logs/traces with `/admin/accounts/{account}/databases/{database}/...`.
* Render database rows from `DatabaseSummary`, including latest status, success/failure timestamps, and latest error.
* Replace `instance_id` and `instance_name` fields in northbound payloads with `engine_id`, `engine_name`,
  `database_id`, and `database_name` as appropriate.
* Replace `latest_per_instance` query parameters with `latest_per_database`.
* Use `protection_lifecycle` on database rows/detail for billable lifecycle state. Continue to use backup status, logs, and traces for operational health only.
* Use account billing routes for workspace-attributed views and billing-customer routes for payer-wide views.
* Regenerate consuming clients from `src/api.yaml` with `make sdk`; generated Go bindings are under `go/api` and TypeScript definitions are under `ts/api.d.ts`.

## Backend Seed Contract

This repository contains the API contract and generated bindings, not backend seed scripts. Backend
implementations should seed deterministic frontend integration data with:

* one admin-capable user, one account owner, and one member-only user
* one account with multiple backup engines and multiple databases
* at least one engine config
* at least one successful backup and one failed backup
* logs and traces attached to backups/databases
* stable IDs and timestamps documented by the backend repository's seed command
