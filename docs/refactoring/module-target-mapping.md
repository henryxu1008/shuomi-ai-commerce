> #301 / #923 **SUPERSEDED，仅旧租户 Prompt 闭包**：用户已确认[完整退休设计 §1–§7（含 §4.2/§6.1）](../architecture/legacy-tenant-prompt-closure-retirement-boundary.md)。本文涉及旧 `internal/prompt`、`internal/promptmgmt`、`shared/tenantctx`、五 YAML 及其现役 KEEP/复用建议仅在该准确矩阵内被退休决定替代；如涉及镜像打包，仅退出两条 `prompts` COPY。当前 AI/图像/采集、身份/有效规则、独立 SHEIN 登录与 Store 同步及其它 KEEP 不变。停止全新 schema 创建旧 Prompt 表，不改 Goose 登记或已有表/行，不新增替代 Store/迁移/兼容。以下原正文、历史数字/勾选/证据完整保留，不扩大实施、合并或真实资源权限。

> #301 / #921 当前适用边界：用户已确认并按[旧任务身份与数字映射合同完整闭包 §1–§7/§6.1](../architecture/legacy-task-identity-mapping-closure-retirement-boundary.md)整批实施。仅替代本文件 listingruntime.ProductImportMapping/原 types.go 必须存在、旧 MappingInfo 编码及 listingruntime 仍属当前依赖的准确保留含义；本轮最后值类型完整退休。SourceIdentity 四 JSON 零值槽位、六 Source*、原 trim/规范化/hash/读取拒绝/当前版本及 opaque metadata 继续保留，不删当前序列化字段或改变业务事实 owner。 当前能力/采集/图像、独立 SHEIN 登录与 Store 同步及原 owner/fence/UNKNOWN 保留；原历史正文/计数/批准/勾选完整保留为后缀，workflow/inventory 不改，不操作既有实例/数据/凭据/进程，不增加试用前置。

> 当前说明（#301 / #912）：已确认并整批实施[旧 SHEIN 闲置策略与辅助合同完整退休闭包](../architecture/legacy-shein-policy-contract-closure-retirement-boundary.md) §1–§7（含 §6.1）。旧 SHEIN 价格/活动/话题/标题/属性辅助暂留映射只在附录 A 的22工件及对应旧 KEEP 检查上被替代，不产生新模块或事实 owner；平台 API/transport、当前规则/JSON、采集、独立登录和 Store同步保持；下方原历史段落/计数/链接保留。workflow/inventory 与既有实例/数据/凭据/进程不动，不增加试用前置，#301 OPEN。

# Module Target Mapping

> #301 / #870 [用户确认的完整旧宿主退休边界](../architecture/legacy-host-runtime-closure-retirement-boundary.md)（冻结设计a76db47d7，Reuse Existing Architecture / Ready）已确认并整批实施。仅 supersede 七个旧根 infra/httpx、infra/auth、app/monitoring、app/lifecycle、core/errors、app/ports、app/updater 的旧迁移/保留/存在性/替换目标；38准确Go文件与专属文档/原guard同退，无新EXTRACT或替代框架。当前身份/HTTP/module/Retry/观测/共享基础层、采集、Store及独立SHEIN登录/同步继续原owner。历史正文/计数/失败保留，workflow/inventory/依赖与真实文件/进程/凭据/数据不改，无新试用前置，#301 OPEN；准确证据留PR。

> #301 / #868 implements the [approved complete numeric context/value-contract retirement](../architecture/legacy-numeric-value-contract-closure-retirement-boundary.md), frozen design 01d8b8a50 / Independent Architecture / IMPLEMENTATION_READY. Existing pricing/activity owners directly consume the extracted 5/8 value fields; old numeric logging/sharding APIs, both listingruntime rule types and SourceIdentity alias derivation/Key retire together. ProductImportMapping/MappingInfo, both SHEIN attributes, all four zero-value source JSON slots and original normalization/hash/read rejection contracts remain. This precisely supersedes only #867's retention of those two old input types and alias behavior; current acquisition, Store and standalone SHEIN login/sync keep their owners. Workflows/inventory/dependencies/runtime/data/credentials are untouched; #301 remains OPEN, with no new trial prerequisite.


> #867 当前校准：按[完整数字采集/平台尾链边界](../architecture/legacy-numeric-acquisition-tail-closure-retirement-boundary.md)整批退休无当前装配的 fetch/cache/Task、数字 Cookie/StoreInfo 适配及 Amazon 专属运维源。当前采集 Processor/browser/source port、平台 API/规则、两种 SHEIN 属性、品牌/定价、Store 与独立 SHEIN 登录/同步保留；旧工厂不是后续迁移/重建目标。只删除获准七条 CURRENT inventory 消费者行；下文历史观测/计数保持，workflow/ceilings不变。

> #301/#863 supersedes only the numeric worker/Store retention rows below under user-confirmed frozen design 94fa4cab §1–§7 (Independent Architecture / IMPLEMENTATION_READY). The complete numeric account/Store/Profile/Redis/TEMU StoreRuntime source closure retires together; current opaque Organization/Store, authorized CLI, private fresh bindings and optional Store execution ownership preserve login/Store sync. The unchanged historical race target is test-only. No workflow, real runtime/data/credential action or trial prerequisite; UNKNOWN never auto-takes-over and #301 stays OPEN. Exact implementation evidence stays in #863.


> #29 / #301 [user-confirmed parent-retirement closeout](../architecture/legacy-retirement-parent-closure-boundary.md), frozen design 5fb20f877 / Design Basis N/A, is record-calibrated in #854. Historical target/checkpoint rows now apply through the approved R1-R10/A1-A7/O1-O6 current owner/evidence mapping: old root/bridge/execution closures are retired, while current Product/Listing/Marketplace/Organization owners, native Login V2, shared pure contracts and standalone SHEIN login/Store sync remain. #29 source-level closure evidence is prepared, with merge/closure separately authorized; #301 remains OPEN because its retained numeric persistence owner and unexecuted runtime scope are not cleared by zero bridge imports. Original mappings/counts, production/tests/workflows/inventories/dependencies and real assets remain unchanged; no new trial prerequisite.

