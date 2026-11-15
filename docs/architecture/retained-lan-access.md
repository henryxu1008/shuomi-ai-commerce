# Retained trial LAN access: scoped identity admission

Issue [#1300](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1300). Design Basis: Independent Architecture. Status: IMPLEMENTATION_READY. Independent boundary review `/root/lan_boundary_review` approved the pre-implementation design (SHA256 E16A067A3802C77626667D98C8FE1954BE4C4E0B2648C3BD5C5D8080D4CE17AD) on 2026-11-15, with no unresolved design BLOCKER.

## Product outcome and authority

Henry requested direct browser access to the existing retained 35444 installation from another computer on the same LAN, preserving its existing accounts, permissions, enabled functions and data. Phones are deferred. The user performs any operations rejected by the automation environment. This is an internal LAN trial, not Internet publication, product acceptance or a production release.

Current authorities: [identity and tenancy](auth-and-tenancy.md), [fixed canonical HTTPS issuer](public-trial-identity-origin.md), [manual-only synthetic source](research-existing-synthetic-read-candidate.md), [controlled local draft](controlled-local-platform-draft.md), [TLS maintenance](local-tls-revocation-maintenance.md), and the current user decision above. Figma N/A. No legacy migration or second identity/data owner.

## Observed blocker and bounded scope

The operational LAN switch retained the exact current image (9b30d86, matching its build receipt), data volumes and feature flags. The backend failed before opening business services: first, the generated Windows bind file appeared as 0777 instead of the original 0600; fixing its actual Linux mode exposed `manual research requires loopback HTTPS native identity`. Read-only source inspection found the enabled controlled-local-draft profile has the same loopback-only identity condition. The original configurations were restored and services are healthy. No DB restore, model call or publishing occurred.

This is a new runtime-proven BLOCKER: the current happy path cannot run with the approved LAN canonical issuer while preserving the two enabled restricted functions. Reopen only this identity-admission increment. Do not disable the profiles or weaken their existing synthetic-source/execution restrictions to make startup pass.

Change only `validateResearchManualOriginal` and `validateControlledLocalDraft` to consume the existing `validateIdentityURL` admission. Both continue requiring HTTPS. For a canonical DNS origin, require `identity.authorizationAPIURL == identity.issuerURL` exactly, matching the existing global identity contract. Existing explicit-port loopback HTTPS remains valid. Reject public HTTP, IP literals, custom ports, paths, credentials, query/fragment, noncanonical DNS and mismatched issuer/API. The operator manifest pins the authority; request headers, user/model input and remote metadata cannot select it.

The other isolated fixture/local-trial/title-pilot/private-draft profiles retain their existing loopback-only admission; they are not enabled in this installation and are outside this task. No global search-and-replace of loopback guards.

## Owners and invariants

Official ZITADEL remains identity/subject owner; Auth.js remains browser-session owner; existing membership/tenant and effective-organization checks remain authoritative. Contract → the existing private manifest → the two existing config validators → unchanged current-application composition → unchanged manual-research and controlled-draft consumers. Same issuer, client/project/user/organization IDs and permissions; no issuer rewriting, fallback, second verifier or cookie copying.

Manual research still requires its exact acknowledgment, frozen B000123457 scope/member/project/item/publication/version/hash binding, native directory token, current source/research owners, and exclusion of full fixture mode. Its source cannot enter AI/prepare/start/resume, transfer or publishing. Controlled draft still requires its exact scope/source/R1/separate synthetic store/fixture, current canonical Asset reader and all existing incompatible-mode exclusions; no real platform mutation, execution, paid plan, billing or resource semantic changes. Network reachability never grants an operation or broadens these bindings.

No new business table, persisted fact owner, state machine, transaction, idempotency/retry key, UNKNOWN/recovery protocol, external side effect or model/provider business call. Startup errors remain fail-closed. TLS/discovery and same-origin verification remain enabled. Only the configured origin transport changes; permission, source and execution restrictions do not.

## Operational follow-through and failure

Reuse the already reviewed two-host HTTPS/Traefik/HAProxy setup, private-LAN IP binding and exact read-only HTTP certificate/CRL routes. Before any subsequent cutover, enforce and read back the actual backend manifest mode 0600 via the existing Linux helper, retaining its private Windows ACL; never disable the startup permission guard. Validate the full enabled manifest with the candidate backend before starting workers. Build only the backend from its exact Git commit through the standard image builder; preserve the actual UI version and every unrelated capability/configuration.

Current fake-IP DNS resolves the LAN hostname to 198.18.x.x and a normal hostname request times out, while exact-IP HTTP CRL access returns 200. Resolve that client DNS path explicitly (e.g. user-owned exact hosts entries for the two names on the server/client PCs), without TLS bypass or a false DNS PASS from `--resolve`. No automatic router/firewall/proxy changes. Preserve old configs, images and scoped identity backup. Rollback restores configuration and preserves all business facts and identity audit/domain events; never overwrite the DB with a prior dump.

## Verification and acceptance

TDD first: each restricted profile accepts a valid pinned canonical HTTPS issuer/API pair; malformed/insecure/mismatched origins and all existing scope/token/fixture/owner violations remain rejected. Keep the existing loopback and unrelated isolated-profile tests unchanged. Run focused currentapplication tests and existing required checks; validate actual full retained configuration with the new image using the existing read-only inspection entry before another switch. No new acceptance framework or parallel environment.

Then verify real manifest permissions, exact image/source receipt, health, namespace DNS, ordinary Windows TLS including CRL, issuer and same-origin endpoints, anonymous rejection and existing restricted-mode flags preserved. Another computer's browser/ordinary account login remains user verification until observed. Development checks and CI do not sign product acceptance or close the issue.
