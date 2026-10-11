# TEMU Models

Owns TEMU-specific models used by TEMU marketplace business logic.

Current official supplier/local category and template DTOs preserve their own
field names and missing flags. Local attribute queries include the selected
shipping template because it changes cross-border requirements (E36).

`LocalGoodsV2` follows E17's outer JSON business parameters, exact string
amounts/package measurements and numeric identifiers. It includes no credential
or client-selected site/mall and invents no optional commercial or legal fact.
An absent inventory quantity remains absent instead of becoming zero. The
second-hand scope enum disagreement between E17/E29 remains unresolved; these
DTOs do not convert it or grant readiness. Creation IDs and warnings are separate
from approval/on-sale status. The current shared TargetRecord/Submission owners
will consume these explicit contracts; no new lifecycle owner is introduced.

`SupplierGoods` follows E13's one-product body, with E27's separate full-managed
supplier price and semi-managed per-site prices. Required numeric facts retain
presence; prices stay integer fen/cents, stock zero remains explicit, and package
weight/size stay integer mg/mm. It does not copy the local country currency or
create a product per bound site. Optional origin/brand/IP/legal facts are never
invented. The `len` documentation inconsistency is retained without reordering
the supplied dimensions. Creation identifiers are separate from on-sale status.
