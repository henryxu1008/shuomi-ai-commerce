# SHEIN Marketplace

- `offlinepackage` owns local draft value structs, bounded strict JSON codec, deep copies and offline checks; it has no provider, App, DB, clock or file I/O dependencies.
- `draft.Builder` assembles an exact Catalog snapshot and approved assets. Source variants, images and attributes use private deterministic helpers. Attributes are ordered by raw name before existing cleanup.
- `publishing` owns Price and current pricing, readiness, image, site and weight rules.
- `workspace` owns pure review and taxonomy rules; `validator` owns current deterministic diagnostics.

Saved payloads, bindings and hashes are not rebuilt from current configuration. Record persistence, permission and transactions retain their current owners. Local drafts and offline diagnostics do not authorize platform submission.

See [current boundaries](../../../docs/architecture/project-boundaries.md) and the [validator contract](../validator/README.md).
