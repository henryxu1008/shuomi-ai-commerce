# Auth.js instance cookie isolation — #706

Status: IMPLEMENTATION_READY. The original independent reviewer approved design
`d0ea048c6fe8ed4d7ab5551976897fd4ab474089` before production modification, with no
Architecture BLOCKER. Native regression tests below close IMPLEMENTATION_TEST
items; this does not authorize deployment or claim user acceptance.

## Product outcome and authority

The retained-instance user must complete ordinary OIDC sign-in without another
trusted local instance overwriting its encrypted session or authorization checks.
The user explicitly requested a root-cause fix after the first sign-in failed,
with retained data and one existing source Writer / instance owner.

Authority: #706 and #137; the current Identity / Organization contract linked by
[auth-and-tenancy](auth-and-tenancy.md); [current Product/Figma Authority](../product/final-ui-ia-authority.md),
page 31:463. This changes no screen, navigation, naming, or visible login action.

In scope: Auth.js's seven native cookie names and their common configuration
consumer. Out of scope: identity/Organization policy, permissions, effective-
Organization selector cookies, session lifetime/refresh, new IAM or provider
operations, business tables, a session service, a general test platform, and
parallel authorization attempts within the *same* instance.

## Evidence and cause boundary

The first retained UI logged `JWTSessionError` with `no matching decryption
secret`, then `InvalidCheck: pkceCodeVerifier value could not be parsed`.
The original instance owner compared frontend credential fingerprints with the
cold backup: auth secret, OIDC client ID/secret and project ID were unchanged.
Ordinary re-entry subsequently succeeded. This is a workaround, not a fix.

Current `buildAuthConfig()` supplies no cookie names. Installed @auth/core 0.41.3
uses host-wide `authjs.*` names, secure prefixes when HTTPS, and Path `/`.
[RFC6265 §8.5](https://www.rfc-editor.org/rfc/rfc6265#section-8.5) explicitly says
cookies for a host are shared across ports. Distinct localhost instances with
different secrets therefore overwrite or mix the same browser cookie slots.

A nonformal in-memory spike uses the installed Auth.js implementation and jsdom
CookieJar, two localhost ports, and synthetic secrets. Both native session
decoding and PKCE succeed before the second instance writes its defaults; after
that write, native Auth.js logs the exact `JWTSessionError` and PKCE `InvalidCheck`
above. It uses no network, real credentials, browser cookies or business data.
This establishes a reproducible configuration defect. It does not establish
which historical foreign cookie or tab caused the *first* incident: original
cookie values were not captured and must not be fabricated or extracted now.

## Contract → implementation → injection → consumer

1. Reuse the existing configured public-origin resolution. Auth.js `AUTH_URL`,
   when supplied, is the canonical authority for this Auth.js instance; otherwise
   use the existing `resolvePublicAppOrigin()` behavior, including its current
   localhost development default. There is no new configuration flag or legacy
   origin fallback. If needed, EXTRACT that existing pure resolver to a shared
   module; its current callers keep the same behavior.
2. Parse the selected URL and take its canonical HTTP(S) `origin` (scheme, host,
   effective port); reject unsupported schemes, credentials, or malformed URLs.
   Use the existing normalized OIDC issuer/client ID when configured, empty
   strings when unconfigured. No secret, user, Organization, request header or
   business identifier determines the namespace.
3. Hash the tuple `[origin, issuer, clientID]` with native SHA-256, retaining
   128 bits as 32 hexadecimal characters. This is a stable *name*, not an
   authorization mechanism or a secret. Restarting the same deployment produces
   identical names; different configured ports or OIDC applications differ.
4. Use the supported [Auth.js cookies option](https://authjs.dev/reference/core#cookies)
   to supply names for sessionToken, callbackUrl, csrfToken, pkceCodeVerifier,
   state, nonce and webauthnChallenge. Example form:
   `__Secure-shuomi.<namespace>.session-token`; HTTPS CSRF uses `__Host-`.
   HTTP development names have no secure prefix. Do not import SDK private
   cookie internals into production or implement a cipher/cookie store.
5. Set native `useSecureCookies` from that trusted canonical origin. Supply only
   the cookie names; native options retain HttpOnly, SameSite, Secure, Path,
   expiration, chunking and clearing. This cannot downgrade HTTPS from an
   untrusted request header.
6. Both `buildAuthConfig()` and `buildServerAuthConfig()` consume that one
   configuration. Existing exported handlers/signIn/signOut/serverAuth and BFF
   consumers continue to use native Auth.js. There are no production manual
   `getToken`/default-name parsers to leave behind. Check those sibling paths.

## Ownership, security and persistence invariants

- Auth.js remains owner of encrypted browser session and checks; ZITADEL remains
  identity authority; Go still verifies identity and Organization permissions.
- Signing/decryption secret, provider, redirect allowlist, PKCE/CSRF/state/nonce,
  token refresh and session/role/tenant contracts remain unchanged.
- New names are never filled by copying a legacy/default or another instance's
  cookie. Unknown/wrong-secret/invalid checks continue to fail closed.
- Shared-host deployments are trusted local applications. Cookie names prevent
  accidental collision, not a malicious same-host server reading cookies. No
  new isolation/security claim or IAM policy is introduced.
- Replica configuration must agree on canonical origin, OIDC application and
  secret as before. Do not hide secret drift by hashing the secret into the name
  or adding a decryption fallback.
- The existing Organization selector is only a server-revalidated selector; its
  namespace is out of scope. Do not expand into a tenant/session framework.

## Cutover, concurrency and failure

Legacy decision: RETIRE global default-cookie consumption. New code does not
read, migrate, dual-read, copy or broadly delete those cookies. Existing old
browser sessions need one ordinary sign-in after cutover; this is a consequence
of the fixed names, not the root-cause workaround. Other local applications'
cookies remain untouched. Valid business data and original runtime secrets do
not change.

No durable state machine, database, transaction, external mutation, retry or
UNKNOWN owner changes. Native Auth.js still consumes/deletes its own one-use
checks and chunks under the selected namespace. Wrong canonical configuration
fails rather than trying another cookie set. Ordinary failed token/code exchange
is not automatically retried by a new mechanism.

The original instance owner alone handles any later approved UI deployment and
browser verification. Keep backend, worker, volumes, backups, credentials,
existing product/drafts and the saved recovery R1. No merge/deploy/data action
is granted by this design document.

## Bounded validation and handoff

- TDD RED on the current configuration reproduces collisions using native
  Auth.js and an in-memory standards-based cookie jar. GREEN must demonstrate
  interleaved writes from two origins/two synthetic secrets preserve each
  session and PKCE check, not merely compare generated names.
- Check deterministic same-config names, different port/client separation,
  native HTTPS/HTTP flags, and the common public/server config consumer.
- Seed foreign legacy/default cookies; verify they are not consumed or cleared.
  Keep invalid/wrong-secret failure closed. Cover chunked native session handling
  if the practical session size crosses native cookie size.
- Run only relevant existing auth/login/logout/session tests and typecheck,
  then accurate stable-head CI and the original independent final reviewer.
- The original runtime/browser follow-up, if authorized for that final source,
  must show normal login with foreign default cookies left intact, then current
  session access. This is distinct from controlled fixture evidence and user
  acceptance. Do not mutate IAM or unrelated applications to create a repro.

Independent reviewer: existing `content_final_review`, read-only, only this
increment. No new Writer/Reviewer, browser instance, or acceptance platform.
