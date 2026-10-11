# TEMU Integration

Owns TEMU external-system adapters that are not listing-target business rules.

`OfficialClient` uses the fixed official routers, bounded standard HTTP transport,
exact UTF-8 JSON signing and private credential variant admitted by #31.
Identity verification remains separate from publishing or merchant eligibility.

The named category reads implement E33/E34. Supplier categories remain on CN:
current CN token expiry and the exact category scope are queried on CN before
metadata is read. PA permissions cannot authorize an unmigrated CN interface.
Local categories use the original US/EU/GLOBAL application route and only a
verified LOCAL mall. Both reads share one total deadline with their scope query.
The two response types retain their different flags; a missing list is an error,
an explicit empty list is an observation, and missing availability is unknown.

The named template reads implement the supplier CN attribute template and the
local category field rules. Parent conditions, explicit zero precision and
unknown enum values are retained. The local protocol's `filedName` spelling is
intentional; warnings and second-hand requirements are evidence for completion,
not automatic approval. No channel override or historic template fallback is
exposed.

Local attribute reads additionally bind E36's exact category, language and
selected costTemplateId; the original regional token and exact API scope remain
required. Category/language mismatch or missing/duplicate attribute identity is
rejected. Parent-value groups, explicit zero precision and transnational
requirements are preserved for completion; field-rule permissions cannot grant
this separate metadata API.

Current Store access is `OfficialProductAccess.AuthorizeTemuRules`, a server-only
five-second handle minted after current native Store/member/grant/service and
official identity checks. Every read rechecks the exact private material and
connection. This read-only capability has no submission or image mutation.
The shared TargetRecord/editor and publishing consumers remain future work in
the same #31 batch; there is no public arbitrary-API proxy or acceptance claim.
