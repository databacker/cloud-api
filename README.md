# cloud-api

This repository describes the northbound API for the cloud service. It provides a set of endpoints that allow users to interact with the cloud service.

This does not cover the open source API between the cloud service and individual
databacker instances. Those APIs are documented in
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
* OAuth2 Client Flow, for databacker instances (reporting, orchestration, [api repository](https://github.com/databacker/api))

#### OAuth2 Client Flow

Databacker instances use the [OAuth2 Client Flow](https://datatracker.ietf.org/doc/html/rfc6749#section-4.4).

The databacker instance authenticates using the ECDSA public key affiliated with the instance, using
it to generate a JWT, which is submitted for all future requests. Once the JWT expires, the databacker
instance must re-authenticate.

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
e.g. `/config/{instance}` and `/report/{instance}`. New resources will be released at new endpoints.

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
