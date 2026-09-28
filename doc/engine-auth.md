# Engine registration proof

This document defines the additional proof required by
`POST /admin/accounts/{account}/engines`. It does not change northbound user
authentication. Cloud first authenticates and authorizes the user normally,
then verifies this proof before creating the engine.

Registration is immediate. A valid request returns `201 Created`; there is no
pending challenge, registration-completion endpoint, or server-issued key ID.

## Submitted keys and deterministic fingerprints

The request body supplies an Ed25519 authentication public key and an X25519
configuration-encryption public key, each as exactly 32 raw bytes encoded with
strict standard padded base64 and accompanied by a positive uint64 generation.

Cloud computes both key IDs itself. A key ID is a deterministic fingerprint,
not an arbitrary database identifier. For a raw public key and generation, the
digest inputs are exactly:

```text
"databacker/engine-key-id/authentication/ed25519/v1\x00"
|| uint64_be(generation)
|| raw_ed25519_public_key

"databacker/engine-key-id/configuration-encryption/x25519/v1\x00"
|| uint64_be(generation)
|| raw_x25519_public_key
```

The canonical IDs are `auth:` or `config:` followed by the complete SHA-256
digest as 64 lowercase hexadecimal characters. Cloud stores each ID with the
corresponding public key and engine. Authentication key IDs MUST be globally
unique. The signed request binds the submitted configuration-encryption key to
the proven authentication key, but an Ed25519 signature is not proof of
possession of the X25519 private key. This profile therefore does not use a
global X25519-key uniqueness check as a substitute for such proof.

## Signed registration request

The registration request uses RFC 9421 HTTP Message Signatures with Ed25519 in
addition to ordinary user authentication. It uses the single signature label
`databacker-engine` and these required parameters:

- `created`: integer Unix time;
- `expires`: integer Unix time after `created` and no more than 300 seconds
  later;
- `keyid`: the deterministic `auth:` fingerprint computed from the submitted
  authentication public key and generation;
- `nonce`: exactly 16 random bytes encoded as unpadded base64url;
- `tag`: exactly `databacker-engine-registration-v1`.

The covered components, in this exact order, are:

```text
"@method" "@authority" "@path" "@query"
"content-digest";sf "content-type" "idempotency-key"
```

RFC 9421 appends `@signature-params` to the signature base. `@path` binds the
proof to the target account. The request body binds the proof to the submitted
name, description, and both public keys. `Content-Digest` is an RFC 9530
Dictionary containing exactly the SHA-256 digest of the transmitted JSON body:

```text
Content-Digest: sha-256=:STANDARD_BASE64_DIGEST:
```

`Idempotency-Key` is a UUID retained across retries of the same logical
registration. The signature nonce and timestamps are fresh for each HTTP
attempt. Request-target canonicalization, Structured Fields serialization,
clock tolerances, duplicate-field rejection, and other verification rules are
the same as the Databacker engine HTTP-signature profile.

After verifying each HTTP attempt, Cloud applies the idempotency key before the
global authentication-key uniqueness check. Retrying the same logical request
with the same idempotency key and body returns the original successful result;
reusing that idempotency key with a different body is a conflict. A different
registration request for an already-registered authentication key returns
`409`.

## Verification

Cloud does not look up the authentication public key when verifying a new
registration. It instead:

1. validates and decodes the submitted keys and generations;
2. recomputes both deterministic key IDs;
3. requires `Signature-Input.keyid` to equal the computed authentication key
   ID;
4. verifies `Content-Digest` and reconstructs the RFC 9421 signature base;
5. verifies `Signature` using the submitted Ed25519 public key;
6. rejects expired, future, replayed, malformed, or ambiguous proofs;
7. resolves an exact idempotent retry to its original result;
8. rejects the authentication key ID if it is otherwise already registered;
   and
9. creates the engine and both key records atomically.

A malformed or invalid proof returns `400`. An authentication key already
associated with an engine returns `409`. Error text must not disclose the
account or engine that already owns the conflicting key.

When PATCH submits unchanged keys, normal user authentication is sufficient.
If key material changes, Cloud requires a new proof by the submitted
authentication key using the same profile and a body/path for that PATCH.
