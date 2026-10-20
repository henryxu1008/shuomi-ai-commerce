# GRSAI VIP 图片可选绑定 V1

Status: IMPLEMENTATION_READY. Execution: #840.
Original independent Reviewer admitted design HEAD
2538d3be3775b0a98cda3661b7b23b6c744388ab with no BLOCKER before production edits.
Implementation must verify fixed medium through edit/artwork, unchanged old
serialization and recovery, strict original dimensions with paid facts retained,
and existing credential qualification/revalidation without publishing a binding.
Design Basis: Independent Architecture, limited to an additional admitted image model.
Source baseline: main f8f7853677765337986cd2673422c59fae65f94e.

## Product outcome and authority

User decision PD-GRSAI-VIP-BINDING-2026-10-20: admit `gpt-image-2-vip`
through the existing maintainer-only Model and Pricing editor. The user explicitly
selected maintenance bindings only, with no per-plan route selector. VIP uses
`medium`; original media must be exactly 1024x1024 without transformation.
Existing Bailian defaults, credentials, old requests, exact Skill versions,
UNKNOWN reservations, audit records and data remain untouched.

Product authority: Issue #840 and the user's entry-point clarification.
Figma file tg48P46SSXl6TBy9lZwg63, page 31:463: existing image editor
7660:2143 / 7660:2276 remains; editable VIP candidate successors are
[dark 8110:2250](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=8110-2250)
and [light 8110:2274](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=8110-2274).
Reuse linked controls, tokens and styles.
The current binding remains visible while editing a separate unpublished VIP
candidate; quality and dimensions are read-only constraints, not adjustable fields.
An operator must use the existing validate/preview/publish flow to change current
configuration. This development batch does not publish or install a binding.

Reuse approved model-route-pricing-management-v1, product-image-agent-v1,
bailian-product-image-wiring-v1 and official-skills-and-image-method-v1.
The completed independent VIP comparison returned an original 1024x1024 PNG
at 2000 gateway credits with readable approved text. This is one sample, not
proof of availability, sustained reliability, official upstream model identity,
application integration or product acceptance.

## Scope and boundaries

Must: qualify an explicitly provisioned VIP binding for the original image
capability; show fixed medium/1024 constraints; preserve validate/publish
authorization and revision semantics; consume the selected exact model and quality
in quote, intent and single dispatch; reject wrong quality or original dimensions;
preserve ordinary image and shared artwork consumers and original recovery.

Out of scope: per-plan selection, new configuration units or tables, new protocol,
new secret owner, key copying/provisioning, production configuration changes,
default publication, provider probes in normal UI/startup, automatic failover,
retry, OCR/rendering, resizing/cropping/re-encoding, quota increases, migration,
historical rewrites, runtime deployment and real provider calls.

Threat model and accepted risks inherit #830. A member must not gain maintenance
access; a descriptor cannot admit a foreign, disabled or changed credential.
Model, provider, protocol, endpoint, credential version, price and exact revision
must still match at final dispatch. No new accepted risk or weaker check.

## Owner and complete call chain

1. The original private deployment admission descriptor lists a separate client
   name, exact VIP model, GRSAI endpoint, credential version and maximum points.
   Existing organization credential provisioning remains the sole key owner.
   No descriptor/key is created by startup or by this development batch.
2. Existing routeconfig Catalog qualifies that descriptor against the exact
   current enterprise credential row. Bindings API and maintainer editor expose
   only qualified choices; an absent binding remains unavailable.
3. Existing service Validate/Publish and immutable revision/current tables freeze
   the selected binding and point tariff. Adding support does not publish a new
   current revision or alter Bailian. A later expressly authorized publication
   has the original shared capability scope, including artwork.
4. Current image quote reader and generationProviderFactory resolve the original
   published revision and organization-only credential, with no global-key
   fallback. They admit VIP alongside the existing approved tuples.
5. Existing ImageGenerationQuote and GenerationIntent already store Quality.
   Use model-specific validation: GRSAI VIP + sync protocol requires medium;
   GRSAI base 2.5 and approved Bailian models require their original auto.
   Invalid tuple/quality is rejected. No new persisted fields or schema version.
   Do not add fields to metadata whose serialization would change old quote
   fingerprints; derive fixed quality from its existing exact tuple.
6. Existing source-only adapter and shared artwork consumer send VIP medium to
   the GRSAI synchronous client. The transport validates the same exact model
   and quality, sends aspectRatio=1024x1024 and replyType=json exactly once,
   disables submit replay and persists the original success observation before
   downloading. No endpoint/response-schema substitution.
7. Existing original-output validation, candidate materialization, human selection
   and product save owners remain. Source edit, artwork and original-result
   recovery all keep exact 1024x1024 rejection and original bytes/diagnostics.

Read-only UI constraint text derives from the admitted model; it is not a price,
permission, health or execution authority. Existing payloads/history do not gain
fields or get reserialized into a new fact. The API/browser does not accept an
arbitrary quality parameter. The same provider protocol and transport are reused;
no new SDK, route registry or model-selection framework.

## State, transactions, recovery and external effects

No new state machine, fact owner, table, transaction, database role or grant.
Original Publish transaction/CAS/idempotency, original plan and exact Skill
snapshots, original reservation/dispatch claim/settlement and result locator remain.
Same key/different input remains a conflict. Old base/Qwen auto inputs remain
valid with identical serialized values and fingerprints; VIP auto/high/empty
quote values must not be accepted as medium. Explicit SDK quality omissions may
use the fixed model policy before dispatch, but an explicit wrong value is rejected.

Current-route/credential/price drift rejects new dispatch. In-flight success still
records its original fact and price. UNKNOWN is not retried, switched to another
model or released by inference. Successful output with wrong size remains paid
generation plus unusable media, never a rewritten no-effect result. Original
result recovery uses the saved locator and does not require new model execution.

Preserve existing request/prompt/reference/response bounds, deadlines, SSRF/TLS,
single-output limit, observer-before-download and no-submit-replay behavior.
Maintenance validation and publication do not call the provider. No new estimate
of provider currency or conversion of gateway credits into platform point tariffs.

Legacy decision: N/A; current owners and immutable current contracts only, no
legacy wrapper, migration, fallback, dual-write or alternate recovery owner.

## Verification and rollout boundary

TDD first reproduces rejected VIP admission/quote/intent/wire and missing UI
constraints. Then add only exact VIP support. Cover catalog credential/tenant
qualification and no publication, managed quote/provider drift, domain quality
pairings, shared artwork, exact medium wire and source bytes, wrong-quality zero
dispatch, observer failure/UNKNOWN no retry, original 1254 rejection and normal
1024 bytes, old-model recovery and serialized facts, maintenance authorization,
preview/receipt/history and light/dark UI components. Reuse existing fixtures,
isolated PostgreSQL tests and relevant CI; do not build an acceptance platform.

Implementation and independent final review go in one main PR, original Writer
and Reviewer. No merge, deployment, Issue closure, real database mutation,
credential binding installation/publication or paid execution is authorized by
development. The previous one-call VIP budget is exhausted. Any later real trial
needs a fresh bounded budget, and stops on exception/UNKNOWN/refusal.
