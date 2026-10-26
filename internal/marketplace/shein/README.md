# SHEIN Marketplace

Current responsibilities:

- `offlinepackage`: persisted local draft value types, bounded strict JSON codec,
  owned deep copies and deterministic offline checks. No provider, persistence,
  App, implicit clock or file I/O dependencies.
- `draft.Builder`: fixed local assembly from the exact Catalog snapshot and
  approved assets. Its helpers are private; it exposes no resolver/AI/config ports.
- `publishing`: existing platform rules. `workspace` and `validator` retain their
  current review/readiness/diagnostic responsibilities.

`internal/publishing/shein` is retired without aliases or compatibility wrappers.
Current JSON fields/order, hashes and binding/rule versions are unchanged. Record
persistence, permissions, Store observation and independent SHEIN login/Store
synchronization remain with their current owners. Local drafts and offline
diagnostics do not authorize a platform submission.

Boundary and original-check conversions: [#960 design](../../../docs/architecture/legacy-shein-draft-assembly-closure-retirement-boundary.md).
