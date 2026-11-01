# Listing Domain

Current listing contracts keep their existing owners:

- `preparation` transfers selected Collection items into preparations;
  `materials` and `completiondraft` own current material and draft contracts.
- `record` owns immutable local listing records and their scoped read contracts.
- `submission` owns the current execution kernel, intents, receipts and provider
  effect boundaries. `readiness` and `sourcefacts` retain pure rules.

Product facts and approved assets remain in `internal/product/catalog` and
`internal/product/asset`. Canonical inspection reads the exact
ProductKey/CatalogVersion; the old `task` Reader/lifecycle was retired under
[#1025](../../docs/architecture/legacy-task-openai-convenience-closure-retirement-boundary.md).
The old preview/platform/downloader owners were retired under
[#891](../../docs/architecture/legacy-listing-preview-download-closure-retirement-boundary.md).

Independent SHEIN browser login and identity writeback were retired under
[#985](../../docs/architecture/legacy-shein-login-identity-sync-closure-retirement-boundary.md).
Current Store observations consume official authorization under their existing
owner; they do not recreate the old Task or login runtime.
