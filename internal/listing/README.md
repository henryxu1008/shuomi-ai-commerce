# Listing Domain

Current listing contracts keep their existing owners:

- `preparation` transfers selected Collection items into preparations; `materials`
  and `completiondraft` own the current material/draft contracts.
- `record` owns listing records and their read contracts; `task` retains only the
  authenticated, tenant/owner-scoped canonical-subject lookup contract.
- `submission` retains the current execution intents, receipts and platform
  effect contracts. `readiness` and `sourcefacts` retain pure readiness rules.

Product facts and approved assets stay in `internal/product/catalog` and
`internal/product/asset`; source/approval lineage is not owned by a Task preview.
The unused preview/platform assembly and export/revision/settings/workflow
placeholders are source-retired under the [confirmed #301 boundary](../../docs/architecture/legacy-listing-preview-download-closure-retirement-boundary.md).
Do not migrate the legacy ListingKit facade or create a replacement Task lifecycle.
Current capabilities, including independent SHEIN login and Store sync, remain.
