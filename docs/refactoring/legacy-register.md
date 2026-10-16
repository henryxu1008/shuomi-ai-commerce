# Legacy Register

> Status: Active legacy handling register

> #29 / #301 [旧数字任务执行链退休设计](../architecture/legacy-numeric-task-execution-retirement-boundary.md)
> is user-confirmed and implementation-admitted after #726. The unregistered
> task/status/processor, root Amazon executor and exclusive distributed crawler
> sources are absent. Mixed fetcher local/remote contracts, platform API/rules,
> shared repositories and standalone SHEIN login remain under their original owners.
> Source retirement has not changed runtime/data or internal-trial gates.

> #29 / #301 [ownership-controller / auto-shard retirement](../architecture/legacy-ownership-controller-retirement-boundary.md)
> is user-confirmed and implementation-admitted in #738. The old workload,
> unconsumed heavy/lite ConfigMap overlay and exclusive configuration closure are
> absent from this source candidate. Shared config/helpers, valid repositories,
> current acquisition, Store observations and standalone login retain their owners.
> Real ConfigMaps, ownership keys/locks, instances and data have not been touched.

> #529 / #301 [发布身份预检退休设计](../architecture/release-identity-preflight-retirement.md)
> is user-confirmed and implementation-admitted. The legacy API/UI and PS
> entrypoints now fail before operations; identity-preflight, its exclusive
> readers/resolver and packaging are absent. The exact frozen import set is
> empty, with all three depguard exclusions removed. Standalone SHEIN worker,
> current schema and valid security checks remain. Production stays NOT_READY;
> no historical executor, instance, IAM or data has been changed.
> Policy: `docs/refactoring/legacy-hard-cut-policy.md`  
> Last reviewed: 2026-09-11 (bounded #386 calibration; full scan baseline unchanged)
> Repository scan baseline: `main @ 6e3b87208e2c4e51039b95145a2377edfbb9b1cb`
> Bounded mixed-edge calibration: Issue #386 against `main @ 08a6195c7aa8528acdec0bf0f4e742a8c4998e6d`

This register records known obsolete or partially migrated internal abstractions so implementation agents do not independently invent compatibility work.

Current repository policy has only two legacy decisions:

- `EXTRACT` — reusable behavior is moved/reimplemented under the current owner; the new architecture must stop depending on the legacy owner.
- `RETIRE` — obsolete design is not extended or internally supported; remove it after cutover.

There is currently **no Compatibility category**. A directory or type named `compatibility`, `legacy`, `bridge`, `adapter`, or `facade` does not create an exception to this rule.

## Greenfield baseline

[PD-GREENFIELD-NO-LEGACY-MIGRATION-2026-09-08](../product/greenfield-no-legacy-migration.md) controls all future business work: use a fresh installation, empty business data and current models. Rows below that describe a historical migration/cutover are evidence of existing debt only; they do not authorize old-data migration/backfill, old-ID mapping, legacy wrapper/adapter, fallback, dual read/write/synchronization, tenantbridge consumer or a second fact source. Existing legacy code is still only `EXTRACT | RETIRE`.

An external compatibility obligation is never inferred from a row, historical Issue/PR, test or code. Concrete current evidence must be reported to the user; there is no executable legacy-migration/compatibility Exception, and only a new explicit product decision from the user can change that prohibition. Reviewers and Agents cannot authorize it themselves.

## 1. Repository scan summary

### Account surface clarification (#409)

The current Account profile chain on main `aff6b8def46cf7c2d989408d115fa3c9628e585e`
is CurrentIdentity -> `workbenchcontext/httpapi` -> `authidentity.SelfProfileReader`
-> `authruntime/zitadel.UserInfoClient`, consumed by the dedicated account BFF
and current Console. It is not the old profile/migration owner and must not be
retired solely because it resides in `web/listingkit-ui`. All disclosed profile
fields are read-only; absent claims or write ownership cannot be supplied by a
legacy fallback, old-ID mapping or simulated save. Existing Task-first entries
below remain RETIRE; this clarification adds no compatibility category and
does not claim other Account slices or real-environment acceptance.

### Current committed-operation projection (#412)

The bounded Account audit projection reuses current immutable
`source_account_operations` receipts through `sourceaccountregistry.HistoryService`
and its existing persistence owner. `internal/app/accountaudit` only projects
allowlisted fields for successful `register / enable / disable` operations.
It does not consume old `internal/sourceaccount`, `tenantbridge`, or Task-first
abstractions. Reusing these current receipts is not legacy compatibility and
adds no migration, fallback, duplicate facts or compatibility exception.
This is not a complete enterprise/security audit platform; empty results do
not prove absence of all enterprise activity. Existing historical calibration
and RETIRE decisions below remain unchanged. Code mounting does not establish
deployment or real-environment acceptance.

### Physically retired — keep absent

The following old production package roots were not present on the scan baseline and must remain absent:

- `internal/productenrich`
- `internal/productimage`
- `internal/catalog`
- `internal/asset`
- `internal/imageasset`

Their current owners exist under:

```text
internal/product/catalog
internal/product/asset
internal/product/enrichment
internal/product/image
internal/product/sourcing
```

Historical plans, test names, baselines, and architecture guards may still mention the retired names. Such references are evidence or anti-regression guards, not permission to recreate the packages.

### Active legacy hotspots

The scan found four high-value areas that still affect production paths:

1. `internal/compatibility/listingkit`
2. root `internal/listingkit`
3. `internal/tenantbridge`
4. `web/listingkit-ui` old Task-first / ListingKit product projection

These areas are drain targets. New Product, Marketplace, Agent, Tool, BusinessTask, Store, and Console work must not add new dependency on them merely because they already exist.

### #301 bounded 1688 numeric source-account retirement

The unused `httpx.Crawler1688Handler`, repository-backed 1688 account-profile
resolver and entire old `internal/sourceaccount` package (including its GORM
schema/bootstrap) are retired. Their constructors had no production consumers;
the current build no longer pulls old numeric account persistence through the
shared crawler/HTTP packages. No database, profile or retained runtime was changed.

The current `sourceaccountregistry` remains the Organization-scoped registration
owner; registration still means `pending_connection`, not a usable login.
Current anonymous acquisition, Browser Capture and local-agent collection retain
their existing owners and contracts. #719 also retired the
unconstructed 1688 queue `Service`, `Crawler1688Processor`, numeric `AccountProfile`
interface/resolver and account-assisted browser/fallback chain. No current
consumer constructed that service or called the account-profile methods.
`Alibaba1688Processor.Prepare/Process/ProcessBatch/Shutdown`, its public browser
path, extractors, typed failure classification and cancellation/cleanup remain.
The original configured browser factory stays available; anonymous attempts
continue using a fresh non-persistent public manager. Shared result/worker/HTTP
behavior, scoped protections and Amazon routes remain with their original owners.

The unregistered `internal/app/productsourcing/httpapi` package is retired under
#301: its numeric source-account/Store import DTO, unimplemented Importer,
snapshot/receipt projections and exclusive tests/fixture had no production
composition or consumer. Current acquisition and Browser Capture use their own
admitted HTTP/application owners; do not mount, wrap or recreate the old import.
The shared crawler task's `SourceAccountID`, `SourceAccessMode` and compatibility
inference, plus the result's source-access/fallback metadata, are also retired
after the old 1688 queue/Profile consumers disappeared. Trusted `TenantID`,
Amazon task fields, task/result state behavior, scoped result access, Redis,
browser, workerpool and HTTP infrastructure remain. This does not retire current
Organization-scoped source-account registration or the current sourcing producer.

The standalone SHEIN worker and its original account persistence, Store sync,
Redis/browser/profile, health and shutdown remain unchanged. Source retirement
does not establish provider, user, deployed-runtime or production acceptance.

### #29 bounded backend retirement

The mixed root Go package, old Task/API/Workspace/Temporal chains and default
HTTP assembly are absent in this checkout under #29. Valid route-role transport
now lives in `internal/authz/httpapi`; request-role context stays with the existing
`internal/shared/aiidentity` transport. Current image catalog/byte/URL functions
are extracted in place in `app/httpapi/image_asset_transport.go`, and Temporal
client validation stays in the current application runtime. Submission owns its
original in-progress error directly. No Task mapping, schema/history migration,
new identity authority or second execution owner is introduced.

Both current-application and image-agent-temporal-worker are checked by their
actual transitive dependency graph, not just their route registrations. Remaining
`listingkit/sheinsync`, its exact store adapter and standalone identity/history
leaves are still registered retirement debt under #29/#301/#529, not a current
landing zone. Existing numeric consumers call their original leaf without the
removed root aliases; no new current consumer is admitted.

The current image recipe requires the existing private current-application
manifest. It does not implement the old API/YAML rollout contract. The old
API/UI source workflows now have only fixed permission-free failure entries.
The original security verifier, OIDC/RBAC negative fixtures and independent
worker/schema/canary contracts remain under their existing owners. Current worker/schema capabilities
are retained; legacy API deployment of this artifact is NOT_READY. No runtime,
data, provider or user acceptance is claimed.

### #29 / #301 standalone listing CLI and exclusive scheduler retirement

The user-confirmed [boundary](../architecture/legacy-listing-cli-retirement-boundary.md)
§5/§6/§6.1 admits this bounded RETIRE batch. The three CLIs, exclusive app
bootstrap/consumer/runner/scheduler, local runtime, SHEIN sync adapters and
platform module/scheduler/task-processor glue are physically absent. Retained
platform clients/rules, generic infrastructure and current Store observations
are not deleted. CI replaces only the obsolete build target with the current
application and standalone login worker; required/security checks remain.
Taskstatus, ListingAdmin and root ListingRuntime retain their original contracts.
The control-plane retained in #723 subsequently exits under #726 below. Mixed
deployment sources retained ownership-controller and configuration in that batch;
the exclusive declaration/configuration closure subsequently retires under #738.
Other effective configuration, Secret and monitoring objects remain; no historical
image is rebuilt or executed.
This is source retirement, not full numeric-task-chain closure, runtime/data
cleanup, deployment or a new internal-trial prerequisite. Production NOT_READY.

### #29 / #301 confirmed control-plane and numeric dispatch/recovery retirement

The user-confirmed [follow-up design](../architecture/legacy-listing-control-plane-retirement-boundary.md)
§5/§6/§6.1/§7 admits this bounded RETIRE batch after #723. The remaining
control-plane and store-986 platform recovery entrypoints, their exclusive
runtime/control library, cohort helpers/tests and operation packaging are absent.
Current acquisition, Store observations, valid shared contracts, required/security
checks and the independent SHEIN login worker retain their original owners.
ImportTask repository/claim/rollback/recovery races, package-external task/status
consumers and shared configuration were explicit debt in that bounded batch;
the ownership declaration subsequently retires under #738. No whole-package cleanup, current
consumer, compatibility bridge, runtime/data action or internal-trial prerequisite
is authorized. Both workflows remove only the retired runtime race target and
main CI removes its obsolete build line; design section 6.1 preserves all other
steps, pinned-SHA/release evidence, required and security checks. Source retirement
does not prove real-runtime cutover, user acceptance or production readiness.

## 2. Active register

### #529 / #301 release identity-preflight retirement

The old CLI/runtime, identitypreflight, ownerreconcile, userdirectory and
tenantbridge packages are physically absent. Their sole production consumer
was the retired preflight. No metadata owner, mapper, fallback or replacement
identity directory is created. The current image no longer packages that
binary; its dedicated image/Job and only the identity Deployment/resource
items in the aggregate runner, API Role and source policy are removed. The
exclusive old username/role Secret cleanup script and its protocol tests are
also retired after their only publishing consumers disappear.

API/UI normal, rollback and receipt-triggered publishing source paths and the
PowerShell wrapper always reject before credential, build, push or mutation.
Original Conftest/OPA and required CI selection continue; the actual fixed
failure entries are checked in full, with publish/permission/trigger/bypass
negative cases. OIDC, narrowed RBAC, immutable attestation and retained
worker/schema safety remain with their existing helpers and tests.

Workbench uses the verified effective Organization and scoped role directory;
its valid response and role-isolation regression is preserved without the
obsolete resolver hook. Opaque numeric Organization strings remain valid.
The exact bridge freeze is now empty (three to zero); deny rules and parser
logic remain. Older batch calibrations below describe their then-state only.
The standalone SHEIN worker's original numeric account persistence remains
separate #301 debt. No live revoke, deployment, data/profile deletion, user
acceptance or production readiness is implied.

### #301 confirmed SHEIN worker boundary and bounded retirement

The [identity/persistence boundary design](../architecture/shein-worker-identity-persistence-boundary.md)
records current owners and exact deletion conditions. It preserves the worker's
repository-backed account provider, original DB/Redis/Profile/session/state and
Store sync/duplicate lookup. Current Organization/Store/official business paths
must not consume that old identity or browser login state. The unused alternate
`sheinloginmanaged.AccountProvider` and isolated `sheinlogin.VerifyCodeRequest`
and its exclusive helpers/tests are RETIRE and deleted under confirmed §7. Shared
ListingAdmin/ListingRuntime contracts still used by current applications are
not blanket retirement targets. Design section 5.1 records the confirmed
unqualified duplicate-store lookup defect: preserving the capability requires
tenant-qualified lookup, response-scope checks and no Store mutation on lookup
failure. This existing tenant-isolation contract is enforced at the helper and
service effect boundary, with RED→GREEN coverage through the real GORM adapter,
every-page tenant scope and invalid/failed lookup handling. Successful Redis
Cookie/attempt commits keep their original owner and are not rolled back by a
later Store sync failure. The repository provider, verification/cancellation,
durable attempts and independent worker remain. This adds no internal-trial
prerequisite and authorizes no runtime, data or provider action; candidate
checks and real-environment acceptance remain separate.


### #301 unused SHEIN login HTTP retirement

The standalone worker was the only production consumer of
`sheinlogin/bootstrap`, and used only its Service and Close. Its unused HTTP
Handler/Module, API routes and numeric header resolver are RETIRE, including
tests dedicated to that retired protocol. Bootstrap now builds only the same
service; the original repository, Redis, profile, browser, Store sync, worker,
health endpoint and close behavior are retained. The HTTP cross-tenant account
rejection assertion is preserved at the unchanged service boundary alongside
the existing account/session isolation tests.

The worker's actual Go dependency graph no longer contains tenantbridge.
Only the two deleted tenant-context import edges and their exact depguard
exclusions are removed, reducing the frozen baseline from five to three.
Those remaining edges are the release identity-preflight runtime/test and the
current Workbench resolver-isolation test. The preflight's shared readers,
resolver and release protection are unchanged; they remain debt under #529/#301.
No data, IAM, browser login, provider or deployment operation was executed.
The worker's existing numeric account persistence is retained, not reclassified
as current Organization ownership or declared fully retired.

### #529 / #301 obsolete identity preparation retirement

The Yudao-to-ZITADEL migration dry-run script and the entire
`internal/tenantbridge/bootstrap` package, including its dedicated tests, are
RETIRE. The script generated old account/tenant/email/login/metadata migration
plans; bootstrap probed historical ZITADEL databases and installed a global
legacy resolver. Neither has a current production consumer. No maintained
default command compiles bootstrap, and no replacement preparation path is added.

Their two bootstrap-to-tenantbridge import edges and exact depguard exclusions
are removed, reducing that batch's frozen baseline from seven to five. Its then-five
remaining edges belonged to the unchanged identity-preflight, SHEIN transport and current
Workbench isolation test. The tenantbridge resolver itself, shared readers,
current worker/schema paths and release protection remain unchanged. Historical
plan and dated inventory entries are evidence, not callable current preparation.
No old script, database, IAM or provider operation was executed.

### #529 legacy owner-scope tool retirement

The subsequent #529 owner-scope batch retires the two old reconciliation/seed
commands, their exclusive app/runtime/options, PS wrappers and old tests. The
fixed historical seed and legacy-ID backfill write APIs are removed from the
shared ownerreconcile library, together with their exclusive planning helpers
and write-only tests. No callable ApplyUnique or exception-seed API remains.

Identity preflight keeps the same Queryer, inventory, identities, exception
reader and DryRun call. Only its unused Beginner write-handle assignment is
removed with that retired interface. Read-only scanning, metadata validation,
fixed-report matching and fail-closed tests remain; the shared package is still
registered debt, not retired or a new current owner. Passive query/template
metadata is unchanged. That batch preserved release workflow/OIDC/RBAC/canary/
schema protection and the then-seven bridge edges; no existing database or
registry was changed. The subsequent identity-preparation batch above removes
only two unused bootstrap edges.

### #529 Source Account historical migration tool retirement

`cmd/source-account-ownership-preflight` was the only production consumer of
`internal/integration/persistence/sourceaccount/ownershipmigration`; its only
operational wrapper was `scripts/source-account-ownership-preflight.ps1`. All
three are RETIRE under the approved greenfield decision. The unused A/B1
prepare/schema/snapshot/receipt and platform-specific profile validation source,
embedded SQL and tests that assert only the cancelled migration are removed.
No replacement migration owner, facade, command or runtime path is introduced.

Current SourceAccountRegistry, its app-owned schema initializer and valid
authorization/organization/idempotency tests remain. The maintained command
inventory is narrowed; the superseded operational report and historical A/B1
evidence remain explicitly historical. Existing source-absence and actual
command-dependency guards prevent revival without a new scan framework.

That bounded removal had no tenantbridge import edge and preserved the then-seven
edge freeze. Other Store-history/identity-preflight and standalone SHEIN consumers
remain #529/#301 debt, including the unchanged release identity gate. This is
source retirement, not a real migration, cleanup, deployment or user acceptance.

### #678 current-application tenantbridge dependency exit

The dedicated SHEIN worker's valid process assembly is extracted directly to
`app/runtime/sheinloginworker`; its cmd is the sole consumer. The original shared
HTTP entrypoint is deleted without a wrapper. This isolates the existing worker,
not a new domain/legacy facade. Its database/account repository, Redis, health,
signal and cleanup semantics remain unchanged.

The old ListingAdmin HTTP projection has no production constructor/registration
caller after the retired default API. Its handlers and raw-header/list-scope
helpers are RETIRE; repository/API/DTO and valid ownership tests remain. This
also removes the bridge dependency that the current local SHEIN draft inherited
through the broad historical publishing package, without changing draft bytes,
fact ownership, authorization or Submission/recovery contracts.

Current-application and ImageAgent worker production graphs have no tenantbridge,
compatibility or ListingKit. The two deleted exact bridge edges/exclusions are
removed; its then-seven remaining edges (including tests/self-use) remained registered debt.
The later #301 batch above retires unused standalone SHEIN numeric HTTP.
Historical preflight consumers and worker numeric account persistence are not
current Organization owners and remain with #301/#529. No migration/data/provider action
or deployment is part of this source delivery.

| Legacy area / abstraction | Decision | Observed state | Reusable behavior | Current owner / destination | Exit condition / related work |
| --- | --- | --- | --- | --- | --- |
| `internal/productenrich` task / queue / worker / API architecture | RETIRE | Production package root is absent on scan baseline | enrichment analysis, normalization, scoring, proposal logic that remains independently valid | `internal/product/enrichment` + AI Capability where applicable | Keep old package absent. Do not recreate workflow/task ownership. |
| `internal/productimage` task / queue / worker / API architecture | RETIRE | Production package root is absent | provider-neutral image analysis/transformation logic | `internal/product/image` + ImageAgent | Keep old package absent. ImageAgent remains workflow/budget/retry/recovery/approval owner. |
| old `internal/catalog` | RETIRE | Production package root is absent | canonical product behavior already moved | `internal/product/catalog` | Keep absent; new callers use current Product contract. |
| old `internal/asset` | RETIRE | Production package root is absent | approved asset behavior already moved | `internal/product/asset` | Keep absent; no second asset authority. |
| old `internal/imageasset` | RETIRE | Production package root is absent | independently valid image/asset helpers only | `internal/product/image` or `internal/product/asset` by ownership | Keep absent; do not recreate as convenience package. |
| `internal/compatibility/listingkit` as a long-term facade / landing zone | RETIRE | Drain shells and sourcehandoff are absent under #663/#29 | Existing current owners only; no compatibility ownership | Product/Listing/Marketplace/Integration/App according to responsibility | Keep absent; no new consumer, fallback or facade. |
| `internal/compatibility/listingkit/preview_adapter.go` | RETIRE | Removed by the #300 guard slice after confirming no production caller | none required after current `internal/listing/preview` projection exists | `internal/listing/preview` | Keep adapter and dedicated test absent; current callers use the preview owner directly. |
| `internal/compatibility/listingkit/sourcehandoff/a1688` | RETIRE | Package, old `/api/v1/product-sourcing/1688/listingkit/tasks` route, dedicated DTOs/tests and runtime runner are absent in this checkout under #663. Deployment and provider acceptance are separate. | Existing Sourcing/Catalog owners retain valid source evidence, immutable publication, access and receipt recovery; historical Task requests and identity mapping are not reused. | Product Sourcing/Catalog + current Store/Organization/Application | Keep old route/package/runner absent; no old data migration, forwarding or compatibility. Other legacy owners remain with #29/#301/#529. |
| root `internal/listingkit` mixed ownership | RETIRE / EXTRACT valid leaves | Mixed root Go package and its Task/API/default runtime chain are absent under #29 | Unchanged current role/context/Temporal/image transport and Submission error | Existing authz/aiidentity/app runtime/image transport and Listing/Submission owners | Keep root absent. Remaining numeric sync and identity/history leaf debt is explicitly separate; source retirement is not deployment or whole-parent completion. |
| marketplace rules still owned by root ListingKit | EXTRACT | Remaining rule/policy ownership is part of #29 hard-cut | deterministic platform rules/policies/readiness logic | `internal/marketplace/*` / stable Listing seams | Move rule to current owner and switch callers. Do not wrap the old rule owner. |
| `internal/tenantbridge` Organization ↔ legacy Yudao numeric tenant mapping | RETIRE | Package and all three frozen imports are absent; Workbench current Organization/scoped role behavior is preserved, including opaque numeric IDs | existing current-identity behavior already belongs to its current owner; numeric mapping is not reusable for a new system | current Organization/identity ownership + each domain's current persistence boundary | The admitted old release preflight consumer and its resolver are retired. Keep absent. No new consumer, old-data migration, old-ID mapping, fallback or dual authority. |
| `web/listingkit-ui` Task-first / CreateListingKitTask product projection | RETIRE | #664 removes old `/listing-kits/**` pages, their dedicated components/hooks/Task DTOs and Task BFF; the current frontend application and Console remain | current shared UI primitives, Auth.js/ZITADEL and independently valid readonly provider transport | #298 / final Figma Product Projection; current Store Center / AI Workbench / product pages | Keep old routes/request chains absent, without redirect, fallback or Task mapping. Source retirement does not prove deployment or user acceptance. |
| Generic independent Listing Workspace (#27) | RETIRE | Backlog abstraction already closed | independently valid Product/Listing UI behavior only if required by current IA | final Product Projection pages defined by Figma authority | Do not revive a generic top-level Listing Workspace. |
| Old Product Task Center / unified internal Task Dashboard (#35) | RETIRE | Old issue closed; final UI now has a different BusinessTask Center | diagnostics may be projected if useful | #298 BusinessTask Product Projection; internal execution remains with Workflow/Task owners | BusinessTask is a new user-facing object, not a renamed legacy Task. No bidirectional sync. |
| Task-first Product architecture | RETIRE | Superseded by Product/Listing facts + BusinessTask | none as an ownership model | Product/Listing domain facts + BusinessTask projection | Internal Task/Workflow stays execution infrastructure. |
| Platform-specific TEMU/Amazon Workbench expansion model | RETIRE | Superseded product projection | marketplace clients, rules, submission adapters and other valid capability code | shared Marketplace/Listing capability; Store Center / AI Workbench projection | #31/#32 expand capabilities without copying Workbenches. |
| Independent top-level Product Center / Listing Center assumption | RETIRE | Conflicts with final Figma IA | Product/Listing facts and useful screens only | Figma `31:463`; `docs/product/final-ui-ia-authority.md` | Domain facts remain; obsolete navigation assumptions do not. |
| Old tenant quota / task-limit resource model (#42) | RETIRE | Superseded by **PD-GREENFIELD-NO-LEGACY-MIGRATION-2026-09-08** | none; current entitlement facts are created by their current owners, not carried over | Resource Ledger / Store Service / entitlement owners | Do not restore task-count quota as resource authority or migrate legacy quota values. |
| “Build ZITADEL tenant/role model from scratch” backlog abstraction (#39) | RETIRE | Superseded by current identity/Organization implementation | existing verified identity/organization behavior | current ZITADEL + Organization/RBAC boundaries | Remaining work is release/staging/production acceptance. |
| Separate Product/Asset/Listing fact source introduced for Agent convenience | RETIRE | Forbidden architectural duplication | none | existing canonical domain owners | Agent output is Proposal/trace, never a second business fact source. |
| Parallel Agent RBAC / Tool Registry / durable retry/state-machine ownership | RETIRE | Forbidden architectural duplication | none | current authorization, #133 Tool Registry, Temporal/queue/submission owners | Do not isolate Agent code by cloning platform contracts. |

### 2.1 Bounded mixed-edge calibration (#386)

This section calibrates only the exact edges below; it does not claim a new full-repository scan and does not alter the historical `6e3b87208e2c4e51039b95145a2377edfbb9b1cb` baseline. Current owners, code/merge/isolated/default/runtime distinctions, and fixed objects SA1/SA2, SRC-1/Catalog, PA-1, DRAFT-S1, REV-1 and SUB-K1 are recorded once in [`module-target-mapping.md`](module-target-mapping.md#21-bounded-current-delivery-calibration-386). This register records their legacy counterpart and exit only.

| Exact legacy / mixed edge | Decision | Current replacement / bounded evidence | Reusable behavior | Exit condition / prohibited new work |
| --- | --- | --- | --- | --- |
| Old `internal/sourceaccount`, unused 1688 HTTP/resolver and numeric queue/Profile chain | RETIRE | Old composition was removed under #29; #301 retires the old account package and remaining unconstructed queue service/worker/Profile methods. Current registration remains `sourceaccountregistry` + its existing persistence/schema/application boundary. | Anonymous browser processing, typed public failure classification and shared scoped result protection remain in their original owners. Provider connection is separate from registration. | Keep old package, queue/worker, numeric Profile and HTTP submission absent. No registry wrapper, old table/profile dependency, tenantbridge, ID migration, fallback, dual read/write or second account fact source. |
| `composition_builder.go` -> `internal/compatibility/listingkit/sourcehandoff/a1688.NewTaskCommandService` -> legacy Task/request | RETIRE | Handoff construction, module field/accessor/registration and source-only validator removed under #663. Original unrelated source repository/crawler and ListingKit callers remain. | Current Sourcing/Catalog source normalization, identity, authorization, evidence and atomic publication are already owned by current applications. | Current admitted source applications, without old Task side effects. | No wrapper, old ID/data migration, forwarding, dual facts or new tenantbridge consumer. Remaining root ListingKit/tenantbridge debt belongs to #29/#301. |
| `internal/app/httpapi.newImageAgentAuthorizedAssetCatalog(listingkitstore.NewTaskRepository(...))` and `imageAgentCatalogFromTask*` accepting root `listingkit.Task` | RETIRE input bridge | ImageAgent domain/workflow/store, Temporal worker, `cmd/image-agent-temporal-worker` and deployment definitions remain **CURRENT**. Organization worker governance is the separate isolated `BuildOrganizationImageCapabilities` + `OrganizationExecutionAuthorizer`; default `buildProductionImageCapabilities` does not install its `reviewGovernance`/invocation recorder, and the default Temporal activity assembly does not populate `imageagenttemporal.ActivityDependencies.ExecutionAuthorizer`. `NewImageAgentOrganizationApplication` is a separate current opt-in HTTP/catalog/service assembly, also **NOT_PRESENT** in default composition. | Deterministic asset selection/validation only when moved behind a current Product/Asset-owned catalog input; ImageAgent behavior and isolated Organization governance are not legacy. | Replace the default task catalog with an admitted authoritative current input and switch callers, then delete the Task bridge. No fallback/dual catalog, no blanket retirement of ImageAgent, and no silent default-worker installation of the isolated Organization governance. |
| Root `internal/listingkit` Task-first submission/workspace paths beside `internal/listing/submission.ExecutionKernel`, DRAFT-S1 and other extracted leaf owners | EXTRACT by behavior; RETIRE old path after caller cutover | SUB-K1 kernel is merged in PR #385 but has no non-test application/default caller; DRAFT-S1 is merged in PR #377 but remains isolated. Their existence does not mark the whole root or whole submission chain current. | Deterministic marketplace/readiness rules and provider-neutral submission behavior only under `internal/listing/*` or `internal/marketplace/*`; `ListingKitAuthorizer` and leaf packages are not legacy merely by name. | Cut over exact callers/provider adapters under separate admission. Do not wrap root ListingKit, clone a state machine, or claim `submit_lock`/provider/default deployment complete. #29 owns the root drain. |
| Former `web/listingkit-ui/src/app/listing-kits/**` / `src/components/listingkit/**` Task-first and platform-island projection | RETIRE | Pages and dedicated request chains physically removed under #664; current `/workbench`, Source Account/Review/Store BFFs remain. The SDS readonly BFF's shared identity transport is extracted to `src/lib/server/upstream-identity.ts`, with unchanged behavior and valid tests. | Existing UI/shared primitives and current identity/provider transport only; no legacy Task/DTO/state ownership. | #298 owns current projection. Retired routes, shell, DTOs and Task mutations stay absent; no fallback, Task mapping or second authority. Browser/provider/runtime acceptance remains separate. |
| Remaining `internal/tenantbridge` callers, including the legacy A1688 handoff and other registered legacy paths | EXTRACT only independently valid current identity behavior; RETIRE numeric bridge | Current SA1/SRC-1/REV-1 applications use verified Organization identity without making tenantbridge a new-system dependency; the legacy default chain still proves the bridge is not retired. | Current Organization identity/access behavior under its actual owner, never numeric mapping as a reusable abstraction. | Caller-by-caller removal until zero consumers, then delete package. No new consumer, migration/backfill, old-ID mapping, fallback, dual authority or relocation into another compatibility facade. #301 owns the drain. |

## 3. Current drain rules

### Store Center current-application admission (#552)

Observed at `8478800e31bf2e126bd03fae5ec45536056d823e`: the Store repository still
uses `compatibilityStateForNewStore/Save`, lifecycle/record/service dual writes,
and historical resolution fields; its enable persistence path requires service
resume. The old HTTP builder opens `cfg.Database` with `OpenShared`. These paths
cannot be newly mounted into current application under the greenfield decision.

- Decision: **EXTRACT / RETIRE**.
- Reusable behavior: Organization-scoped identity/profile, CAS, safe audit,
  existing Subscription StoreQuotaLedger admission/replay, core CRUD and Console/BFF.
- Current owner: Store Center, Subscription quota owner and application assembly.
- Cutover/deletion condition: current consumers switch together to the admitted
  single current Store contract; no old DTO/state/shared-builder fallback or
  history backfill is introduced. Current service/resource execution remains its
  existing owner and is not blanket-retired.
- Related Issue/design: [#552](https://github.com/qq550723504/task-processor/issues/552),
  [Store Center current application v1](../architecture/store-center-current-application-v1.md).
  The design's explicit admission status must pass before formal implementation.

### `internal/compatibility/*`

- Existing production callers are migration debt, not precedent.
- New code must not add another consumer.
- A reusable behavior is moved to its current owner first; callers switch there directly.
- Do not create a new `internal/compatibility/<domain>` package to solve a new feature.
- Consumer count should monotonically decrease.

### `internal/tenantbridge`

- Do not add new consumers of legacy numeric tenant resolution.
- New schemas and new domain contracts use current Organization identity directly.
- Existing callers are drained by their owning domain, not hidden behind a new facade.
- Under PD-GREENFIELD-NO-LEGACY-MIGRATION-2026-09-08, drain and RETIRE without legacy migration/backfill, old-ID mapping or cutover. Normal schema evolution inside the new system remains separate.

### `web/listingkit-ui`

- Existing commercial pages may continue operating until their replacement is actually ready; this is current runtime reality, not an architecture compatibility requirement.
- New product IA follows Figma and `docs/product/final-ui-ia-authority.md`.
- Do not reproduce old Task-first semantics inside BusinessTask just to make both models align.

## 4. How to use this register

Before reusing a legacy package/service/type:

1. Find the matching row.
2. Follow its `EXTRACT` or `RETIRE` decision.
3. For `EXTRACT`, name the exact reusable behavior and current owner before coding.
4. For `RETIRE`, do not add a new caller, wrapper, fallback, or synchronization path.
5. If useful behavior is not covered, update this register before propagating the dependency.
6. Do not create a third `compatibility` state locally.

If code turns out to be reusable, tests should be rewritten against the current owner/contract. If a test only proves an obsolete implementation detail, retire the test with the code.

## 5. New discoveries

When a new legacy area is discovered, record:

```text
Legacy area:
Decision: EXTRACT | RETIRE
Observed state:
Reusable behavior:
Current owner:
Cutover/deletion condition:
Related issue/PR:
```

If concrete, current externally observable contract evidence appears, stop and report it to the user. Only a new explicit user product decision can permit a compatibility exception; it must not silently become a new register category.

## 6. Automated guard baseline (#300 slice)

`tests/import_boundaries_test.go` enforces `TestLegacyConsumersStayWithinDrainBaseline` using the existing tracked-source/import parser. `tests/legacy_hard_cut_test.go` records the exact file-to-import drain edges at `main @ 2fd42cc06`. The baseline includes production, test, and intra-legacy callers across build tags and nested modules. Only fixture `testdata` trees are excluded. Stage new Go files before local verification; CI checks the committed tree and its workflow is triggered by `**/*.go` so nested-module source-only changes cannot bypass the guard.

- New files and new imports in existing callers fail, including new compatibility domains and tenantbridge subpackages.
- Remove a drained edge from the baseline in the same PR. Stale entries fail so removed callers cannot silently return. Remove its exact depguard file exclusion when its last import of that legacy root disappears.
- `legacy_compatibility_consumer_freeze` and `legacy_tenantbridge_consumer_freeze` in `.golangci.yml` reject new caller files; the import guard checks exact edges inside the remaining files. Do not expand either baseline for features. Baseline changes require review against this register and any explicitly approved exception.
- `TestCurrentOwnersDoNotImportLegacyListingKitRoot` and `current_owners_legacy_listingkit_root` block the mixed root service from Product, Listing, Marketplace, Agent, CommerceTool, Console, BusinessTask and Store Center packages, including tests. Existing extracted leaf contracts (for example `internal/listingkit/core`) remain subject to #29 and their existing boundary rules.
- Existing `TestPhase3LegacyProductRootsAreAbsent`, `TestPhase3LegacyProductImportDeclarationsAreAbsent`, and `phase3_retired_product_imports` keep ProductEnrich/ProductImage/catalog/asset/imageasset retired. The preview adapter guard scans the full compatibility package tree for the retired declaration and any dependency on the current preview owner, so renaming its files cannot revive the bridge.

Validation: `go test ./tests -count=1`, `go test ./... -count=1`, and `golangci-lint run --config .golangci.yml --enable-only depguard ./...` (existing Backend Checks CI).

This is a partial #300 delivery, not issue closure. BusinessTask dual-authority/constructor seams and additional Agent/Tool legacy runtime boundaries still need focused coverage as their current contracts land. #29, #30 and #301 own their respective extraction/cutover work; this guard does not perform those migrations. Static imports cannot prove the absence of every semantic fallback or dual write.
