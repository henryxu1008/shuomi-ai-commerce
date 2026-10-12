# Store observations internal preparation runtime

Status: IMPLEMENTATION_READY (2026-10-11). Design Basis: Independent Architecture, bounded deployment admission supplement. Independent review of base f1d77a84fa653813235ddbfeedac59ea6bb0a94c and design blob 3f54f87d35fa1da35c8b18883dda84008c4eb978 found no design BLOCKER.

## Product outcome and authority

The user explicitly requested: complete internal wiring, especially the Store sync module and background worker, while model/payment/SHEIN accounts will be supplied later; preserve existing data. A retained local installation can prepare the real observation repository, current HTTP consumers and Temporal worker without platform credentials. The user sees that internal wiring is ready and SHEIN configuration is pending; no fake products, orders, successful syncs or usable platform actions.

Reuse the frozen [Store observations v1](store-center-platform-observations-v1.md), [integration handoff](../engineering/store-observations-integration-handoff.md), [Store current application](store-center-current-application-v1.md), and current [Figma Product Authority](../product/final-ui-ia-authority.md): tg48P46SSXl6TBy9lZwg63 / 31:463, products 429:5323 and orders 429:5863. No new page/layout or business workflow. This explicit stage only replaces the startup dependency on an already configured official application; full-mode authorization, service/connection qualification and official provider contracts remain frozen.

Scope: opt-in preparationOnly Store observation configuration; existing module/worker lifecycle; accurate pending configuration capability/UI; existing schema-owner command and final Compose layer. Out of scope: model invocation, payments, real SHEIN access, platform writes, automatic sync, new permissions/owner/tables/state machine, legacy migration, acceptance tooling, production deployment and Issue closure.

## Owners and contract to consumer

SHEIN owns platform facts; existing Store owns identity, membership, connection and secrets. Existing shein_observations commands/syncs/records/heads remain the sole observation owner; the existing module-specific Temporal queue and worker remain the sole execution owner. No dummy provider, fallback, empty official application or alternate ledger is constructed.

`storeCenter.observations.preparationOnly=true` -> current-application validation -> existing scoped Store pool and strict schema/privilege preflight -> original observation repository plus live Authorization/Directory and an Access with no Official port -> existing module HTTP routes and Temporal worker -> current products/orders consumers. The process still starts its real worker before HTTP serving and stops it/clears readiness on failure or shutdown.

Preparation mode requires enabled Store, current exact membership authorization, valid literal loopback Temporal target, and **zero** configured official applications. It is never inferred from missing secrets. Default/false retains the original full-mode official application requirement and constructor. A configured application cannot silently be downgraded to preparation mode.

Preparation mode refuses any existing row in the four observation tables before constructing its worker. Therefore no retained sync command can be stranded or rewritten by a provider-free worker. No schema/role changes occur in serving. Existing Store data are not removed or reinterpreted.

Separate process/module readiness from provider execution availability. RuntimeReady means the original repository, current authorization/directory, starter and worker are wired/healthy. Available and CanSync remain false throughout preparation mode, including after worker start. The current workbench context may project RuntimeReady only through its original scoped Store/module read permission checks; this permits the real page to display a pending notice. Products/orders capabilities expose runtimeReady and providerConfigured as non-secret booleans. No page may render a successful empty result/0 statistics when provider configuration is pending. Lists/details/sync/ensure/command consumers reject unavailable before business writes or dispatch; capability reads retain live scope/read authorization. Direct worker Open has no Official port and also rejects unavailable. Full mode and worker-failure readiness remain strict.

## Persistence, initialization and data retention

Reuse the existing owner-only InstallObservations transaction and its guard requiring an empty workbench_stores table and absent observation schema. Do not weaken these guards, migrate old observation rows or add a retained-nonempty-Store exception. If the named trial contains Store facts, installation stops for a concrete user decision instead of deleting them.

