> #301 / #1025 USER_CONFIRMED：本批按用户确认的[完整设计](legacy-task-openai-convenience-closure-retirement-boundary.md)整批实施49路径：当前草稿/报表的Actor与纯校验归 `internal/listing/record`，审核单方法合同归 `internal/ai` 的ChatCompletionClient；旧listing/task、两宽接口/便利方法与独占ptr退出，不留alias。当前product.canonical.inspect继续按ProductKey/CatalogVersion读Catalog，不新增record依赖；当前JSON/hash/版本/身份权限/事务/治理/图像采集与Store观察保持。下文旧Task查询端口/宽接口描述仅为历史，不恢复旧owner；workflow/inventory/既有资源不动，#301 OPEN，以下正文完整保留。

> #301 当前设计提案：[旧 listing/task 与 OpenAI 便利调用链完整退休](legacy-task-openai-convenience-closure-retirement-boundary.md)。Independent Architecture / DESIGN_ONLY / SOURCE_NOT_STARTED / NOT_READY；两组同批49路径，不机械拆批。拟将当前草稿/报表的最小身份值与校验抽到 listing/record、当前审核单方法合同留 internal/ai，再整退旧任务包、宽接口、四闲置方法与独占 ptr，不留 alias。§4.2 保留原 JSON/hash/版本/事务/路由与当前能力、Store 观察；§6.1 精确转换原检查/lint。当前只四文档，复核及用户确认后再实施；workflow/完整 inventory/计数/ceilings/依赖/保护与既有实例/数据/凭据/进程不动，不增试用前置，#301 OPEN。以下旧 Task query/宽接口等历史描述仅在本设计获确认的准确边界内被替代，原正文完整保留。

# Project Target Architecture

> Status: approved target architecture for project-wide modularization.  
> Legacy handling: `docs/refactoring/legacy-hard-cut-policy.md` and `docs/refactoring/legacy-register.md`.  
> This document describes the intended end state; current runtime reality is tracked separately in `docs/refactoring/current-refactoring-status.md`.

## 1. Goal

The project converges toward a business-domain-first modular monolith with explicit Product, Listing, Marketplace, Agent/Tool, Organization, commercial/resource, runtime, and external-integration ownership.

The target architecture is designed to eliminate the historical problems where:

- root `internal/listingkit` becomes a complexity sink;
- platform rules, Product facts, Listing orchestration, Task runtime and HTTP assembly mix ownership;
- old Task-first product models leak into the final product experience;
- migration bridges become permanent extension points;
- Agent work creates duplicate facts, state machines, retry owners, RBAC, provider routing or marketplace clients.

**Compatibility is not a target architecture domain.** Existing compatibility/legacy code is drained through `EXTRACT | RETIRE` and disappears from the end state.

## 2. Preferred top-level shape

The exact directory tree may evolve, but the intended ownership shape is:

```text
cmd/

internal/
  app/
    httpapi/
    runtime/
    worker/

  product/
    catalog/
    sourcing/
    enrichment/
    asset/
    image/

  listing/
    task/
    workflow/
    preview/
    revision/
    submission/
    settings/

  marketplace/
    shein/
    amazon/
    temu/
    ...

  commercetool/

  agent/
    runtime/
    prompt/
    ...

  organization/
  commercial/
  ledger/
  knowledge/
  resourcecatalog/

  integration/
    identity/
    authz/
    crawler/
      amazon/
      a1688/
    shein/
    amazon/
    temu/
    ...

  platform/
    config/
    logging/
    metrics/
    database/
    redis/
    queue/
    temporal/
    objectstore/

  shared/

web/
  ... final Product Projection defined by Figma authority ...
```

There is intentionally no final `internal/compatibility/*` owner in this shape.

## 3. Core domain responsibilities

### 3.1 Product

`internal/product/*` owns reusable commerce product facts and capabilities:

- `ProductSnapshot` / canonical product facts;
- source identity, evidence, lineage and normalized sourcing;
- enrichment Proposal behavior;
- approved asset facts;
- provider-neutral image capability.

Product does not depend on Listing, crawler implementations, root ListingKit, compatibility bridges, Agent frameworks, or marketplace SDKs.

### 3.2 Listing

`internal/listing/*` owns marketplace-neutral Listing orchestration and stable listing facts/seams where they are genuinely cross-platform:

- Platform Draft / Listing coordination;
- preview/read models;
- revision/history coordination;
- submission orchestration around the existing single submission-state owner;
- deterministic cross-platform seams.

