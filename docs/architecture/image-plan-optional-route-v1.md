# Single-product image plan optional route V1

Status: DESIGNING / NOT_READY. Execution Issue #840. Source baseline:
`773389eefe2ded6027a72c9351e9b2aa50bac96b`.
Design Basis: Independent Architecture. No production edits before the original
independent Reviewer explicitly admits IMPLEMENTATION_READY.

## Product outcome and authority

PD-IMAGE-PLAN-VIP-2026-10-21: the user approved a model selector for one product
image plan and maintainer publication as an option without changing the enterprise
default. This supersedes only the earlier maintenance-only/per-plan-out-of-scope
decision in grsai-vip-image-binding-v1. Historical evidence is retained.

Figma authority (existing tokens, Noto Sans SC and linked controls):
- Maintenance [dark 8158:2291](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=8158-2291),
  [light 8158:2317](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=8158-2317).
- Product plan [dark 8158:2343](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=8158-2343),
  [light 8158:2388](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=8158-2388).
Original design nodes remain. The 8158:2433 composition has editable text/control
layers rather than a whole-screen raster and was visually inspected before code.

Must: default to Bailian; allow explicit enterprise-published VIP for a whole new
product image plan; show model, medium, original 1024x1024 and points before
confirmation; freeze route/rate versions alongside original images, template,
confirmed text and exact Skill. Use existing permissions, budget admission,
generation, review and product-image save owners. Reject original size mismatch;
never resize. Changing model requires a new plan/quote and explicit confirmation.

Out of scope: slot-level mixed models; artwork model selection; default switching;
new providers/keys; secret copying; automatic fallback/retry; text-model calls;
new price policy or free application execution; increases to member limits;
OCR/rendering; resizing/cropping/re-encoding; rewriting old facts; migration or
backfill; new runner/platform; runtime installation/publication/deployment or real
provider calls. One Writer, one main PR, original Reviewer at design/final checks.

Threat model inherits approved route pricing, image-agent, VIP binding and exact
Skill contracts. Protect enterprise/identity, maintainer-only publication, qualified
credential and tariff, selected revision consistency, original idempotency and
UNKNOWN. No newly accepted risk or unrelated new security requirement.

## Owners and minimal persistence

`internal/aicapability/routeconfig` remains the only route/rate authority.
Its immutable revision payload gains `optional: true` ONLY for a newly published
image VIP option. Omitted false retains the existing canonical payload/receipt
semantics. The input gains the same optional flag (omitempty), included in the
request fingerprint, so a key cannot be reused between default/optional publication.
Text and other bindings cannot use optional mode in this increment.

Add one current-option pointer relation under the same persistence owner:
`ai_route_config_options(organization_id, capability, binding_id, revision_id)`.
PK is (organization_id, capability, binding_id); FK scopes the immutable revision
by organization/capability. It is not a second route or price fact: the payload,
tariff, audit, credential reference and receipt remain in existing revisions.
The option reader verifies payload.optional, binding ID and exact qualified
credential against the current enterprise admission catalog; arbitrary revisions
and historical/unpublished bindings are not selectable.

Default publication continues to use `ai_route_config_current` and its existing
CAS. Optional publication performs the same revision+pointer atomic transaction,
but CAS/parent use the selected binding's option head (NONE for initial publish).
It never updates/inserts the enterprise default pointer. A replacement makes the
old option revision unavailable for NEW admission/dispatch. Existing credential
disable/removal or qualification drift also makes the option unavailable. No
new withdraw UI/state machine is necessary. Current() must reject optional payloads.
Rollback, if used through existing maintenance, cannot cross publication modes.

The explicit installer creates the relation and grants only SELECT to the existing
route reader/image worker and INSERT,UPDATE to the existing configuration writer.
Immutable revision INSERT and metadata SELECT remain as today. No secret grant,
new role or image API direct DB grant. Update formal permission inventory, strict
reader/writer verification and image-worker optional relation admission together;
wrong relation kinds/RLS/owner/extra writes fail closed. Startup inspects only.
No retained instance DDL/grant/data operation is part of this development authority.

Imageagent owns new prepare `Route *ImageRouteSelection` and the same optional
field on ImageSetPlan. The selection contains only binding ID and exact revision
ID, no caller organization, endpoint, credential, price or quality. nil means the
existing enterprise-default behavior; it is omitted from old serialization. A
non-nil selection is validated and copied, never aliased. Existing recipe quotes
already freeze provider/model/protocol/quality/configurationVersion/priceVersion/
points/credential reference. No new GenerationIntent columns or ledger.

## Contract → implementation → injection → consumer

1. Original maintainer routeconfig Validate/Publish accepts optional mode only for
   exact qualified VIP. Current default and optional current are read separately.
   Existing preview/receipt/history expose publication mode and exact published
   option revision. The maintenance editor labels its explicit confirm action
   “发布为可选，不设为默认”; rates remain maintainer-specified, positive and within
   existing admission ceilings. No hardcoded inheritance of Bailian's price or
   the standalone comparison's zero platform points.
2. Add a bounded read-only options action to the existing source/product image
   API and BFF, under current source/image permissions and tenant identity. Reuse
   original source ownership checks. Return default and qualified option labels,
   exact selection IDs, model/quality/resolution/points. Do not expose endpoint,
   secret, credential reference or raw maintenance revisions to ordinary members.
   Enumerate bounded admitted candidates, not all historical revisions.
3. Extend the existing ImageSetQuoteReader contract with the explicit optional
   selection. Default/artwork consumers pass nil. The existing injected route
   service resolves Current for nil, or exact current Optional for non-nil, then
   validates the existing credential/tariff. There is no browser override or
   temporary global publication. Unmanaged installations reject explicit options.