Add an explicit observations-only option to the current schema-owner CLI. It verifies the existing current Store schema, then invokes only InstallObservations and its existing minimal grants in one transaction. It does not run Store base migrations, Cockpit installation or general grants. The mutually exclusive observations-grants-only mode explicitly restores original GrantRuntime privileges on the existing owner-controlled four-table schema, without installation, fact changes or base grants. The observations-verify mode uses exactly store_center_runtime and a read-only connection to verify existing observation table presence/count and original minimum privileges, both before manifest publication and when checking a completed initializer. Repetition never automatically restores deliberately revoked grants. Serving still uses store_center_runtime, fixed public search_path and schema-qualified observation queries. Owner credentials remain installer-only.

A final optional Compose initializer consumes the retained acquisition-plans manifest read-only, first proves its Store database is the same fixed 127.0.0.1:5433 / store_center target used by the owner, then verifies the empty/absent observation setup, installs the existing four tables using the original Store owner, and emits a separate private serving manifest with preparationOnly enabled. It must preserve the original manifest and all model/payment/channel settings. The runtime configuration volume and completed marker are separate; retained initialization verifies original input/configuration identity and reuses the completed output without invoking schema installation again. An interrupted marker or changed base manifest fails for operator inspection. No previous initializer is rerun during the retained update. Existing source bind paths, serving roles, private support token/network and named volumes remain intact.

Apply the new serving manifest only after installation succeeds. Only backend/UI serving containers need replacement; model/payment/Store official configuration remain closed. Back up the named local Store database privately before an authorized installation, compare original Store/Product facts and permissions/Chat/support reads after restart, and preserve rollback manifest/images. Stop/restart never destroys volumes.

Rollback to the old disabled-observations manifest requires stopping the observation worker and revoking only its newly installed table privileges and schema USAGE from store_center_runtime in one owner transaction; leave all four tables and their data intact. The unchanged strict disabled preflight must then pass before starting the old images/manifest. A later explicit re-enable reuses the original GrantRuntime under owner authority, not InstallObservations again. Verify both enabled and disabled permission combinations in the temporary PostgreSQL test. Saved images alone are insufficient rollback evidence.

## Authorization, failures and unchanged boundaries

No new role, grant catalog, identity, tenant or Store authorization rules. Module flags cannot authorize a member or a store. Capabilities/routes continue exact current Organization/Actor/Member and original read/sync permissions; full execution continues active service, verified official connection, application type/version and private credential hash checks on every call.

No new business receipt, recovery transition, retry rule, accounting or external side effect. Preparation mode makes no provider request and writes no sync command. Existing full-mode stable keys/checkpoints/monotonic heads/UNKNOWN and worker lifecycle remain unchanged. Missing schema, privilege drift, authorization service, Temporal or worker startup fails closed. New platform configuration later requires an explicit validated full-mode manifest; it does not modify existing conversations, Store facts or completed observations.

Legacy decision: N/A for this supplement; reuse only current owners. The original RETIRE decision for legacy sheinsync remains in force.

## Necessary verification and handoff

TDD for explicit preparation admission/no configured official apps/full-mode strictness; process worker start/failure/cleanup; capabilities runtime-ready/provider-pending and no business writes/dispatch; rejection of retained observation facts; CLI observations-only isolation; real temporary PostgreSQL transactional installation/minimal grants/preserved base facts; retained initializer identity and repeat behavior. Reuse current tests and bounded temporary resources, no new runner or acceptance platform.

Final delivery includes a normal retained URL, truthful worker/readiness evidence, current pending configuration pages, saved data/volume evidence, exact candidate PR/CI and known limits. Real SHEIN/model/payment and final user business acceptance remain NOT_RUN. Actual retained-instance schema execution must have applicable user authority; architecture readiness alone is not that authority.

## Review

Bounded independent review completed by the existing read-only reviewer: IMPLEMENTATION_READY, no design BLOCKER. R1 IMPLEMENTATION_TEST covers runtime/provider capability separation, scoped context, strict UI and closed HTTP/worker paths. R2 IMPLEMENTATION_TEST covers owner-only initialization, rollback/grants, completion marker and input identity. These are implementation/authorized-trial checks; no additional product scope or Writer. Design readiness does not authorize retained database installation.