> #29 / #301 [user-confirmed complete API token-tool retirement](../architecture/legacy-api-token-tools-retirement-boundary.md), frozen design e6ac780d7 / Reuse Existing Architecture, is source-implemented in #852 after both Issue Ready readbacks. Three obsolete fetch/save/auth-check sources and inline helpers have no replacement token store, issuer or API target; five direct operating documents receive narrow historical headers. Existing Auth.js handoff, native Login V2 probe, acquisition/Store/standalone SHEIN login and sync keep their owners. Original mappings/counts, workflows/inventories/dependencies, credentials/runtime/data and trial prerequisites remain unchanged; exact HEAD/CI/final review stay in the PR.

> #29 / #301 [user-confirmed complete Kubernetes release-principal closure](../architecture/legacy-kubernetes-release-principal-retirement-boundary.md), frozen design da4ac7d16 / Independent Architecture / IMPLEMENTATION_READY, is source-implemented in #850. Twenty-three exclusive principal/authentication/helper/fixture sources have no replacement publisher or token target. Seven mixed sources keep current safety at the existing Rego/Conftest/Go owner with a canary-only assembly, zero authority objects, no old machine fields, and missing-input rejection. Shared test utility bodies remain in place. Current UI/native identity/schema/ImageAgent, generic tools, Store and standalone SHEIN login/sync retain their owners; original mappings/counts, workflows/inventories/dependencies and runtime/data/IAM remain unchanged, with no new trial prerequisite.

> #29 / #301 [user-confirmed Kubernetes release-execution closure](../architecture/legacy-kubernetes-release-execution-closure-retirement-boundary.md), frozen design a73f44144 / Independent Architecture / IMPLEMENTATION_READY, is source-implemented in #847. Fifteen exclusive sources have no relocation or replacement publisher/drain target; twelve mixed sources remove exact wiring/defaults/obsolete metadata/permissions and correct retired credential guidance. Only three existing test utilities relocate unchanged to the existing security-test owner. Current UI/schema/ImageAgent recovery/workers/canary, shared apply/security and SHEIN/Store retain their owners. Historical mappings/counts, workflows/inventories/dependencies and runtime/data remain; no new trial prerequisite.

> #29 / #301 [user-confirmed complete local acceptance closure](../architecture/legacy-local-acceptance-closure-retirement-boundary.md), frozen design 6a654c04f / Reuse Existing Architecture, is source-implemented in #846. The eight exclusive old Task API/dev launch and ImageAgent acceptance sources, inline helpers and direct historical tails retire together with no relocation target or replacement acceptance runtime. Current UI/tests, native ZITADEL CLI/library, token handoff, issue357, ImageAgent/Temporal, generic tools, Store and standalone SHEIN login/sync keep their owners. Historical mappings/counts, workflows/inventories/dependencies, runtime/data and trial prerequisites remain unchanged; exact HEAD/CI/final review stay in the PR.

> #29 / #301 [user-confirmed complete Task diagnostics closure](../architecture/legacy-task-diagnostics-retirement-boundary.md), frozen design ddc6414b0 / Reuse Existing Architecture, is source-implemented in #844. The old pricing-cache inspector and preview dataset checker have no relocation target or successor Task/cache owner; exclusive test/example/helpers and exact help/validation/Temporal PoC documentation tails retire together. Current pricing/API/rules, Temporal infrastructure/workers, generic DB tools, prompt replay, Store and standalone SHEIN login/sync keep their owners. Historical mappings/counts, commercial gates, workflows/inventories/runtime/data and trial prerequisites are unchanged.

> #529 / #301 [confirmed Yudao identity operations closure](../architecture/legacy-yudao-identity-operations-retirement-boundary.md): three historical user importers and the owner-recovery dry-run are source-retired together in #842 under frozen design 370695e82, including inline helpers/report generators and direct documentation tails. There is no replacement or relocation target for old-ID mapping/import/recovery. Current authruntime/Organization/phone onboarding, shared ZITADEL infrastructure and standalone SHEIN login/Store sync keep their owners. Historical mappings/counts, all workflows/inventories, runtime/data and trial prerequisites are unchanged.

> #839 用户确认冻结设计 `0871cb591` 后按[完整运维源码闭包](../architecture/legacy-listing-operations-closure-retirement-boundary.md)整批实施，Independent Architecture / IMPLEMENTATION_READY。旧部署/监控/rollout/恢复SQL没有替代业务owner；仅原Prometheus PVC低空间规则及同namespace窄路由由现有monitoring源owner承接。SHEIN共享凭据装配及当前Store/登录/采集/监控/通用工具保持，准确矩阵见§5/§6.1。历史映射、计数、全部workflow/inventory不改，不授权运行清理。

> #836 [旧数字任务与刊登专属配置完整闭包](../architecture/legacy-runtime-config-closure-retirement-boundary.md)：冻结设计 `209b40f54` 已确认并整批实施，旧 Processor/Worker/control-plane、全局 RabbitMQ 任务 schema、平台旧调度/核价/同步/监控、debug/usage 及其默认/加载/env/校验/示例/专属测试尾部一起退出；仅替代§5/§6.1准确对象的旧保留表述。真实采集 workerpool、独立通用 queue/Retry、当前身份/计费/图像/平台/Store 与独立 SHEIN 登录/同步 owner 保持；原 Viper/Manager/合并算法不变。历史矩阵/计数保留，全部 workflow/inventory 不改，无真实配置/实例/数据操作或新试用前置。

> #833 [旧数字内容配置完整闭包](../architecture/legacy-numeric-content-management-closure-retirement-boundary.md)：冻结设计 d139bcccc 已确认并整批实施，三族无生产注入的词库/话题配置仓库读写、DTO、全局 setter、专属 schema/迁移和调用尾部源码退出；此前整体保留表述仅在该设计 §5/§6.1 的准确对象上被替代。当前默认清理直接复用原 content 内存工厂，品牌/字段/runtimeCtx、generationtopics 纯规则、Store/API/schema/权限与独立登录/Store 同步保持。物理对象 UNKNOWN、不操作；原历史矩阵/计数保留，全部 workflow/inventory 不改，无新试用前置。

