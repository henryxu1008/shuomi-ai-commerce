# Refactoring Documentation

This directory contains current boundary references and historical refactoring records for Shuomi AI Commerce. Start with the [documentation index](../README.md); this directory does not maintain a second execution queue.

## Document authority

Apply current product decisions and explicit supersession before reading historical plans. A newer file date alone does not replace an approved contract.

The current product baseline is [PD-GREENFIELD-NO-LEGACY-MIGRATION-2026-09-08](../product/greenfield-no-legacy-migration.md): design for a fresh installation with empty business data and current owners only.

### Current authorities and navigation

- [Product / Figma Authority](../product/final-ui-ia-authority.md) - current product projection and interaction authority.
- [Architecture index](../architecture/README.md) and [Project Boundaries](../architecture/project-boundaries.md) - applicable contracts, ownership and dependency rules.
- [Repository Structure](../development/repository-structure.md#current-entrypoint-map) - source entrypoints and maintained commands; read its exact retirement updates before older catalogues.
- [Legacy Hard-Cut Policy](./legacy-hard-cut-policy.md) and [Legacy Register](./legacy-register.md) - approved EXTRACT / RETIRE decisions and their bounded supersession.
- [Roadmap #137](https://github.com/henryxu1008/shuomi-ai-commerce/issues/137) and the current execution Issue / PR - current scope, dependencies, owner and progress.
- [../product/product-sourcing-mvp-plan.md](../product/product-sourcing-mvp-plan.md) - Product Sourcing implementation and closeout plan.
- [../product/product-sourcing-handoff.md](../product/product-sourcing-handoff.md) - Product Sourcing ownership boundaries and handoff expectations.
- [../architecture/legacy-listing-preview-download-closure-retirement-boundary.md](../architecture/legacy-listing-preview-download-closure-retirement-boundary.md) - confirmed source retirement of old Task preview/platform and listing downloader (#301 / #891); current Product/Catalog/Asset and real Listing owners remain authoritative.

### Long-term architecture authority

- [project-wide-refactoring-plan.md](./project-wide-refactoring-plan.md) - historical reference, not a current implementation authority for greenfield work.
- [../architecture/project-target-architecture.md](../architecture/project-target-architecture.md) - approved target architecture shape.
- [project-migration-roadmap.md](./project-migration-roadmap.md) - historical reference, not a current implementation authority; it is superseded for greenfield work by the current product baseline and Hard-Cut policy.
- [module-target-mapping.md](./module-target-mapping.md) - working map from current package areas to target domains.

### Strategic decision records

- [decisions/2026-06-26-next-growth-sequence.md](./decisions/2026-06-26-next-growth-sequence.md) - dated growth-sequence decision; read with later product decisions and the current execution Issue, not as a live priority list.

<a id="historical-records"></a>

### Historical execution references

- [current-refactoring-status.md](./current-refactoring-status.md) - baseline-bound maturity and gate record, last reviewed 2026-09-05; later bounded notes do not refresh every historical Now / Next / Later statement.
- [next-phase-plan.md](./next-phase-plan.md) - historical reference, not a current execution authority; use current product decisions and the execution Issue instead.
- [listingkit-boundary-checkpoint.md](./listingkit-boundary-checkpoint.md) - historical reference, not a current implementation authority for greenfield work.
- [project-wide-execution-plan.md](./project-wide-execution-plan.md) - earlier broad execution plan; keep for historical PR slices, not the active sprint queue.
- [project-migration-roadmap.md](./project-migration-roadmap.md) - historical reference, not a current implementation authority; do not use its legacy compatibility directions for a greenfield task.
- [listingkit-refactoring-progress-2026-06-24.md](./listingkit-refactoring-progress-2026-06-24.md) - dated progress snapshot and backend Control Plane closeout evidence. Treat it as evidence, not the current execution queue.
- Older local plans provide background only within their recorded baseline; current approved contracts and explicit supersession determine what remains applicable.

For submission and Temporal-related work, use current Listing / Marketplace contracts through the architecture index. The superseded ListingKit checkpoint cannot restore old package ownership, compatibility or migration requirements.

## Status vocabulary

Use these terms consistently across refactoring and product documents:

- **Implemented**: the code path exists on the referenced baseline.
- **Repository-validated**: the exact baseline has recorded automated test/build results.
- **Production-validated**: a real environment or real API run is recorded in a dated validation note.
- **Deferred**: code or runtime assets may exist, but the capability is not an active product-expansion commitment.

Do not use official command existence, retained historical code, or an implemented adapter as proof of full product maturity.

## Boundary rules and enforcement

Use these files before starting broad package moves:

- [../architecture/project-boundaries.md](../architecture/project-boundaries.md) - allowed ownership, forbidden import directions, placement rules, and review checklist.
- [dependency-baseline.md](./dependency-baseline.md) - worksheet for capturing the current dependency and package-shape baseline before broad moves.
- [../../scripts/analyze-project-deps.ps1](../../scripts/analyze-project-deps.ps1) - advisory dependency analysis script for package file counts, largest files, ListingKit import pressure, and likely boundary violations.

Recommended first command from the repository root when fresh dependency evidence is needed:

```powershell
New-Item -ItemType Directory -Force .local/refactoring | Out-Null
./scripts/analyze-project-deps.ps1 6>&1 | Tee-Object -FilePath .local/refactoring/dependency-baseline-output.txt
```

To make the script fail when it detects advisory boundary violations:

```powershell
./scripts/analyze-project-deps.ps1 -FailOnViolation
```

The script is advisory unless a current PR explicitly promotes a specific check to CI. Generated dependency/package outputs should remain under `.local/` unless a dated validation note summarizes the relevant result.

## Current active direction

The following are retained structural principles, not a dated product delivery queue. Current priority belongs to #137 and the execution Issue:

1. Keep the project as a modular monolith first; do not split microservices before package boundaries are stable.
2. Keep source implementation, repository checks, instance availability and user acceptance distinct; old production gates cannot assert current delivery status.
3. Use the current product-source flow and its approved owner; do not route new work through a legacy ListingKit handoff.
4. Let current product decisions and execution Issues determine source/platform expansion; do not dispatch from old Now / Next / Later lists.
5. Extract current-owner behavior and RETIRE legacy ListingKit ownership; keep task flow, preview/export aggregation, revision/history, persistence ordering, and API shell responsibilities only with their current owners.
6. Keep marketplace-specific rules in marketplace-specific packages.
7. Keep product facts, product-source identity, source normalization, and reusable visual assets outside root ListingKit.
8. Hide infrastructure and external clients behind small interfaces.
9. Prefer small, testable current-owner changes over broad rewrites; any schema evolution is normal new-system schema evolution, never legacy-data migration.
10. For submission refactoring, prefer `internal/listing/submission` for generic mechanics and approved marketplace publishing owners for platform rules. Historical `internal/listingkit` references describe a shrinking extraction/RETIRE surface, not a package to recreate.

## Current refactoring documents

This is a reading catalogue. Historical rows retain their recorded baseline and do not become current execution authority by appearing here.

- [current-refactoring-status.md](./current-refactoring-status.md) - historical baseline snapshot and bounded later notes; not a live maturity dashboard or execution queue.
- [next-phase-plan.md](./next-phase-plan.md) - historical reference, not a current execution authority.
- [listingkit-boundary-checkpoint.md](./listingkit-boundary-checkpoint.md) - historical reference, not a current implementation authority for greenfield work.
- [project-wide-refactoring-plan.md](./project-wide-refactoring-plan.md) - historical reference, not a current implementation authority for greenfield work.
- [project-wide-execution-plan.md](./project-wide-execution-plan.md) - earlier broad execution plan; historical/reference when it conflicts with current checkpoint docs.
- [project-migration-roadmap.md](./project-migration-roadmap.md) - historical reference, not a current implementation authority for greenfield work.
- [module-target-mapping.md](./module-target-mapping.md) - working map from current package areas to target domains.
- [product-sourcing-inventory.md](./product-sourcing-inventory.md) - historical Amazon / 1688 source-flow inventory; new work uses current sourcing contracts and the greenfield baseline.
- [listing-preview-migration-map.md](./listing-preview-migration-map.md) - historical, superseded preview migration inventory; use the confirmed source-retirement boundary above, not its old owner/compatibility proposals.
- [dependency-baseline.md](./dependency-baseline.md) - baseline worksheet for dependency scans and legacy exception tracking.
- [preview-subpackage-feasibility.md](./preview-subpackage-feasibility.md) - historical, superseded feasibility note; its keep-in-root and future extraction proposals are not current implementation authority.
- [submission-inventory.md](./submission-inventory.md) - historical submission consolidation inventory; current ownership comes from approved Listing / Marketplace contracts, not the superseded checkpoint.
- [service-slimming-checkpoint.md](./service-slimming-checkpoint.md) - Phase 4 first checkpoint for root service object file-group slimming.
- [checkpoint-validation-2026-06-17.md](./checkpoint-validation-2026-06-17.md) - focused next-phase validation run covering ListingKit, HTTPAPI, boundary tests, dependency scan, and blocker classification.
- [listingkit-refactoring-plan.md](./listingkit-refactoring-plan.md) - detailed ListingKit root-directory reduction plan.
- [studio-migration-plan.md](./studio-migration-plan.md) - Studio migration planning.

## Refactoring principles

1. **Hard-Cut**: apply `EXTRACT | RETIRE`; do not create legacy compatibility facades, shims, bridges, wrappers, fallbacks, or dual paths.
2. **Small steps**: each refactoring task should be independently reviewable and testable.
3. **Test first**: capture or run relevant tests before and after each current-owner change; normal new-system schema evolution must not be mistaken for legacy migration.
4. **Incremental change**: avoid one-shot rewrites and package renames.
5. **Boundary ownership**: new business rules should be placed in the package that owns the business concept.
6. **No silent authority drift**: record current scope and progress in the execution Issue / PR; product or architecture decisions belong to their existing authority. Do not turn an old status snapshot into another live queue.
7. **Evidence discipline**: do not claim a baseline is green unless the exact workflow, command output, or validation note is visible.

## Risk control

- Establish a test and dependency baseline before large moves.
- Keep behavior-preserving moves separate from feature changes.
- Record known import-boundary exceptions instead of normalizing accidental dependencies.
- Keep rollback simple by limiting each PR to one bounded capability.
- Require code review for boundary changes, package moves, production-sensitive SHEIN behavior, pricing logic, submission behavior, or new cross-module dependencies.
- Do not continue helper shaving, file splitting, or package renaming unless it reduces real ownership or improves dependency direction.
- Do not use Product Sourcing implementation progress as justification to start several new source integrations at once.
