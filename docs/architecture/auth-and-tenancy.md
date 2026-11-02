# Authentication and Organization Scope

This is the current identity, effective Organization and authorization contract. Product login behavior follows the [official ZITADEL Login / Auth.js decision](../product/official-zitadel-login.md); domain placement follows [Project Boundaries](project-boundaries.md).

## Owners and call path

| Responsibility | Current owner |
| --- | --- |
| Authenticated identity values | `internal/authidentity` |
| Official issuer, introspection, user and grant adapters | `internal/authruntime/zitadel` |
| Effective Organization and bounded grant cache | `internal/workbenchcontext` |
| Module/role policy decisions | `internal/authz`, existing scoped policy and module permission catalog |
| Browser session and same-origin BFF | `web/listingkit-ui`, Auth.js and server-side authorization helpers |
| Route assembly | owning domain `httpapi` module, composed by `internal/app/httpapi` |

Browser session -> trusted server bearer -> official identity verification -> verified effective Organization / membership / project grant -> scoped policy -> owning domain handler.

The existing authorization implementation includes `authz.ListingKitAuthorizer`; internal engineering names do not change its current scoped policy obligations.

## Identity and Organization invariants

- The official issuer owns authentication and subject identity. Local business code does not substitute its own password, consent or identity authority.
- Home Organization and effective business Organization are separate facts. A selected Organization is only a candidate until the current verified grant establishes access.
- Browser headers, cookies containing a candidate scope, role names and request DTOs do not grant authority by themselves. Never accept a caller-provided numeric tenant fallback.
- Bind the authenticated subject, configured project, effective Organization and current membership. One Organization's administrator grant cannot become another Organization's permission.
- Resource ownership remains with the original domain. A project association, Store row, report link or cache hit cannot bypass that domain's read/write permission.
- Missing identity/scope, removed membership and unsupported authorization data fail closed. Keep route-specific error, timeout and response-size contracts.

## Freshness and switching

Cached reads use the existing grant cache with a maximum 60-second lifetime, bounded by token/grant expiry and the verified subject/project/scope identity. Do not cache bearer tokens as business facts.

Live writes and live organization switches bypass cached grants. Rejected or failed switches clear Organization-scoped state and dependent client queries. Revocation prevents new mutations/dispatch; recovery of an existing operation follows its original authorized owner and immutable receipts rather than creating a fresh mutation.

Fresh authorization before an external effect is not an atomic transaction with that provider. Each existing dispatch/effect contract defines its final fence, UNKNOWN and recovery behavior; this map does not invent a universal lifecycle gate.

## Current consumer contracts

- [Account read contract](../engineering/account-readonly-contract.md)
- [Membership contract](../engineering/account-membership-contract.md)
- [Official provider boundary](../engineering/zitadel-official-provider-boundary.md)
- [Enterprise role management](enterprise-custom-roles.md)

Current source tests cover scope isolation, forged target/header rejection, live-write freshness, organization-switch cache invalidation and original-owner permission checks. Documentation is not evidence that a particular retained instance or real provider has been exercised.