> #831 [confirmed numeric Store closure](../architecture/legacy-numeric-store-management-closure-retirement-boundary.md) retires only the unused Store management/configuration declarations listed in §5, without relocating them or adding an owner/adapter. The original account repository's current reads/sync writes, nine-method StoreAPI, worker/TEMU consumers, shared values and current Store/Organization boundary remain. Earlier broad Store retention is narrowed only for this exact closure; no persistence/runtime operation, workflow or inventory-row change.

> #827 [confirmed Publishing runtime closure](../architecture/legacy-shein-publishing-runtime-closure-retirement-boundary.md): The old Publishing runtime/factories/resolvers, persistence cache and remote submission/recovery helpers have no replacement owner or relocation target. Current Publishing values/assembly/strict validation and the 13 official ExecutionKernel files keep their current owners. Old mappings for only the exact closure are historical; no new facade or runtime/data migration. Workflows, dependencies, ceilings, runtime/data and trial prerequisites are unchanged.

> #824 supersedes only the old numeric listing support mappings: taskrpcapi, workspace/shein, core/metrics and the eight unused listingadmin management families are source-retired under the [confirmed boundary](../architecture/legacy-numeric-listing-support-closure-retirement-boundary.md). No replacement transport, compatibility shell or persistence owner is introduced. Retained Store/content/topic contracts and the offline validator's exact readiness rules keep their existing owners; historical rows below are not new relocation work.

> #822 已完整退休旧 internal/amazon 非API刊登子包、amazonlisting、marketplace/amazon、state、httpbootstrap 与 shared/timex；下文涉及这些对象的旧迁移目标仅保留历史映射，不是当前抽取/重建任务。当前 Amazon API/来源采集、httproute、其他共享能力与独立 SHEIN worker 保持，详见[本批边界](../architecture/legacy-amazon-listing-closure-retirement-boundary.md)。

> Status: active current-owner and retirement mapping.<br>
> Legacy policy: `docs/refactoring/legacy-hard-cut-policy.md`  
> Product baseline: `PD-GREENFIELD-NO-LEGACY-MIGRATION-2026-09-08`
> Scan baseline: `main @ 6e3b87208e2c4e51039b95145a2377edfbb9b1cb`
> Bounded delivery calibration: Issue #386 against `main @ 08a6195c7aa8528acdec0bf0f4e742a8c4998e6d`

### #29 backend retirement calibration

The root ListingKit Task/API/default HTTP assembly is physically retired.
Current commands no longer compile any ListingKit package. Role middleware is
owned by `internal/authz/httpapi`, request-role context by `shared/aiidentity`,
current image transport by `app/httpapi/image_asset_transport.go`, and the original
submit-in-progress error by Listing/Submission. Existing identity/history and
numeric sync leaves remain explicitly registered debt; they are not new owners
or permission to add current consumers. Current schema evolution and worker
contracts remain intact; source/build/review/CI and actual deployment are separate.

### #529 / #301 release identity-preflight retirement

The approved internal-trial boundary retires identity-preflight and its
exclusive runtime/readers/tenantbridge packages. Current source API/UI
release/rollback and PS entrypoints always fail before operations; production
remains NOT_READY. The current image and source runner/RBAC inventory remove
only the identity-preflight artifacts. Existing OIDC/RBAC/attestation security
helpers, worker/schema contracts and required CI checks remain.

Workbench current Organization/scoped roles are preserved without a legacy
resolver hook, including valid numeric string Organization IDs. All bridge
imports are absent. Earlier calibrations below are historical then-state;
standalone SHEIN numeric account persistence remains separate debt. No live
IAM, deployment, data, profile or historical executor is changed.

## 1. Purpose

### #301 standalone SHEIN service assembly calibration

The dedicated worker consumes only Service/Close from its original bootstrap.
Unused SHEIN login HTTP handlers/routes/numeric header resolution are retired;
bootstrap builds the unchanged service without a Handler/Module. Its actual
command dependency graph no longer contains tenantbridge. The original account
repository, worker, Redis, browser/profile, Store sync, health and cleanup remain.
Worker numeric account persistence and release identity-preflight remain debt;
this does not create current Organization ownership, deploy or touch stored data.

### #678 current command assembly calibration

The current application and ImageAgent worker no longer compile tenantbridge.
The existing standalone SHEIN worker's process assembly belongs to
`internal/app/runtime/sheinloginworker`, directly consumed by its cmd; its
shared HTTP entrypoint is absent. Unused ListingAdmin HTTP projection/helpers
are retired, while original repository/API behavior and current SHEIN draft
semantics are unchanged. The later #301 calibration above removes idle HTTP
transport; remaining worker numeric account persistence and historical tools
still require their own #301/#529 retirement; this does not create a new owner
for their old facts or claim deployment/user acceptance.

### #529 owner-scope tooling calibration

The later identity-preparation batch retires the Yudao-to-ZITADEL migration
dry-run script and unconsumed `tenantbridge/bootstrap`, without moving or
wrapping them. Current Organization identity remains with its existing owner;
no historical account/tenant mapping preparation is admitted. Only bootstrap's
two real import edges and depguard exclusions disappear (then seven to five).
The release identity-preflight, remaining resolver readers and standalone SHEIN
worker keep their existing contracts and remain explicit retirement debt.

The old owner-scope dry-run/exception cmd and their exclusive process/write
paths are also RETIRE under #529. The shared ownerreconcile package remains
consumed by the unchanged identity-preflight read path; only cancelled backfill
and fixed-snapshot seed methods and write-only tests are removed. The old
deployment procedure is SUPERSEDED evidence. This does not replace or remove
the current release gate, shared read policy, schema or real historical data.

### #529 historical Source Account tooling calibration

The old ownership-preflight cmd, PowerShell wrapper and dedicated
`integration/persistence/sourceaccount/ownershipmigration` package are RETIRE.
The cancelled A/B1 profile/mapping/receipt/prepare/schema path has no current
consumer or replacement owner. Its historical operational report is retained
as SUPERSEDED evidence; it is not an executable operation in the current tree.
Current `source-account-registry-schema-init` and SourceAccountRegistry owners
remain unchanged and do not consume the retired package. Other Store-history
tools, release identity-preflight and standalone SHEIN numeric consumers remain
explicit debt; this is not whole #529/#301 closure or runtime acceptance.

### ACC-1 bounded current-owner calibration (#409)

