# Project Target Architecture

The current product uses a business-domain-first modular monolith. Product, Listing, Marketplace, Agent/Tool, Organization, commercial/resource, runtime and integration responsibilities have explicit owners. The [boundary rules](project-boundaries.md) describe dependency direction; the [repository map](../development/repository-structure.md#current-entrypoint-map) identifies real entrypoints.

## Product and fact ownership

- Product sourcing preserves source identity, immutable publication and lineage; Catalog owns canonical product versions and Asset owns approved assets.
- Listing owns current materials, drafts, records and submission intents. A platform adapter owns its wire protocol and pure platform rules, not Product or Asset facts.
- Store owns stored connections and scoped official observations. A stored connection does not itself grant an official application or permission to send.
- Organization identity, membership, scoped permissions and commercial/resource facts retain their current owners.
- BusinessTask is a user-facing business projection. Temporal and other native executions keep their original status, cancellation and recovery facts.

## Agent and Tool boundary

`internal/commercetool` owns Tool Definition, Schema, Registry, Agent Allowlist, Invocation Policy and the Tool Audit Port. `kernel/module.Registry` performs startup contribution collection; it is not the Tool Runtime.

Agents call current Service/Query ports through the composition root. Tool effects remain explicit: read, compute, propose, write and publish have different permissions and confirmation requirements. The existing `depguard: commercetool_boundaries` rule enforces package direction.

`internal/aicapability` and the selected provider adapters own model invocation contracts. Provider outputs are candidates, not authorization, prices, resource grants, deterministic readiness decisions or permission to publish. Model routes, credentials, budgets and immutable execution identities remain server-owned.

## Runtime and consistency

- `internal/app/*` assembles domain modules, native clients, database pools and workers. It does not absorb new domain policy.
- `internal/platform/*` owns runtime mechanisms; `internal/integration/*` implements external protocols and persistence adapters.
- Temporal owns durable orchestration where it is already used. Agent Runtime does not replace it or create another submission/retry state machine.
- Same-database facts use their current transaction owner. Existing cross-owner receipt/proof boundaries retain original operation identity, UNKNOWN and recovery ownership.
- Neutral domain values remain independent of provider DTOs, browser state, HTTP middleware and persistence implementation details.

## Development basis

The [greenfield decision](../product/greenfield-no-legacy-migration.md) governs new work. There is no old-data migration, legacy ID mapping, compatibility facade or second fact source requirement. Only a new explicit user product decision can permit a compatibility exception.

Reuse current behavior at its correct owner and mature libraries. Requirements and priorities come from current Product Decisions and execution Issues, not a static phase roadmap. UI/IA uses the [current Figma Authority](../product/final-ui-ia-authority.md); delivery, installation and real-platform acceptance remain separate states.

## Canonical inspection

`product.canonical.inspect` reads an authorized exact ProductSnapshot and the current `internal/listing/record` contract. It does not own a Task lifecycle, update Catalog facts or grant publication permission. The original permission and canonical-inspection import guards continue to apply.