Listing is a domain capability, not a requirement for a top-level “Listing Center” product navigation.

### 3.3 Marketplace

`internal/marketplace/<platform>/*` owns platform-specific business behavior:

- category/attribute rules;
- platform validation adapters;
- publishing payload rules;
- platform-specific models and capability seams.

SHEIN/TEMU/Amazon capabilities are projected mainly through Store Center / Store Products and AI Workbench according to final Figma IA. They do not each get an independent Workbench architecture.

### 3.4 Product Sourcing and crawlers

Crawler implementations live under Integration and output source-specific/raw extraction results:

```text
integration/crawler/*
        ↓
product/sourcing
        ↓
ProductSnapshot / ApprovedAsset / downstream Listing
```

`product/sourcing` owns `SourceIdentity`, `SourceEnvelope`, normalization, lineage, warnings and handoff. Crawlers do not own canonical Product facts; Listing/Marketplace do not own crawling.

### 3.5 Commerce Tool

`internal/commercetool` owns the framework-neutral Tool contract and governance:

- definition/schema;
- registry;
- risk class;
- allowlist/policy;
- invocation/audit boundary;
- stable error contract.

Business behavior stays in the owning domain and is exposed through narrow adapters. Tools do not become a second implementation of Product/Marketplace/Listing logic.

Agent/Tool code does not directly import GORM repositories, provider SDKs or marketplace clients.

The framework-neutral commerce Tool contract includes Tool Definition, Schema,
Registry, Agent Allowlist, Invocation Policy, and Tool Audit Port. It owns the
policy and invocation boundary, not business-domain implementations or runtime
framework integration.

`kernel/module.Registry` remains limited to startup contribution collection; it
is not the Tool Runtime. A domain Tool adapter receives only narrow
Service/Query ports and implements the Executor contract. The `internal/app`
composition root constructs those adapters, injects the ports, and registers
the resulting Tools with the commerce Registry.

Phase 2A admits only `read`, `compute`, and `propose` risk classes. `write` and
`publish` remain outside this phase until later governance adds their controls.
The import boundary is enforced by `depguard: commercetool_boundaries`, a
strict allowlist that keeps application, domain, framework, transport,
workflow, persistence, and provider implementations outside this owner.

The first bounded vertical slice is `product.canonical.inspect`. It accepts a
task ID, resolves the authorized task-to-product binding through the
provider-neutral `internal/listing/task` query port, and reads the immutable
`internal/product/catalog.ProductSnapshot`. The Tool adapter owns only strict
wire projection and stable error mapping; it owns neither task persistence nor
canonical product facts, and it receives no write or publish port. Composition
injects one configured Casbin authorizer into permission preflight, task owner
scope, and the Executor's defensive scope check. A B0-only bounded read adapter
uses a database-side byte-length conditional projection so oversized payloads
are not scanned into Go, without narrowing the shared Catalog repository's
existing publication/read compatibility.

There is no user-accessible Tool or Agent runtime in Phase 2A. Runtime assembly,
model orchestration, and public entrypoints remain gated on Phase 2B. The B0
slice therefore proves domain and governance contracts without introducing an
HTTP route, worker, queue, or second IAM implementation.

### 3.6 Agent / AI Capability

The current Product Agent contract and concrete consumer gaps are specified in
[Product Agent runtime contract](product-agent-runtime-contract.md) for #131/#132.
That document distinguishes contract preparation from executable/runtime delivery;
its presence does not mean a user-accessible Agent has been implemented.

Agent runtime owns bounded Agent execution semantics such as AgentRun/AgentStep, budgets, tool allowlists, checkpoints/interrupts and stop reasons.

AI Capability owns provider-neutral model capability/routing/policy/ledger/cost/fallback control-plane behavior.

Agent does not own:

- canonical Product/Listing/Asset facts;
- BusinessTask product lifecycle;
- Temporal durable workflow ownership;
- marketplace submission state;
- provider API-key/routing implementation;
- an independent RBAC system.

### 3.7 BusinessTask / product projection

Final product semantics are:

```text
BusinessTask
  -> AgentRun
    -> AgentStep
      -> ToolCall / ModelCall
        -> Temporal / Queue / Internal Task
```

BusinessTask is the user-facing business goal/progress/decision/result object. Internal Task/Workflow remains execution infrastructure.

The final UI/IA and user-facing names are governed by Figma `31:463` and `docs/product/final-ui-ia-authority.md`, not by historical ListingKit navigation.

