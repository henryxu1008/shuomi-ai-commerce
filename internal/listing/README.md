# Listing Domain

- `materials` and `completiondraft` own current listing material and draft contracts.
- `preparation` transfers selected Collection items and coordinates explicit preparation operations.
- `record` owns immutable local listing records and their scoped read contracts.
- `submission` owns current execution identity, intents, receipts and provider effect boundaries; `readiness` and `sourcefacts` hold pure rules.

Product facts and approved assets belong to `internal/product/catalog` and `internal/product/asset`. Canonical inspection binds the exact ProductKey / CatalogVersion. Current Store observations consume official authorization through their existing owner. Local readiness and saved drafts do not grant submission permission.

See [current boundaries](../../docs/architecture/project-boundaries.md).
