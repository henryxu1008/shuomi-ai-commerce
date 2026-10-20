# Image target language, localization and confirmed copy

Design Basis: **Independent Architecture / DESIGN_REVIEW_PENDING**.
Execution: [#840](https://github.com/qq550723504/task-processor/issues/840),
parent [#774](https://github.com/qq550723504/task-processor/issues/774).

## Product outcome and authority

On 2026-10-20 the user approved template language -> automatic translation ->
human preview/confirmation -> immutable image plan and exact Skill reference.
The user explicitly chose automatic translation rather than manual-only input.
Figma authority: [dark 7811:2248](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=7811-2248),
[light 7811:2273](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=7811-2273).
Original 7768 frames and PR838 trial evidence remain historical authority.

Must: language comes from the exact selected image template revision; translated
copy is reviewed before image generation; source, language, exact Review revision,
literal text and exact Skill ID/version/digest freeze together. Old requests,
tasks, failure/audit evidence, existing configuration and data remain intact.

The existing DRONA trial still has extra garbled leader labels. This work fixes
language selection and copy provenance, not image-model typography. It must not
turn that FAIL into PASS. OCR, deterministic typography, automatic visual
acceptance/retry, new translation provider/service, default/price/quota/model
changes, publication, data migration, merge and deployment are out of scope.
Real translation and image calls require fresh bounded authority and stop on
UNKNOWN, rejection, errors, exhausted balance or bounds. No old budget remains.

## Reuse and fact owners

Reuse `product.content.localize`, Content Agent, governed text route/admission,
usage/accounting and Product Review from the existing content workflow. The
translation button invokes that existing workflow; it does not call a model
from the browser or add a second invocation/accounting owner. Select the exact
template language and factual title/selling-points fields, without an image Skill
being passed as a text Skill. Existing text-budget confirmation remains mandatory.
Existing Product Review supports preview, editing and manual accept. This image
flow never calls Apply or changes the product's canonical content.

This entry submits `manualReviewOnly=true` through the existing Content Agent
start contract. This optional/omitted-on-old-requests field is part of the frozen
Request and request fingerprint, not a browser-only hint. After freezing the
exact existing mode snapshot, and before any text invocation, the server requires
that snapshot to be `default`; auto_approve, full_access and read_only are
unavailable for this entry. The existing finish path therefore cannot auto-adopt
the candidate into Catalog. The confirmed-copy reader also requires default
provenance and rejects PolicyAdoption. Hiding Apply is only UI containment, not
this enforcement. Existing unrelated content requests retain their behavior.

Product Review remains the owner of the accepted content, its source binding,
language, revision, original candidate and decisions. Imageagent consumes a
snapshot as execution input, not as a second product fact source. Its original
plan/recipe store owns the frozen image request. Image Skill remains an exact
immutable method reference. ProductContextRef continues to own original product
grounding; translated copy does not replace its title/attributes.

Contract -> existing Content Agent preparation/confirmation/execution -> existing
Review -> narrow confirmed-copy reader -> imageagent preparation -> existing
plan/quote/confirmation -> exact Skill composer -> existing SourceEdit/provider.
Composition injects the existing Product Review service into the image application;
imageagent does not import or query Review persistence. Missing translation/review
capability is shown unavailable and fails closed for text; image-only users may
explicitly choose a picture with no text using existing image permissions.

## Narrow confirmed-copy contract

New preparation accepts an optional, bounded `text` selection with:

- mode `none` or `reviewed`;
- for reviewed mode: exact proposal UUID/revision and selected image task IDs;
- the browser supplies no authoritative translated string, language, permission,
  model/price, approval flag or product identity.

New preparations require an explicit text choice after replay lookup. The
none choice applies an exclusive no-text instruction. Reviewed mode uses a
bounded server reader for the existing Review and its ordinary live access
checks. It requires manual state `accepted`, exact revision, no automatic policy
adoption, no pending Apply, the same product key, exact source catalog version and
publication, and the same canonical target language as the selected template.
For an effective source version newer than the original, the existing source
reader supplies the exact effective PublishedSnapshot publication ID as transient
preparation context; the original publication ID must not be substituted. The
copy proof freezes that actual publication alongside the version.
Review source must match the source reader's actual publication, not a browser
item ID or an unrelated current product head. Execution source/tenant/actor and
review access are checked through existing capabilities; no admin impersonation
or cross-tenant lookup is introduced.

The accepted title and at most the first accepted selling point are eligible
literal lines, in that order. The user previews these exact lines before preparing
the image plan. Lines retain the existing two-line / 32 Unicode-code-point bounds,
valid UTF-8 and no controls/markup/URLs. Do not truncate or silently skip an
oversize translation: show the reason and use the existing Review edit/accept
flow to confirm a suitable short version. Empty or unsupported accepted fields
cannot silently restore source-language text. Human confirmation establishes
linguistic/factual approval; regex/script detection cannot prove that French,
English or a mixed brand name is correctly translated, so no language detector
or extra validation-model call is introduced.

The image plan freezes text-policy version, mode, template language, resolved
literal lines, Review ID/revision/source binding and selected task IDs. Selected
tasks get only these literal strings; other tasks explicitly get no text. A
default key-benefits selection may suggest text; other tasks default to none,
with an explicit user choice rather than an implicit translation call. Changes
to template language/revision, Review revision, selected text tasks or exact Skill
require another preparation and quote. Review must be re-read for a fresh prepare;
cached UI output is not authority. Later Review edits do not rewrite a frozen
image request; its exact accepted snapshot remains an auditable consent input.

## Versions, persistence and failure boundaries

New Skill preparations use visual assembly policy v4. Retain frozen v2 and v3
composers/validators exactly for their existing plans; prompt-v1 stays historical
only. An optional image text binding also covers new no-Skill image plans. The
new binding is omitted on old JSON, including prepare input and fingerprints.
This is current immutable execution-history handling explicitly requested by the
user, not a retired-service compatibility adapter or a data migration.

The new binding is included in the existing plan/catalog/recipe digests and
generation input/effect fingerprints. All source/plan/catalog/Tools dispatch and
successful-output recovery consumers validate the same frozen binding and literal
copy before side effects or settlement. Do not re-run translation on dispatch,
GET, recovery, timeout, repeat prepare or model/image retry. No new table,
cross-database write transaction, state machine, outbox, reconciler or ledger.

Old prepare lookup must precede the new mandatory-choice check. Same original
key/payload reads the frozen original; same key with a changed text selection
conflicts. Old admitted or UNKNOWN requests stay governed by their existing
deadline/reservation/owner; no new key or admission is synthesized by recovery.
New missing/stale/cross-source/cross-language Review rejects before a new image
execution is admitted. Read-only bounded Review consumption and image preparation
use ordinary request deadlines; existing payload/slot/response limits remain.

Translation has its own original request, budget, quota and accounting; image
generation retains its separate plan/quote/confirmation and accounting. A paid
translation does not authorize image generation, and an image allowance does not
authorize text calls. Preserve both receipts. Unknown translation stops this
preparation and exposes only same-request read/verification. It never falls back
to source copy, a different model/tenant/Review, or a second translation request.

## UI and permissions

Show the selected template's human language name and code, then the existing
translation budget confirmation, candidate/review and short-copy preview. Text
is generated only by explicit action. Reuse existing Content Agent and Review
components/clients, including their request-cache identity and read-only UNKNOWN
recovery. Preserve their source/actor/organization binding across switches and
navigation. Do not store approval authority in localStorage.

After manual acceptance, read the exact Review, display its ID/revision and
literal text, and require explicit confirmation before preparing the image plan.
Existing image point confirmation still happens afterwards. Frozen plan/history
shows language and accepted text read-only; old v2/v3 renders original evidence
without fabricated translation status. Changing a template or source invalidates
the draft linkage but never deletes original requests, Review or tasks. Theme
tokens and original Skill entry/version behavior remain unchanged.

Text execution/review uses current source/local-agent/review permissions; image
read/write uses existing image permissions. No grants, entitlement modifications
or new broad access are added. Members who lack text permission see why text
preparation is unavailable, can read only what existing access allows, and may
still explicitly prepare no-text images where image permission permits.
Existing Product Review accept/reject remains an administrator operation; this
work does not silently give ordinary image users that permission. A permitted
member can translate/edit as currently authorized and consume a Review accepted
by an authorized reviewer. The UI states that dependency; ordinary image users
can still choose no text. Inline image-copy Review hides product Apply controls.

## Verification and admission

TDD first: reproduce English-template/Chinese-copy mismatch; prove translated
accepted copy is used instead of source copy. Exercise pending/automatic/stale
Review, wrong source/version/publication/language/tenant, invalid/oversize lines,
automatic-mode text starts rejected before any model call or Catalog write,
different payload same key, exact Skill/plan/digest, no-text sibling tasks, no
Skill, old v2/v3 JSON/prompt/replay and successful-result recovery. Use existing
fixtures/integration tests; do not build a separate acceptance platform.

UI checks target-language locking, existing text confirmation and manual Review,
exact copy preview, draft invalidation, explicit no-text mode, preservation of
old requests, readonly/UNKNOWN and member/switch behavior, and both themes.
Run relevant Go/UI/type/lint/build and required CI at a stable candidate. Real
calls remain NOT_RUN until the user sets a fresh bounded translation and/or image
budget; actual provider output still needs human inspection and separate adoption.

Legacy decision: N/A. Only current capability owners are consumed; no legacy
fallback, duplicate facts/queue/service or migration. One original Writer and
the original independent Reviewer, one Delivery Batch and one main PR.

Architecture Review: pending original Reviewer. No production edits are permitted
until the Issue and this design record IMPLEMENTATION_READY.
