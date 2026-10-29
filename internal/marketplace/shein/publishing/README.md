# SHEIN Publishing Rules

This existing owner holds the current platform rules used by local draft,
offlinepackage, validator and workspace paths:

- Price values and `PricingPolicy.Apply` (currency, shipping, fixed markup,
  commission, minimum and increment rounding; its original formulas remain).
- Action-specific final-review and image readiness, image presence and URL
  normalization, payload image/gallery/site-detail shaping.
- Sale-attribute and SKU payload validation, supplier-code selection,
  site/weight rules and the DEFAULT preferred-warehouse sentinel.
- The existing variant-image coverage metadata reader, including its blocked
  status, message keys and fallback text. It does not own a metadata writer.

Old cost-conversion, SDS/POD, source-dimension/size/text/attribute generation,
prevalidation, final-image editing/upload-cache and auto-warehouse selection
families are retired with their exclusive tests and helpers. The old
`CalculateCostPrice` formula is not replaced by `PricingPolicy.Apply`: only the
latter has current consumers and keeps its original formula. Earlier remote
record/confirmation/retry/workflow descriptions are no longer this package's
current responsibility.

`Price` is a real value struct here; the seven source draft value types belong
to `offlinepackage`, which may refer to Price. This package never imports that
data owner or draft/Record/App/provider/persistence wiring. No alias,
compatibility forwarder or replacement common layer is retained.

[#961 frozen deletion and preservation matrix](../../../../docs/architecture/legacy-publishing-support-policy-closure-retirement-boundary.md).
