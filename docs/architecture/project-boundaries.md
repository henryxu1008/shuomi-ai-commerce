# Project Boundary Rules

## How To Use This Document

This is the repository-wide entrypoint for current ownership and dependency direction.
Use the [architecture index](README.md) for feature contracts, the [repository map](../development/repository-structure.md#current-entrypoint-map) for real code and commands, and the current execution Issue for scope and delivery status.

The product baseline is [PD-GREENFIELD-NO-LEGACY-MIGRATION-2026-09-08](../product/greenfield-no-legacy-migration.md): fresh installation and empty business data. It does not authorize legacy profile reuse, migration or cutover; only a new explicit user product decision can permit a compatibility exception.

## Current owner map

| Responsibility | Current owner and boundary |
| --- | --- |
| Executable startup | `cmd/*` parses explicit configuration and delegates to application assembly |
| Application composition | `internal/app/*` opens approved dependencies and injects ports; business rules remain in their domains |
| API route registration | owning module `internal/*/httpapi` first; `internal/app/httpapi` only for shared runtime aggregation |
| Source evidence and publication | `internal/product/sourcing`, `internal/app/productsourcing` and admitted acquisition adapters |
| Private research evidence links and human selection | `internal/product/research`; exact live projections from current source owners, no product snapshots or execution/transfer facts; [contract](market-research-selection.md) |
| Canonical product facts | `internal/product/catalog`; source evidence is not a second canonical store |
| Approved product assets | `internal/product/asset`; generated or uploaded candidates require the existing explicit approval path |
| Reviews and proposals | `internal/product/review` and `internal/product/review/proposal` |
| Listing materials and drafts | `internal/listing/materials`, `completiondraft`, `preparation` and `record` |
| Platform rules and submission | `internal/marketplace/*` and `internal/listing/submission`; one owner per external operation |
| Agent and model execution | `internal/aicapability`, `internal/commercetool`, `internal/imageagent` and the selected provider adapter |
| Business task projection | `internal/aiworkbench`; execution status remains with the original native execution owner |
| Identity and effective Organization | `internal/authidentity`, `internal/authruntime/zitadel`, `internal/workbenchcontext` and scoped authorization |
| Money | `internal/ledger/money`; accepted payment/settlement facts are not inferred from UI or resource balances |
| Resource quantity and consumption | `internal/ledger/orgresource` and `internal/integration/orgresource` |
| Commercial orders | `internal/commercial/billing`; server-owned offers and immutable quotes |
| Store facts and observation admission | `internal/storecenter`; official grants and explicit purpose remain separate from a stored Store record |
| Runtime infrastructure | `internal/platform/*`, current configuration and existing native clients |
| External protocols and persistence | `internal/integration/*`; adapter DTOs stay outside neutral domain contracts |

## Dependency direction

`cmd/* -> internal/app/* -> domain ports / owning module -> integration adapters`

- Domain HTTP packages do not import `task-processor/internal/app/httpapi`.
- `internal/app/*` is the composition root, not a landing zone for new business policy or a second domain facade.
- internal/product must not import internal/listingkit, internal/compatibility, internal/crawler, or internal/integration.
- internal/integration/acquisition/a1688/browser and internal/integration/acquisition/browsercollector produce current internal/product/sourcing SourceEnvelope values.
- Product/Catalog/Asset, current Listing records, Marketplace payloads and external transport remain distinct owners. An adapter cannot become a second fact source.
- Agent/Tool consumers use current Service/Query ports. They do not own GORM repositories, provider credentials, platform submission clients or deterministic readiness rules.
- New imports use the current owner directly. Existing source absence and dependency bans remain enforced by code; no wrapper, fallback, dual read/write or compatibility bridge is a default solution.

## Authorization and scope

Authentication identifies the actor; the verified effective Organization and current membership/grant determine business scope. Never derive authority from browser-supplied tenant/role headers, opaque product metadata or a Store row alone.

Cached reads use the existing bounded grant cache. Mutations, organization switching, dispatch and recovery use the applicable live checks. Fail closed for missing scope, withdrawn access or mismatched owner identity. Each domain retains its own visibility and purpose contract.

## Persistence and external effects

- Keep one canonical fact owner and one transaction owner. Use existing shared transactions where facts share a database; do not introduce a generic distributed transaction layer for a local operation.
- Bind immutable organization, actor/member, operation key, payload/version and intent before a mutation. Same key with changed payload conflicts; a retry does not create a new operation identity.
- Across owner databases, use the existing fixed-owner receipt/proof contract. Image execution consumes its closed Resource contract rather than exposing a generic transaction runner, raw SQL or caller-supplied charge outcome.
- Model price, member/month scope, reservation and terminal result remain bound to the original attempt. UNKNOWN stays with the original owner and recovery path; cancellation or restart does not prove that no external effect occurred.
- Money, resource quantities and audit tokens are different facts. Preserve debt-first settlement and original receipt/event identities; do not infer credit or release held funds from a local error.
- Serving verifies schemas and grants. Schema creation/installation is explicit and owned by the existing installer; ordinary startup does not perform DDL.

## Placement and review

Before changing a domain contract, identify its owner, current consumer, transaction/scope boundary and recovery behavior. Follow the Design Basis admission in [AGENTS.md](../../AGENTS.md). A small patch does not create an exception to identity, authorization, idempotency or side-effect rules.

Use current code and contracts to determine behavior. Completed retirement reports, historical counts, old phase ordering and prior implementation notes are not new product requirements.

Current source enforcement is maintained by the existing tests and depguard rules, including route ownership, domain/app separation, Product/Marketplace boundaries, Temporal runtime isolation and retired-package import bans. Run the applicable existing CI; this document does not duplicate its full test inventory.