On main `aff6b8def46cf7c2d989408d115fa3c9628e585e`, personal account reads use
CurrentIdentity -> `workbenchcontext/httpapi.GetAccountProfile` ->
`authidentity.SelfProfileReader` -> `authruntime/zitadel.UserInfoClient.ReadSelf`,
through the dedicated Next account BFF and `src/lib/api/account.ts`. This is a
current, read-only identity projection independent of Effective Organization;
there is no admitted profile write owner. The current Console Account surface
is reused under #409, not replaced because its web directory has a legacy name.
See `docs/engineering/issue409-account-profile.md` for its presentation boundary.
This row does not claim #410/#411/#412 implementation, final browser acceptance,
deployment or real IAM verification.

### ACC-4 bounded current-owner calibration (#412)

The Account audit slice exposes `/workbench/account/organization/audit` through
the dedicated `/api/account/audit` BFF and `GET /api/v1/account/audit`.
`internal/app/accountaudit` is a read-only allowlist projection, not an audit
fact owner: it calls `sourceaccountregistry.HistoryService`, which reuses the
current Service authorization, then the existing persistence repository's
`ListCommittedOperations` over `source_account_operations`.
The permission remains `workbench.source_account.read`; `LiveWrite` selects
fresh grant resolution and does not grant write permission. The default current
application has this GET in its exact 11-route code contract. This is code
mounting, not deployment or production acceptance.

Only committed successful source-account `register / enable / disable`
receipts are projected. Failed or unknown outcomes, members, renewals, Store
and general security audit are outside this slice. An empty page describes only
this source and cannot establish that the enterprise has no other activity.
The cursor is scoped to the enterprise and ordered by owner operation time,
account ID and version; it is not global commit order. No new table or second
fact source is introduced. See `docs/engineering/account-audit-v1.md`.

This document maps current package areas to current target owners. It is an ownership/retirement aid, not a requirement to rename everything at once or migrate old business data, IDs, profiles or runtime state.

Current target ownership domains include:

- `listing`
- `marketplace`
- `product`
- `agent`
- `commercetool`
- `knowledge`
- `resourcecatalog`
- `commercial`
- `ledger`
- `organization`
- `integration`
- `platform`
- `app`
- `shared`

**`compatibility` is not a target domain.** Existing `internal/compatibility/*` code is a drain/retirement area governed by `EXTRACT | RETIRE`.

## 2. Mapping table

