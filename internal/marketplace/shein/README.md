# SHEIN Marketplace

Current responsibilities:

- `offlinepackage`: local draft value structs (including Variant, ImageSet,
  bundles, attributes and sites), bounded strict JSON codec, owned deep copies
  and offline checks. No provider, App, DB, clock or file I/O dependencies.
- `draft.Builder`: fixed assembly from the exact Catalog snapshot and approved
  assets. Source variants, images and attributes use private helpers. Attributes
  are ordered by raw name; existing cleanup still runs afterward. No resolver,
  AI or config port is introduced.
- `publishing`: Price and current pricing/readiness/image/site/weight rules.
  `workspace` and `validator` retain review and diagnostic responsibilities.

The complete old `internal/publishing` root and `internal/pkg/jsonx` are retired
without aliases, wrappers or a replacement common package. Field shapes, strict
JSON, hash/binding algorithms and versions remain unchanged. Future newly built
multi-attribute drafts have a deterministic order; their hashes may differ from
an old random ordering. Existing saved payloads are never rebuilt or rewritten.
Record persistence, permission and transaction owners, Store observation, and
independent SHEIN login/Store synchronization remain unchanged. Local drafts and
offline diagnostics do not authorize a platform submission.

Boundary and original-check conversions: [#961 design](../../../docs/architecture/legacy-publishing-support-policy-closure-retirement-boundary.md).
