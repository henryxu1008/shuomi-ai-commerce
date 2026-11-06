# Fixed HTTPS identity origin for the cloud trial

Issue: [#1085](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1085).
Design Basis: **Independent Architecture**. Admission: **IMPLEMENTATION_READY**.

Independent architecture check: `public_identity_review`, 2026-11-04; no BLOCKER.
One IMPLEMENTATION_TEST finding requires focused foreign-endpoint and untrusted
TLS preflight tests. Production implementation follows this committed design.

## Product authority and outcome

Henry explicitly authorized a new empty Alibaba ECS colleague trial with
`trial.shuomiai.com`, `auth-trial.shuomiai.com`, HTTPS, as many current modules as
possible, a CNY100 budget and a data-preserving stop 24 hours after creation.
The existing [official Login decision](../product/official-zitadel-login.md),
[identity and Organization contract](auth-and-tenancy.md),
[official provider boundary](../engineering/zitadel-official-provider-boundary.md)
and [greenfield baseline](../product/greenfield-no-legacy-migration.md) apply.
No UI, page naming or interaction changes are proposed; existing product/Figma
authority remains [the current authority](../product/final-ui-ia-authority.md).

The actual new backend stopped before serving with
`identity.issuerURL must be a bounded absolute loopback HTTP(S) URL`.
Configuration-only reuse was insufficient. This is a trial BLOCKER because the
normal authenticated happy path cannot run. Stop affected production edits until
this independent design is reviewed and IMPLEMENTATION_READY.

## Bounded change and threat model

The private deployment manifest is operator-controlled input, never browser,
tenant, header, cookie, request, remote metadata or environment input. Its issuer
is already the explicit trust anchor. Extend only current-application's identity
endpoint validation to accept one canonical HTTPS DNS origin on port 443.
Both issuerURL and authorizationAPIURL must equal that origin exactly for the
public case. No path, explicit port, trailing slash, credentials, query, fragment,
opaque form, IP literal, localhost, mixed case, whitespace or invalid DNS labels.
Use the same bounded DNS grammar as the reviewed deployment-origin validator.
Existing explicit-port loopback HTTP(S) configurations keep their existing
validation and same-origin behavior.

This does not accept user-selected identity providers, arbitrary URL forwarding,
untrusted discovery authorities, TLS bypasses, identity fallbacks or a new IAM.
The threat model includes forged scopes, mismatched issuer/API, HTTP downgrade,
malformed origins, secret exposure and public access to private services.
Compromise of the operator-controlled private manifest, DNS account or official
issuer is outside this bounded change; no new accepted product risk is asserted.

## Contract, owner, injection and consumers

Browser -> fixed Caddy HTTPS host -> existing Traefik -> Auth.js/BFF -> official
ZITADEL. Existing `init.sh` already writes the two configured public origins into
the private manifest; native module/Chat composition preserves those identity
fields. No generated manifest or installer state needs mutation for this fix.
`currentapplication.LoadConfig/validate` admits the configured identity origin;
`CoreConfig` injects it into the existing `internal/authruntime/zitadel` verifier
and native authorization clients. Existing identity preflight checks the exact
discovery issuer and requires authorization/token/userinfo/introspection endpoints
to use the admitted origin before listeners or owner databases are opened.
Runtime configuration and HTTP Membership assembly share the canonical public
origin predicate owned by `authruntime/zitadel`; the assembly's same-issuer check
continues to apply, rather than retaining an independent loopback-only guard.
Normal system/public CA validation and existing timeout/size limits remain.
The same official public issuer is used by Login V2, Auth.js, backend introspection,
native grants, referrals and membership. No host-header or local-issuer rewrite.

Identity and subject owner: official ZITADEL. Effective Organization/cache owner:
`workbenchcontext`. Authorization: existing scoped `authz` policy and domain
handlers. Browser session: existing Auth.js. No role or native grant changes.
Listeners, SQL, Temporal, Mailpit and object-storage administration remain private.

## State, persistence, effects and recovery

No new table, persisted business fact, state machine, transaction, retry key,
balance, schema owner or external mutation is added. Startup discovery remains a
read-only preflight; failure leaves the application unavailable. Existing owner
initializers/markers and private manifest hashes are retained. The new code reads
the already-generated origin, so no marker deletion, reinstallation, password
rotation or data rewriting is required. Existing business UNKNOWN/idempotency
and recovery ownership remain unchanged. Model, Store, payment, SMS and email
provider operations remain unconfigured/unexecuted.

Build the corrected backend from its exact Git source using the standard image
builder, read back the image ID on this one ECS, and replace only that project's
application container. Preserve every volume, old image and configuration.
Rollback restores the prior image and leaves the application unavailable under
the original guard; never rewrite the issuer to bypass it. ECS's approved native
OOS stop remains scheduled for 2026-11-04 22:18 UTC+8 and preserves all disks.
Legacy decision: none; this is a fresh installation with no migration/compatibility.

## Verification and acceptance

TDD: add a config test with both endpoints at the canonical public HTTPS origin;
confirm rejection before the fix, then acceptance. Test mismatched public origins,
mixed loopback/public, HTTP public origins, explicit ports, paths, trailing slash,
userinfo, query/fragment, IP/localhost aliases, invalid labels and bounded length.
Keep existing local config, listener/database isolation and schema checks.
Include the native Membership configuration in the public-identity config test;
its provider-origin consumer must use the same admitted identity validator.
Reuse existing issuer-mismatch/unavailable preflight tests and add the reviewed
coverage gaps: correct issuer with a foreign endpoint, and untrusted TLS rejection.
Run affected Go tests and required PR CI once the candidate is stable.

On ECS, verify source/image receipt, startup health, public TLS and discovery,
unauthenticated rejection and the normal Login V2 redirect/callback origins.
Private bootstrap credentials are handed to Henry without printing their values.
Actual login/product use remains Henry's acceptance; developer self-check and CI
do not constitute user or production acceptance. No production release gate,
merge permission or Issue closure is granted by this design.
