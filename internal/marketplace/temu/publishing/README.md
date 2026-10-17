# TEMU Publishing

Owns TEMU publishing rules and payload shaping for TEMU as a listing target.

Product-name and description sanitization, brand-text removal, and final
submission formatting live here. Private value-only submission validations check
used specification coverage/nonzero VID and Multiple Sets packaging counts.
They do not mutate inputs, repair products, call platforms, or depend on legacy
Task/context/client/state ownership. There is no new submission entrypoint.

The exclusive Commit/save/submit/result orchestration, quota caller and publish
context members in `internal/temu/product` / `internal/temu/context` are retired
under [#749's confirmed boundary](../../../../docs/architecture/legacy-temu-listing-submit-retirement-boundary.md).
Platform APIs/DTOs and the remaining mixed-package rules retain their owners;
this extraction does not introduce a legacy wrapper or compatibility path.