4. Prepare input validation/digest and immutable plan carry Route; all recipes
   in the plan have the same selected quote. The quote digest includes Route only
   when present. Existing nil-route fingerprints and historical validators stay
   byte-identical. Confirm obtains the quote for this frozen Route before NEW
   admission; original durable admission receipt recovery retains its original
   path and deadline and does not choose a new route.
5. Extend generation provider preparation/revalidation callbacks to take the
   existing SlotExecutionInput, so factory resolution consumes ImageSet.Route
   rather than global current. Artwork/default input has no explicit selection.
   QuoteSlot, GenerateQuotedSlot and final pre-dispatch check all use the same
   selected route. Quote/configurationVersion must match the selected revision;
   full metadata/price/credential compare remains. A default change alone does
   not invalidate an unrelated explicit VIP option; an optional head/credential
   drift rejects before POST. No untrusted context-key routing shortcut.
6. Existing slot fingerprint/input digest includes the frozen plan Route, hence
   GenerationIntent's existing input digest and exact configurationVersion bind
   the selection without a second intent fact. Existing authorizer, source guard,
   member limit, reserve/bind, final authorization/revalidation, dispatch claim,
   observer-before-download and reconciliation remain in original order/owner.
7. The existing GRSAI adapter sends gpt-image-2-vip, medium, aspectRatio=1024x1024,
   replyType=json once. Original raw output must be exactly 1024x1024. A paid
   generation with invalid media remains paid/invalid media, never recast as a
   no-effect rejection or silently transformed. Original response/diagnostics
   and locator remain available. Human selection/save retains separate consent.

## Identity, UI, recovery and failure behavior

Maintainer read/write permissions stay with platform maintenance/admin write;
ordinary image users never gain those APIs. Member choice/prepare/confirm/dispatch
use existing image read/write/source authority, membership and live organization
checks. Current organization is derived server-side; foreign binding/revision,
unpublished option, stale replacement and credential drift are rejected. No key
or endpoint in member UI. Cross-enterprise/identity changes reset UI choices and
pending views by existing scope ownership; no cached choice leaks across scopes.

Initial new-plan selector: “跟随企业默认（百炼）”. Selecting VIP changes only
the unsent draft. Once prepared, confirmation shows the immutable model/quality/
dimensions/points; editing selection prepares a new plan, not mutation of the
old one. Restore uses the original raw request and selected exact version; it
does not substitute the latest VIP version. Missing/stale options produce an
explicit reprepare message, not silent default. Optional lookup failure does not
invent a successful empty list or enable an unverified choice.

Regeneration still requires closed parent effects and original source/template/
exact Skill. Its NEW draft initially carries the parent's explicit selection;
an unavailable old option requires explicit new selection/requote. The caller can
select another currently published route for the new plan only; parent/request
bytes and audit remain unchanged. Same key/different selection conflicts.

Optional publication ACK loss: original key/payload receipt resolves it; no second
publication and no default switch. Initial concurrent publication has one winner;
the loser recovers same-key receipt or conflicts. Replacements CAS against exact
option head. Atomic rollback prevents orphan publication on pointer failure.

Plan/confirmation ACK loss: original request/admission receipt; no model choice
or new dispatch permission inferred from failure. True UNKNOWN remains locked,
its reservation unchanged, and cannot be bypassed by changing model/new request.
Default/optional drift before dispatch uses existing no-effect closure and release
proof, not a fabricated UNKNOWN. After dispatch, original success/unknown recovery
and reserved funds retain their existing owner. Result recovery reads the saved
locator independent of new-route selection and never resubmits.

Preserve original request/prompt/source/response bounds (up to original plan 2MiB,
source references <=8 and 16MiB bundle), API deadlines, source SSRF/TLS and single
provider request/no-submit-replay limits. No polling/retry budget is enlarged.
Old UNKNOWN, including the retained 5-point case, is neither settled nor retried
by this change. Existing tasks and exact Skill/translation records are not edited.

Legacy decision: N/A. Extend current routeconfig/imageagent owners, no retired
abstractions, fallback, migration/backfill, dual-write or second source of truth.

## Validation and rollout

TDD before production edits after admission: route domain/store optional publish,
separate default pointer, immutable qualified rates, CAS/replay/conflicting mode,
cross-enterprise and publication-mode rollback rejection. Reuse existing isolated
PostgreSQL tests for real transactions, exact role grants and startup guards.
Image tests cover selected VIP with Bailian default, same metadata at quote/new
admission/final dispatch, wrong/stale selection zero POST, independent default
change, frozen template/text/exact Skill, old nil-route serialization/recovery,
regeneration, true UNKNOWN and original budget/settlement behavior. Existing GRSAI
network-none fixtures prove medium/1024 request and strict wrong-size refusal.
UI/BFF tests cover maintenance optional preview/receipt, selection/price/confirm,
restored raw request, scope reset, ordinary-member denial and light/dark styling.

One main PR with original Writer and original independent final Reviewer, scoped
self-checks and existing required CI on stable candidate. No new verification
platform or default global/security review. Later final main synchronization checks
only actual combination risks. Implementation/CI do not sign user acceptance.

Runtime rollout is separate: expressly install the new relation/grants, then update
necessary UI/backend, explicitly publish VIP option and inspect that Bailian current
is unchanged. Do not switch a new binary against absent required managed schema,
relax startup gates, reset volumes, overwrite backup or restore stale DB facts.
Source delivery alone does not activate an option. A new real-call budget is
required before any provider request; UNKNOWN/refusal/exception stops immediately.