| Current area | Current role | Target owner / destination | Current owner / retirement rule |
| --- | --- | --- | --- |
| `internal/listingkit` | retired mixed Task/API/default runtime | current Product/Listing/Marketplace/Integration/App owners | #29 physically removes the root Go package; valid role/context/image/Temporal transport and Submission error use current owners directly. Remaining numeric sync/identity/history leaves are registered retirement debt, not current landing zones. |
| `internal/compatibility/listingkit` | retired drain shell/handoff | existing current Product/Listing/Marketplace owners | Physical shells/handoff absent under #663/#29; keep absent, without new facade, fallback or data mapping. |
| `internal/listingadmin` | listing-admin behavior | `internal/listing/settings` or `internal/app/httpapi` | Split business policy from transport/assembly. |
| `internal/listingsubscription` | listing-era plans/entitlements/usage behavior | `internal/commercial/*`, current resource/entitlement owners, narrow listing policy where truly listing-specific | Do not restore old task-quota authority; do not add wallet/payment/order ownership here. |
| `internal/ledger/money` | canonical payment/refund/chargeback and payout-method facts; Organization wallet target owner | `internal/ledger/money` + `internal/integration/persistence/money` | CURRENT money owner. Extend only with money semantics; pricing, commercial orders and resource balances stay outside. See #457 contract. |
| `internal/ledger/orgresource` | Organization platform-resource balance/events/reservations | `internal/ledger/orgresource` + `internal/integration/orgresource` | CURRENT resource owner. Commercial acquisition may enter only through the narrow source-bound purchased-resource grant; no generic browser mint. |
| `internal/commercial/billing` | sellable offer / authoritative quote / commercial order and recovery coordination | `internal/commercial/billing` | TARGET/CURRENT landing zone introduced by #457. It composes money and resource owners and must not duplicate their balances or settlement facts. |
| `internal/platformtask` | task execution helpers | `internal/listing/task` or runtime owner such as Temporal/queue | Business task semantics and runtime execution stay separate. |
| retired `internal/taskstatus` / `internal/app/taskstatus` | old numeric task status helpers | current owning domains; no replacement numeric executor | **RETIRE: absent after #729; keep absent.** |
| old `internal/catalog` | old canonical product package | `internal/product/catalog` | **RETIRE: package is already absent; keep it absent.** |
| old `internal/asset` | old asset package | `internal/product/asset` | **RETIRE: package is already absent; keep it absent.** |
| old `internal/imageasset` | old image/asset helpers | `internal/product/image` or `internal/product/asset` | **RETIRE: package is already absent; extract only if a still-valid behavior is rediscovered.** |
| old `internal/productimage` | old ProductImage runtime | `internal/product/image` + ImageAgent | **RETIRE: package is already absent; do not recreate task/queue/worker/API ownership.** |
| old `internal/productenrich` | old ProductEnrich runtime | `internal/product/enrichment` + AI Capability where needed | **RETIRE: package is already absent; do not recreate workflow/task ownership.** |
| `internal/product` | current Product Domain | `internal/product/*` | Current owner; extend by Product subdomain rather than creating another product fact source. |
| `internal/pricing` | mixed pricing helpers | Product, marketplace-specific policy, or listing submission according to semantics | Classify by business owner before moving. |
| `internal/sds` | specialized POD/design capability | Product asset/image or listing studio according to behavior | Preserve POD specialization; do not make it a generic source owner. |
| `internal/ai` | mixed provider-neutral/provider runtime AI concerns | `internal/agent/*`, current AI Capability owner, or `internal/integration/<provider>` | Provider SDK types do not leak into domain contracts. |
| `internal/aicapability` | current model capability/routing/policy/cost foundation | current AI Capability/Agent control-plane owner + persistence adapters | #126/#130 stabilize before Product Agent main implementation. |
| `internal/prompt` / `internal/promptmgmt` | prompt contracts, persistence, management transport | Agent/prompt owner + integration persistence + app transport | Split domain contract, persistence, and HTTP assembly. |
| `internal/crawler` | historical crawler implementations | `internal/integration/crawler/*` | Crawler owns extraction/runtime, not canonical Product facts. |
| `internal/amazon/*` retained subpackages | Amazon API/rules/models/pipeline; root task executor retired by #729 | `internal/marketplace/amazon/*` + `internal/integration/crawler/amazon` | Keep source and target listing concepts separate. |
| `internal/amazonlisting` | Amazon listing behavior | `internal/marketplace/amazon/*` / listing seams | Extract platform rules; do not copy a platform Workbench. |
| `internal/shein` | mixed SHEIN logic | `internal/marketplace/shein/*` + `internal/integration/shein` | Separate API/client integration from platform business rules. |
| `internal/publishing/shein` | historical SHEIN publishing shell | `internal/marketplace/shein/*` / existing submission owner | Extract valid rule/transport behavior; retire shell after callers switch. |
| `internal/workspace/shein` | historical SHEIN workspace shell | current SHEIN marketplace/listing projection where still needed | Final UI does not require a platform Workbench. Extract useful behavior, then retire shell. |
| `internal/sheinlogin` / `internal/sheinloginmanaged` | SHEIN authentication integration | `internal/integration/shein` + app lifecycle wiring | External auth adapter, not listing/product owner. |
| `internal/temu` | mixed TEMU capability | `internal/marketplace/temu/*` + `internal/integration/temu` | #31 expands shared capability without independent Workbench. |
| `internal/platforms` | mixed platform abstractions | marketplace or listing owner | Keep only genuinely cross-platform contracts outside platform-specific owners. |
| `internal/workspace` | historical generic workspace behavior | listing/marketplace owner only if behavior remains valid | Generic Listing Workspace is retired; do not recreate it as target architecture. |
| `internal/publishing` | generic/platform publishing support | marketplace-specific rules + single listing/submission owner | No second submission state machine. |
| `internal/app` | runtime composition/lifecycle | `internal/app/*` | Wiring only; no new business-rule ownership. |
| `internal/httpbootstrap` / `internal/httproute` | HTTP bootstrap/routes | `internal/app/httpapi` | Transport/assembly ownership. |
| `internal/taskrpcapi` | internal task transport | app transport/runtime owner | Do not expose as BusinessTask product semantics. |
| `internal/infra` | mixed runtime infra/external clients | `internal/platform/*` or `internal/integration/*` | Split runtime infrastructure from external adapters. |
| `internal/platformbase` | shared runtime helpers | platform/shared only if truly generic | Drain catch-all behavior toward named owners. |
| `internal/authidentity` | verified tenant/user/organization identity envelope | current Organization/identity authority or minimal shared identity envelope | Organization/membership policy remains with Organization owner. |
| `internal/authruntime/zitadel` | ZITADEL verification/middleware | integration identity + app HTTP wiring | ZITADEL remains identity provider; no parallel IAM. |
| `internal/authz` | permissions + policy-engine concerns | Organization/business permission owner + Casbin integration | Permission semantics are business-owned; engine is adapter. |
| `internal/kernel` / `internal/core` / `internal/pkg` | generic/cross-cutting helpers | `internal/platform/*`, `internal/shared`, or a named domain | Avoid new dumping grounds. |
| `internal/validation` | generic validation primitives | `internal/shared/validation` or deterministic validator owner when business-specific | #34 owns readiness/validator business contract. |
| `internal/state` | generic state handling | runtime or business state owner | State machines must have one named owner. |
| `internal/pipeline` (numeric `internal/processor` retired by #729) | remaining orchestration | app worker, Product sourcing, Listing workflow, or runtime owner | Split by business/side-effect ownership. |
| `internal/ports` / `internal/domain` / `internal/model` | broad aggregation packages | local owning domains | Prefer local contracts/models over global catch-alls. |
| `internal/scheduler` | scheduling runtime | app worker / platform queue/Temporal | Runtime only. |
| `internal/sourceaccount` + unconstructed 1688 numeric queue/Profile chain + unregistered `app/productsourcing/httpapi` | old package, numeric import HTTP/DTO, queue Service/worker, numeric Profile/account-assisted methods and unused shared source-account metadata retired under #301 | `internal/sourceaccountregistry` + its existing persistence for current registration; current sourcing/acquisition/Browser Capture owners, original public browser/processor, local-agent and shared HTTP/result owners retain valid behavior | **RETIRE:** keep absent without wrapper, old-table/profile reuse or ID mapping. Registration is not a connected-account guarantee. Shared trusted TenantID, task/result state, scoped protections, infrastructure and the independent SHEIN worker remain unchanged. |
| Independent `sheinlogin` worker numeric account/runtime boundary | Retained repository provider, DB/Redis/Profile/session/state and Store sync; unused alternate managed provider/helpers/verification DTO retired, duplicate lookup tenant-qualified and Store sync fails closed on incomplete/wrong-scope results (§5.1/§7) | Current Organization/Store/official business owners remain separate; valid shared contracts and independent worker capabilities stay with existing owners, without accepting cross-tenant duplicate handling | See [#301 confirmed boundary](../architecture/shein-worker-identity-persistence-boundary.md). No new current consumer, ID mapping, migration, runtime/data action or internal-trial prerequisite; code/CI does not imply real-environment acceptance. |
| Standalone `listing-scheduler` / `shein-listing` / `temu-listing` and exclusive local/scheduler/sync graph | RETIRE: source and exclusive operation entrypoints absent under confirmed #29/#301 scope | Current acquisition, Store observations, valid shared contracts and independent login worker keep their owners; task/status/dispatch repository debt remains, control-plane subsequently retires in #726 | [Confirmed boundary](../architecture/legacy-listing-cli-retirement-boundary.md) §5/§6/§6.1 and the follow-up row below; original security/required CI retained; no runtime/data operation or trial prerequisite. |
| `listing-control-plane` / `shein-import-platform-recovery` and exclusive orchestration | RETIRE: commands, runtime/control library, cohort helper/tests and exclusive operation packaging absent under confirmed scope | Current acquisition/observations/shared contracts and independent login worker keep original owners; task repository/races and package-external task/status consumers remain debt | [Confirmed boundary](../architecture/legacy-listing-control-plane-retirement-boundary.md) §5/§6/§6.1/§7; both workflows remove only obsolete targets, preserving required/security checks. No runtime/data actions or new trial prerequisite. |
| `internal/tenantbridge` | package and all frozen imports absent after admitted preflight retirement | current Organization identity + each consuming domain's current persistence ownership | #301: freeze new consumers; new domains use native Organization identity; RETIRE legacy callers and package by owner; it allows no legacy numeric mapping, old-data migration/backfill or cutover. Never move it to another compatibility facade. |
| `internal/zitadelprovision` | ZITADEL management/provisioning | integration identity provisioning + app operational entrypoint | External client stays in integration; app owns lifecycle. |
| `web/listingkit-ui` | current Console application; #664 removes the old ListingKit/Task-first pages, default shell and dedicated request chains | final Figma Product Projection, especially #298 and Store Center surfaces; shared readonly identity transport lives in `src/lib/server/upstream-identity.ts` | Keep current Console, account, plans and review surfaces; current source/image/store surfaces and neutral primitives also remain. Retired paths stay absent without redirect, Task mapping or fallback. The directory name is not a legacy owner. Deployment and user acceptance are separate. |

### SRC-2B1 current candidate seam (#398)

Frozen `src2b-acquisition-v1` and bounded guard admission
[5643032970](https://github.com/qq550723504/task-processor/issues/398#issuecomment-5643032970)
admit the current sourcing validator, anonymous `integration/acquisition/a1688`
adapter, `integration/persistence/product/acquisition` staging, and
`app/productsourcing` coordinator into the optional current application module.
SRC-1 and Catalog retain evidence/publication and ProductSnapshot authority.
There is no SourceAccount/Connection/login prerequisite or legacy fallback.

The explicit initializer reuses `platform/database.OpenExistingWritableContext`
and `Close`; the public adapter reuses `httpimage.NewPublicImageHTTPClient`.
These are qualified CURRENT leaf capabilities, not RETIRE owners. Precise
file/API guards do not admit siblings or subpackages. The thin maintained
`scripts/product-acquisition-init.ps1` owner only invokes the explicitly
confirmed empty-database initializer; ordinary runtime performs no DDL.

Status: candidate implementation, not a main/deployment receipt. Task-owned
HTTP/PostgreSQL and mounted authorization/ambiguity tests have local evidence;
final HEAD, CI and independent complete-call-chain review belong in the PR.
Real 1688 acceptance remains NOT_RUN; Browser backend/UI is a later slice.

### 2.1 Bounded current-delivery calibration (#386)

This is a **bounded symbol/caller calibration**, not a replacement full-repository scan. The historical scan baseline above remains `6e3b87208e2c4e51039b95145a2377edfbb9b1cb`; the rows below were checked against `main @ 08a6195c7aa8528acdec0bf0f4e742a8c4998e6d`. `CURRENT` identifies the allowed owner, not deployment or default runtime presence. `MERGED`, isolated acceptance, default mounting, deployment health, and business acceptance are deliberately separate states.

| Object / exact entry | Owner and decision | Code / assembly state | Runtime evidence | Allowed new calls | Forbidden edge / exit |
| --- | --- | --- | --- | --- | --- |
| SA1 registry + SA2 Console/BFF: `internal/sourceaccountregistry`; `internal/integration/persistence/sourceaccountregistry`; `internal/app/schema/sourceaccountregistry.Migrate`; `internal/app/httpapi.NewSourceAccountApplication`; `web/listingkit-ui/src/lib/api/source-accounts.ts`; `web/listingkit-ui/src/app/api/workbench/[...path]/route.ts`; `web/listingkit-ui/src/lib/server/workbench-proxy.ts` | `sourceaccountregistry` owns Organization-scoped registration metadata and lifecycle; persistence is an adapter; app owns schema/HTTP lifecycle; Console/BFF owns same-origin transport. **CURRENT.** Provider authentication/connection is a separate capability and registration creates `pending_connection`, not a connected account. | PR #369 merged as `ef6c89ec9388f16d35a9ff964511ab7b2d3d3f0f`; PR #373 merged as `63c1806606a2a98faf510116d0119506ba1f454c`. `NewSourceAccountApplication` has no non-test default-composition caller, so its Go server and explicit schema remain opt-in even though the BFF/client code exists. | #386 controlled chain PASS: browser client -> Next/Auth.js encrypted session -> same-origin BFF -> Go verified identity/effective Organization/permission/UoW -> task-owned empty PostgreSQL. Real ZITADEL issuance, real 1688, shared/default service mount, deployment, runtime health, and business acceptance: **NOT_RUN**. | Call the current `/api/v1/workbench/source-accounts` contract through verified identity/effective Organization; initialize its explicit schema and own DB/listener/server lifecycle at the application boundary. | No call through old `internal/sourceaccount`, `internal/tenantbridge`, or compatibility handoff; no connection-state fiction, fallback, migration, dual read/write, or second registry. Exit is a separately admitted default mount/provider connection, not reuse of the old repository. |
| SRC-1 source evidence -> Catalog: `internal/app/productsourcing.NewInternalProducer`; `internal/product/sourcing.InternalProducer.{Publish,Verify,Read}`; `internal/integration/persistence/product/sourcing.NewRepository`; `internal/product/catalog.ProductSnapshot` | Product Sourcing owns immutable source evidence, normalization and publication identity; Catalog owns canonical `ProductSnapshot`, version/head and publication facts. **CURRENT.** | PR #381 merged as `675302829a4ad40ffedab04c7ba93f19ab7141e0`. The persistence adapter publishes sourcing evidence and Catalog facts under its admitted transaction boundary. The only production construction found is inside the isolated REV-1 application; there is no default producer mount. | Controlled PostgreSQL publication/read/replay paths PASS in #386 REV-1 verification. Real source provider acquisition and default/deployed runtime: **NOT_RUN**. | Use the narrow admitted producer/reader contracts with verified authorization and immutable publication identity. | No direct Product import of crawler/provider DTOs; no old A1688 handoff, root ListingKit Task, tenantbridge, caller-supplied Organization, fallback or second Catalog fact source. Default provider ingress requires separate admission. |
| PA-1 approved inventory: `internal/product/asset.ApprovedAsset`; `internal/product/asset.ApprovedAssetInventory`; `internal/product/asset.Repository.GetApprovedInventory`; `internal/integration/persistence/product/asset.NewRepository` | Product Asset owns explicit human-approved asset facts; persistence reconstructs that inventory. **CURRENT.** | The current contract and PostgreSQL adapter are present. PR #375's exact-version repair merged as `dc2fd156f6872e075963cd6b939b89b5dc8713f6`: a requested source snapshot version does not fall back to another version. This does not classify every still-present unversioned call. | #386's full Go package command PASS, but no dedicated PA-1 execution receipt was established; dedicated PA-1 isolated/provider/approval/deployed business acceptance is **NOT_RUN** here. | Read the exact approved inventory required by the consumer's Product/version/platform scope; keep approval authority in Product Asset. | No “latest” fallback for an exact-version request, no generated/selected asset promoted as approved without the approval owner, and no second asset store. Remaining unversioned callers need their own evidence and admission. |
| DRAFT-S1: `internal/app/httpapi.NewSheinRecordApplication`; `internal/listing/record`; `internal/marketplace/shein/draft.Builder` | Listing Record owns the immutable local record/UoW; SHEIN draft owns the deterministic marketplace projection/validation; app owns isolated composition. **CURRENT.** A local record is not a remote SHEIN draft or submit authority. | PR #377 merged as `06beadc4c4a818fe4b01220a9f5c9b625ce6cb71`. The constructor has no non-test default-composition caller, so schema/server construction remains opt-in. | #386's full Go package command PASS, but no dedicated DRAFT-S1/browser execution receipt was established; dedicated isolated acceptance, real SHEIN API, default mount, deployment, runtime health and business acceptance are **NOT_RUN** here. | Compose exact Catalog snapshot + PA-1 approved inventory + current Store context through the DRAFT-S1 application boundary. | No provider write, AI proposal, root ListingKit Task/Workspace, compatibility bridge, fallback or claim that local persistence equals remote publication. Exit requires separately admitted provider/submission wiring. |
| REV-1: `internal/app/httpapi.InstallProductReviewSchema`; `internal/app/httpapi.NewProductReviewApplication`; current Catalog/Sourcing readers; Product Review repository/UoW | Catalog remains product-fact owner, Sourcing remains evidence owner, and Product Review owns proposal/decision/apply state and UoW. **CURRENT.** | PR #383 merged as `c3c098da00ef4f27e7c291434924e314eb855f29`. Explicit schema and constructor exist, but `NewProductReviewApplication` has no non-test default-composition caller. | #386 task-owned PostgreSQL lifecycle, concurrency, replay, response-loss, authorization-revocation, exact-schema and race checks PASS. Default mount, deployment, runtime health, real model/provider and business acceptance: **NOT_RUN**. | Use the current Catalog/Sourcing readers and Review repository behind verified identity, effective Organization and existing authorization; caller owns schema/DB/listener/server lifecycle. | No in-memory binding fallback, caller-supplied Organization/publication authority, provider publication from Review, legacy Task coupling, fallback or second product fact source. Default mounting needs separate admission. |
| SUB-K1: `internal/listing/submission.ExecutionKernel`; `internal/listing/submission.NewExecutionKernel`; `internal/integration/persistence/listing/submission.NewRepository` | Listing Submission owns the durable provider-neutral execution kernel and its idempotent state transitions; persistence is an adapter. **CURRENT kernel, not whole submission stack.** | PR #385 merged as `08a6195c7aa8528acdec0bf0f4e742a8c4998e6d`. `NewExecutionKernel` has no non-test application/default-composition caller. | Kernel PostgreSQL/race evidence belongs to the merged PR #385 slice. #386's full Go package command PASS, but no dedicated SUB-K1 execution receipt was established; live app authorization, provider call/readback, default mount, deployment and business acceptance are **NOT_RUN** here. | Add only provider-neutral ports/adapters after a separate application/provider admission preserves kernel idempotency and recovery invariants. | Do not infer that old root ListingKit submission, `submit_lock`, platform callers or provider side effects are cut over. No wrapper, duplicate state machine or compatibility path; exit is caller/provider cutover under their own owner. |
| ImageAgent split: `internal/imageagent`; `internal/app/worker/imageagent`; `cmd/image-agent-temporal-worker`; `internal/app/httpapi.buildImageAgentModuleResult`; `internal/app/httpapi.newImageAgentAuthorizedAssetCatalog`; `internal/app/worker/imageagent.BuildOrganizationImageCapabilities`; `internal/app/worker/imageagent.OrganizationExecutionAuthorizer`; `internal/app/httpapi.NewImageAgentOrganizationApplication` | ImageAgent domain/workflow/store, Temporal worker and deployment path are **CURRENT**. The root ListingKit Task input catalog bridge is **LEGACY-RETIRE**. Organization-scoped worker review governance/execution authorization are **CURRENT but isolated**. The opt-in Organization HTTP application is a separate current control-plane assembly, not the worker governance assembler. | Default HTTP composition builds the ImageAgent module, but `newImageAgentAuthorizedAssetCatalog(listingkitstore.NewTaskRepository(...))` still adapts `listingkit.Task`. Default worker composition calls `buildProductionImageCapabilities`; it does not call `BuildOrganizationImageCapabilities` or install `reviewGovernance`, and the default Temporal activity assembly does not populate `imageagenttemporal.ActivityDependencies.ExecutionAuthorizer`. `NewImageAgentOrganizationApplication` separately assembles Organization-scoped HTTP/catalog/service controls and has no default registration. | Worker manifests/deploy workflow are code/config evidence only; #386 did not inspect a live cluster or business run: deployment/runtime/business **NOT_RUN**. Organization worker governance/authorizer and Organization HTTP application are **NOT_PRESENT** in their respective default compositions. | Extend ImageAgent's current domain/workflow/worker contracts; an admitted current Product/Asset input may replace the legacy task catalog. Keep `BuildOrganizationImageCapabilities` + `OrganizationExecutionAuthorizer` and the separate Organization HTTP control plane isolated until each is explicitly admitted. | No new caller of the ListingKit Task bridge and no claim that all ImageAgent code is legacy. Retire only the task-input bridge after a current authoritative input and caller cutover; do not add fallback/dual catalogs or silently install isolated Organization governance into the default worker. |
| Console/BFF versus old Task-first UI: current `/workbench` Console, Source Account and Product Review client/BFF paths; old `app/listing-kits/[taskId]/*` and `components/listingkit/tasks/*` projections | Current Console/BFF/review surfaces are **CURRENT** by capability; old Task-first ListingKit product projection is **RETIRE**. Directory/name alone is not authority. Shared leaf UI primitives are classified by behavior when touched. | Current and retired responsibilities coexist in `web/listingkit-ui`. PR #373 proves the Source Account BFF/client merge; REV-1 client/proxy code exists separately from the old task projection. | #386 frontend lint/typecheck/tests PASS; SA2 controlled BFF chain PASS. A successful build/test is not proof of default Go mount, deployment health or user acceptance: those remain **NOT_RUN** unless separately evidenced. | New Console/BFF calls target current Organization-scoped application contracts and truthful unavailable states. Reuse neutral leaf primitives only after ownership review. | No new Task-first Product flow, BusinessTask/legacy Task synchronization, legacy route fallback, or blanket retirement of the whole web tree. Retire page/caller slices after their current replacement is mounted and accepted. |
| Default mixed edge: `internal/app/httpapi/composition_builder.go` -> old source repository/crawler and root ListingKit | Remaining callers are EXTRACT/RETIRE debt; current Sourcing/Catalog remain the fact owners. | #663 removes only the sourcehandoff Task constructor/module/route, dedicated package and runner. Other source repository/crawler and root ListingKit consumers remain for #29/#301. | Historical #386 observations remain evidence of that baseline; this source retirement does not prove deployment or provider acceptance. | Current source applications consume their existing identity/access/publication owners directly. | Keep the retired handoff absent; no legacy forwarding, identity mapping, old-data migration, fallback or dual facts. |

## 3. Crawler / sourcing direction

Treat Amazon/1688 crawling as source adapters:

```text
internal/integration/crawler/amazon
internal/integration/crawler/a1688
        ↓
internal/product/sourcing
        ↓
ProductSnapshot / ApprovedAsset / downstream Listing
```

Rules:

- crawler packages own fetching, browser/runtime adaptation, parsing, and raw extraction;
- Product Sourcing owns `SourceIdentity`, `SourceEnvelope`, normalization, lineage, warnings, and handoff to canonical Product facts;
- marketplace publishing does not own source crawling;
- Product packages do not import crawler/integration/legacy ListingKit packages directly.

The old `internal/compatibility/listingkit/sourcehandoff/a1688` Task path is retired under #663. Current Sourcing/Catalog applications retain the independently valid source behavior. Other legacy consumers remain separately owned; no old-task/data migration, profile reuse or compatibility transition is introduced.

## 4. ListingKit direction

`internal/listingkit` and `internal/compatibility/listingkit` are **drain targets**, not target layers.

When touching them:

1. identify the actual reusable behavior;
2. identify the current owner (`listing`, `marketplace`, `product`, `integration`, `app`, etc.);
3. put new/rewritten behavior behind that current owner's contract;
4. switch callers;
5. remove the legacy dependency/path.

Do not add a new internal compatibility wrapper just because retirement is inconvenient.

## 5. Immediate landing zones

| New work type | Preferred landing zone | Do not default to |
| --- | --- | --- |
| Amazon source crawler | `internal/integration/crawler/amazon` | current AmazonProcessor/browser/extractor and source port; numeric Task/HTTP/cache factory closure retired by #867 |
| 1688 source crawler | `internal/integration/crawler/a1688` | `internal/crawler/alibaba1688`, compatibility handoff |
| Source normalization / lineage | `internal/product/sourcing` | crawler DTOs, root ListingKit |
| Product facts/assets/enrichment/image | `internal/product/{catalog,asset,enrichment,image}` | retired ProductEnrich/ProductImage/catalog/asset roots |
| Marketplace rules | `internal/marketplace/<platform>/*` | root ListingKit or compatibility tree |
| Listing orchestration/submission | `internal/listing/*` / existing single submission owner | another platform-specific state machine |
| Runtime assembly | `internal/app/*` | business packages |
| External client/integration | `internal/integration/<system>` | Product/Listing business owners |
| Commerce Tools | `internal/commercetool` contract + narrow domain adapters | direct DB/provider/marketplace-client access |
| Organization wallet / monetary ledger | `internal/ledger/money` | `internal/listingsubscription`, UI/BFF balance calculation, orgresource |
| Commercial offers / quotes / orders | `internal/commercial/billing` | `internal/listingsubscription`, referral economics, money ledger tables |
| Purchased AI points/data rows/store periods | source-bound purchase contract in `internal/ledger/orgresource` | generic positive mint or UI-side balance mutation |
| Agent runtime/capability | current Agent/AI Capability owners | legacy Task/Workflow ownership |
| Organization/membership/identity policy | current Organization/identity owner | `internal/tenantbridge` or new compatibility package |
| New AI Workbench UI | #298/Figma product projection | old Task-first ListingKit IA |

## 6. Legacy drain rules

The active rules from `legacy-register.md` are mandatory:

- no new `internal/compatibility/*` consumers;
- no new `internal/tenantbridge` consumers;
- retired package roots stay absent;
- existing legacy consumer counts should monotonically decrease;
- reusable behavior is extracted, not wrapped;
- retirement does not leave permanent fallback/dual-read/dual-write paths.

## 7. Existing code usage

Before moving or reusing code:

1. identify the primary business owner;
2. classify the legacy behavior as `EXTRACT` or `RETIRE`;
3. if a file mixes owners, split behavior by ownership rather than moving the whole legacy abstraction;
4. use `docs/refactoring/legacy-register.md` for active drain decisions;
5. update this map when a major ownership boundary is completed.

## 6. Phase 2 closure landing rules

Runtime ownership remains with Platform/Integration and application assembly. Legacy business consumers drain to their current owners: the product owner uses `internal/product/*` and keeps the retired Product roots absent; the marketplace owner extracts platform rules under `internal/marketplace/*`; the organization owner handles current identity contracts while #301 coordinates tenantbridge consumer retirement. #29 owns the remaining ListingKit extraction. These are EXTRACT destinations followed by RETIRE, never new compatibility landing zones.