### 3.8 Organization / identity

ZITADEL remains the trusted identity provider. Organization/membership/authorization semantics stay with current Organization/identity owners and existing RBAC boundaries.

`internal/tenantbridge` is not target architecture. It is active legacy debt tracked for `EXTRACT -> RETIRE` by #301. New schemas/contracts use current Organization identity directly.

### 3.9 App / Platform / Integration

- `internal/app/*`: composition, bootstrap and lifecycle only.
- `internal/platform/*`: runtime infrastructure owned by the application.
- `internal/integration/*`: external systems and concrete adapters.
- `internal/shared`: small stable primitives only.

App must not accumulate business rules; Integration must not become the business owner of Product/Listing/Marketplace semantics.

## 4. Legacy Hard-Cut

The target architecture does not include an internal compatibility layer.

Current legacy handling:

```text
EXTRACT
  reusable behavior
    -> current owner
    -> switch callers
    -> remove old dependency

RETIRE
  obsolete design
    -> no extension
    -> cut over
    -> delete / keep absent
```

Consequences:

- `internal/compatibility/*` is a retirement zone, not a final package layer;
- root `internal/listingkit` is drained by #29, not converted into a permanent facade;
- active 1688 → legacy ListingKit handoff is drained by #30;
- `internal/tenantbridge` is drained by #301;
- already removed `internal/productenrich`, `internal/productimage`, old `internal/catalog`, old `internal/asset`, and `internal/imageasset` remain absent;
- new code does not add legacy fallback, permanent dual-read/write, bidirectional new↔old synchronization, or second fact/state owners.

If concrete, current externally observable contract evidence appears, stop and report it to the user. Only a new explicit user product decision can permit a compatibility exception. It does not create a general Compatibility domain.

## 5. Dependency direction

Preferred dependency direction is owner-oriented, not layer-oriented:

```text
cmd
  -> app

app
  -> product / listing / marketplace / agent / commercetool / organization / commercial / ...
  -> platform / integration

Agent adapters
  -> Commerce Tool contracts
  -> narrow domain ports

listing
  -> product
  -> marketplace seams where required

marketplace
  -> product/domain contracts
  -> integration through narrow adapters

product
  -> shared / local contracts

platform
  -> shared

integration
  -> shared / external SDKs
```

Hard constraints:

- Product does not depend on Listing or legacy compatibility paths.
- Marketplace does not depend on `internal/compatibility/listingkit`.
- New architecture does not depend on `internal/tenantbridge`.
- New Product/Agent/Tool/Console code does not depend on root ListingKit as a convenience service owner.
- Agent/Tool does not bypass domain authorization or deterministic validation.
- There is one owner for submission state, durable retry, IAM, Tool registry and canonical facts.

## 6. Deterministic validation and side effects

Readiness/Validator is deterministic authority shared by UI/fixed pipeline/Tool/Agent. A model can propose a repair; it cannot reinterpret a failed validator as success.

Agent evolution is read/compute/propose first. Write/publish side effects require independent approval/version/authorization/idempotency/audit/readiness gates. Agent Runtime does not create a second publication state machine.

## 7. Migration method

Do not execute another broad directory rewrite. For each legacy seam:

1. name the current owner;
2. identify reusable behavior;
3. classify `EXTRACT | RETIRE`;
4. add/verify tests on the current contract;
5. switch one bounded caller/path;
6. remove the old dependency/path;
7. add a guard so it cannot be reintroduced.

Use `docs/refactoring/module-target-mapping.md` and `docs/refactoring/legacy-register.md` for the active drain inventory.

## 8. Non-goals

This architecture does not require:

- microservice decomposition;
- immediate renaming of every historical package;
- a generic MarketplaceAdapter framework;
- a generic Listing Workspace;
- a permanent compatibility/anti-corruption layer for internal legacy design;
- duplicate Product/Listing/Asset facts for Agent convenience;
- replacing Temporal with Agent Runtime;
- Multi-Agent orchestration before one bounded Agent proves value.

## 9. Authority relationship

Use together with:

- `docs/product/final-ui-ia-authority.md` — final UI/IA/Product Projection;
- Product/architecture specs — business/safety/contracts;
- `docs/refactoring/legacy-hard-cut-policy.md` — legacy treatment;
- `docs/refactoring/legacy-register.md` — current legacy drain inventory;
- `docs/refactoring/current-refactoring-status.md` — current implementation reality;
- GitHub #137 — executable backlog order.
