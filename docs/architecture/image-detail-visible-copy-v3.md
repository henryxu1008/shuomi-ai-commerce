# Detail image literal copy assembly v3

Design Basis: **Independent Architecture / IMPLEMENTATION_READY** (2026-10-20).
Execution Issue: [#774](https://github.com/qq550723504/task-processor/issues/774).

## Product authority and outcome

The user authorized investigating the DRÖNA detail image's table/garbled text,
updating Figma and Issue before fixing, preserving original requests, exact Skill
versions and audit evidence. Future real calls require a fresh bounded budget and
stop on anomaly. The previous three-call budget is exhausted.

Figma authority: [dark 7768:2246](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=7768-2246),
[light 7768:2265](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=7768-2265).
Reuse the existing plan/quote confirmation, exact Skill selection, original-image
selection, single image call and explicit human adoption. New detail plans show
their literal visible copy before confirmation; outputs still require inspection.

Must: bounded factual copy, one product-focused composition, exact-version
retention, unchanged historical request bytes/digests and no extra model call.
Out of scope: typography renderer/OCR, automatic visual acceptance/retry,
translation/copywriting model, Skill-package replacement, template/default/model
configuration changes, migrations, data edits, merge/deploy and real generation.
Accepted limitation: the provider can still render text incorrectly; literal
input is not a guarantee of readable output. Manual adoption remains mandatory.

## Evidence and cause

Run f234194e-9b9f-5619-814f-a853efd9e974 generated a table with many garbled
labels. It is charged/settled but unapproved and unsaved. Standard and scene
results were independently inspected and saved; they are not rewritten.
The exact v2 package's visual brief and frozen base prompt contain no SKILL.md,
URLs or internal trial metadata. Reconstructed composed input (not a wire capture)
permits up to two supported labels but never specifies literal strings. The
brief permits a detail graphic/leader-line layout. Thus the contract delegates
copy selection and information-page layout to the image model. One output does
not prove which phrase caused which artifact. Provider prompt expansion is
already false; changing it cannot fix this gap.

## Contract and owner

Reuse [visual brief v2](image-skill-visual-brief-v2.md),
[image agent](product-image-agent-v1.md) and
[generic template entry](generic-image-template-entry.md).
`internal/imageagent` owns deterministic recipe assembly; ProductContextRef
remains the frozen product fact owner. Skill remains the immutable method owner.

New preparations with a Skill use `image-method-visual-brief-v3`. Skill ID,
version (including official v2) and digest remain exactly as selected. This is
an assembly-policy version, not a new Skill version. Existing visual-brief-v2
plans remain valid/executable using their original recipe and composer bytes;
the existing prompt-v1 historical-only dispatch prohibition remains intact.

For purpose `key_benefits` under v3, add optional recipe `VisibleText []string`
(`omitempty`). Select up to two strings deterministically from frozen facts:
the title first, then attributes in sorted key order from the closed customer
field set 产品尺寸/尺寸/规格/规格尺寸/材质/颜色/容量 and
size/dimensions/material/color/capacity. An attribute label is its original key,
the fixed separator `：`, and original value. No summarizing, translating or
truncating. Skip a whole candidate if empty, padded, over 32 Unicode code points,
contains controls, URL/markup/quoting delimiters, or is otherwise invalid text.
If none qualifies, freeze no labels and request a photographic detail with no
text. Other purposes and no-Skill assembly retain existing prompt behavior.

The new detail base prompt requests one large accurate product on the selected
background, at most these quoted literal labels, open whitespace, no table/grid,
parameter sheet or extra headings. Facts and method material are not visible
copy. The immutable visual brief is still appended using the existing fixed
composer; the base's literal whitelist takes priority over generic brief text.

Contract → builder freezes copy/prompt → existing recipe JSON, plan/slot hashes,
quote and admission → exact Skill context acquisition → composer → existing
SourceEdit adapter → one Bailian HTTP request. No extra injection/provider owner.
HTTP DTO and UI schema expose bounded optional copy, displayed read-only on the
plan and historical slot. Existing optional omission preserves old JSON hashes.

## Invariants and failure boundaries

No new tables, durable owner, state machine, transaction or accounting semantics.
Existing projection/CAS, budget, quota, identity, tenant/source/Skill access fences,
admission expiry, retry/UNKNOWN owner and human adoption remain authoritative.
Same preparation key returns the original stored plan and uses its original
policy for input comparison, never reassembles. Nil VisibleText is omitted.
Clone helpers copy the new slice. Plans bound/validate copy; dispatch validates
v3 detail copy against the same frozen ProductContextRef before external effects.
Old v2 requests do not gain copy validation or recomposition. Unknown policies,
unsupported extra copy and mismatched frozen facts fail closed before dispatch.
No new client-supplied copy mutation endpoint is introduced.

Legacy decision: N/A. Both frozen visual policies use the current owner and
immutable request contract, not a retired owner/fallback/dual fact source. The
historical prompt-v1 prohibition remains unchanged.

## Verification and admission

TDD: reproduce unconstrained detail input first; test deterministic exact facts,
32-code-point bounds, sorting, no suitable copy, frozen-copy mismatch rejection,
deep cloning and historical-v2 prompt/digest/replay retention. Check sibling
standard/scene behavior. Existing provider contract tests assert one HTTP call
with prompt_extend=false and no extra model. UI checks new copy before confirm,
old payload parsing, read-only historical copy and current theme tokens.
No new verification framework or runtime mutation. Real output readability is
NOT_RUN until a fresh explicit bounded trial is authorized, then anomaly stops
without retry/adoption. Independent review by the existing Reviewer is required
before production edits; final review covers the completed diff.

Existing Reviewer `/root/visual_brief_arch_review` reviewed design candidate
`4d77f4d90` plus the explicit builder/catalog/dispatch validation placement and
returned IMPLEMENTATION_READY, no architecture BLOCKER. Unicode parity,
pre-effect validation, historical bytes/recovery and read-only UI are classified
IMPLEMENTATION_TEST and must be resolved in this delivery. Writer implementation
starts only after this admission record.
