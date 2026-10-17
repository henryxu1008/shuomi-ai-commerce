# TEMU Publishing

Owns TEMU publishing rules and payload shaping for TEMU as a listing target.

Product-name and description sanitization, brand-text removal, and final
submission formatting live here. Private value-only submission validations check
used specification coverage/nonzero VID and Multiple Sets packaging counts.
They do not mutate inputs, repair products, call platforms, or depend on legacy
Task/context/client/state ownership. There is no new submission entrypoint.

The private SPU structure value check reuses the existing specification snapshot
and preserves the seven name/SKC/SKU/specification empty-value checks in their
original first-error order. It does not trim values, filter SKU entries, validate
prices or stock, or resolve temporary specifications. The four unused SPU and
maximum-retail-price Task owners are source-retired under
[#768's confirmed boundary](../../../../docs/architecture/legacy-temu-spu-price-query-retirement-boundary.md).
PriceHandler/SKU, platform APIs/DTOs, AI/text/property/specification implementations
and mixed context retain their owners; no execution or trial prerequisite is added.

Private SKU value functions preserve stable main-then-variant code order,
empty-identifier skipping and deduplication through the existing skugen ASIN-only
strategy. Any reported duplicate failure is rejected while retaining diagnostic
values, including failures with empty fields. They do not mutate source values,
call an API or own state. The four unused TEMU preflight Task handlers are retired
under [#750's confirmed boundary](../../../../docs/architecture/legacy-temu-preflight-retirement-boundary.md);
shared Reader/Cache/Mapping, platform check APIs/DTOs and mixed context remain.

The exclusive Commit/save/submit/result orchestration, quota caller and publish
context members in `internal/temu/product` / `internal/temu/context` are retired
under [#749's confirmed boundary](../../../../docs/architecture/legacy-temu-listing-submit-retirement-boundary.md).
Platform APIs/DTOs and the remaining mixed-package rules retain their owners;
this extraction does not introduce a legacy wrapper or compatibility path.
