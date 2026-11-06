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

`Price` is a real value struct here; the seven source draft value types belong
to `offlinepackage`, which may refer to Price. This package never imports that
data owner or draft/Record/App/provider/persistence wiring. Cross-domain execution stays with the Listing / Marketplace execution owner.

[当前系统边界](../../../../docs/architecture/project-boundaries.md).
