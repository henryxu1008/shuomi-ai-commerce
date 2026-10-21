# VIP original output recovery and paid-result presentation

Design Basis: Reuse Existing Architecture. Status: IMPLEMENTATION_READY. Original Reviewer admitted design commit 612a7ca0b; no BLOCKER.
Execution: Issue #840. Product decision: 2026-10-21 user requests recovery of the paid original VIP result, accurate settlement presentation, no generation/recharge/retry, preserved original request/Skill/default/reservations/audit.

## Product outcome and authority
An authorized member can distinguish a successful, committed generation whose original media needs recovery from a genuinely UNKNOWN provider effect. The same original result is retrieved with the existing recovery command; no new plan, provider POST, reservation, or price lookup is introduced.
Figma authority (updated before implementation): file tg48P46SSXl6TBy9lZwg63, page 31:463, editable dark 8229:2346 / light 8229:2406, wrapper 8229:2345. Original designs remain.

### Saved-approval completion clarification (2026-10-21)

The user authorized a separate append of the recovered VIP candidate to the
IKEA 365+ trial product, retaining the three existing images and their order.
Asset committed all four under the original save action, but completion failed:
the workflow carried an empty recovery slice while the stored projection used
nil. Deep comparison treated these equivalent zero-item collections as drift.

Reuse the existing approved recovery contract; this is a representation bug,
not a new recovery protocol. Compare recovery collections element by element in
both the completion guard and already-persisted-state check, treating nil and
empty as equal. Nonempty collections keep their exact slot, attempt, code and
order checks. Preserve all identity, plan, slot, result, original approval
receipt and projection CAS checks. Do not normalize or rewrite stored requests,
plans, effects, settlement, or audit. Resume only the existing pending save
action and its committed receipt; never select/save again or call a provider.

Before implementation Figma adds editable dark 8287:2354 / light 8287:2381,
wrapper [8287:2353](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=8287-2353):
saved material awaiting task verification, completed four-image readback, and
genuine UNKNOWN remaining locked. Existing UI and buttons are reused. Expected
inventory order is carousel 1, carousel 2, detail 1, then VIP detail 2. The
original approval rows and evidence remain immutable; the source product's
original image/version is not overwritten by approved-material adoption.

TDD reuses the committed-approval fixtures for both empty representations,
same-action completion and lost-ACK replay without another Asset Select, plus
nonempty UNKNOWN/digest/identity/CAS rejection. Reuse existing UI tests for
four-image refresh/readback and original-request/UNKNOWN locks. Runtime
completion and user acceptance are recorded separately from isolated tests.
No new schema, owner, command, budget, grant, default, configuration, migration,
or generic validation tool is introduced. The original committed point and old
five-point UNKNOWN reservation remain untouched. This addendum supersedes the
earlier out-of-scope save restriction only for the user's explicitly authorized
original save and its completion; merge/deployment remain separately gated.

## Reused contracts and owners
Reuse product-image-agent-v1, grsai-vip-image-binding-v1, image-plan-optional-route-v1 and image-text-language-confirmation-v1. Existing GenerationFact owns success and the fixed private locator; Commercial owns committed/released reservations; imageagent owns original identity, immutable plan/Skill/quote and bounded GET-only recovery; Asset owns candidate materialization and separate human save. AdGuard dnsproxy remains the retained runtime DNS owner.

Contract -> implementation -> injection -> consumer: original credential/route -> single synchronous GRSAI dispatch -> durable success observer and Commercial settlement -> public-URL safety/strict original dimensions -> original effect recovery/staging -> authorized HTTP read DTO -> existing product image panel and recent-task list. No new schema/state, fact owner, public command, retry protocol, grant, credential, ledger, or provider SDK.

## Root cause and minimal changes
The actual original result host file1.aitohumanize.com resolves inside the retained backend to 198.18.2.5 through the default upstream. The unchanged httpimage client correctly rejects this special-use address. The existing selective DNS upstream currently covers www.ikea.com and aliyuncs.com only; prepare an exact additional result-host rule through the same pinned dnsproxy and verified HTTPS resolver. Do not permit private/reserved IPs, disable TLS, use a proxy to evade target validation, or replace the recorded result URL. Deployment/config activation remains separately gated; do not edit host-wide networking.

A post-success output error is wrapped into the original generic provider_outcome_unknown phase. Keep that original phase/error/event intact as audit, but derive a read-only outputRecoverySummary from valid original GenerationSucceeded + committed settlement proofs matched to the authorized run/member/plan/slot/attempt/quote. Never infer payment from run usage or provider metadata alone. Pending original media must not be counted as a usable candidate. Correct settledPoints without double-counting slots that already have closures. Genuine UNKNOWN, unsettled, missing/corrupt/mismatched facts do not qualify. A proved invalid/missing result locator is shown as paid but unavailable, never as recoverable; already closed invalid outputs retain their existing charged closure. No new stored status or history rewrite.

Panel and recent list show generated/result awaiting recovery for proved paid output; the relevant row says points settled, and uses the existing recover command with original slot/attempt. Unknown siblings retain their original warning/action. Original raw failure and settlement basis remain expandable. Keep existing cancel, task-switch locks, selection/save permissions and unsafe retry prohibitions. The successful output is still manually selected and separately saved.

Mixed-outcome clarification (automatic finding 4333304780, IMPLEMENTATION_TEST; original Reviewer confirms existing Reuse basis): the same read summary carries optional `releasedSlots` proved by the original valid `GenerationNoEffect + released` receipt and all the same scope/member/plan/slot/attempt/quote bindings. Paid slots and released slots are disjoint; released points never add to settled usage. A mixed run may have no paid slots and must not be labeled as paid or wholly released. Each released row says “未生成，点数已释放” and offers no recovery, while genuine UNKNOWN/pending requests and paid recovery retain their existing locks/actions. Missing facts, no_reservation and unbound/invalid receipts are never inferred released. Detail/recent schema and read consumers agree. Original state/events, requests, Commercial facts and recovery protocol are unchanged. Figma text nodes 8229:2383/2443/2403/2463 were updated before implementation.

The original-result recovery already has no provider-submission capability and uses its fixed saved locator. Add a bounded regression proving VIP medium/1024 recovery with committed points and current exhausted member limit does not reserve, dispatch, reprice, or charge again. Keep strict original dimension rejection and immutable source/Skill/route digests.

## Scope, limits and verification
Must: reproduce fake/reserved DNS rejection; correct authorized paid presentation and genuine-UNKNOWN distinction; original result recovery with unchanged settlement; preserve ordinary-member/tenant/source authorization; relevant backend/UI contract tests and light/dark inspection. Reuse existing tests/fixtures only. No new runner/platform. Runtime investigation is a non-production GET-only spike, never production code.

Out of scope: new model calls, quota increases, refunds by inference, default switch, rescaling/reencoding, URL replacement, automatic recovery, new attempt, save/publication, broad network changes, historical migration. Legacy decision: N/A; only current owners, no legacy wrapper or second facts.

The old UNKNOWN reservation remains reserved 5 points. The new successful original has committed 1 point and member usage 4/4; never describe it as a retained reservation. Audit and raw bytes/receipts are preserved. If original media is inaccessible or wrong-sized, stop with an accurate paid result status, never regenerate. No real model budget is authorized by this design. Merge/deployment/Issue closure require their applicable authority; development ends In Review, user acceptance pending.

## Read-only result and rollout evidence (2026-10-21)

The unchanged safe client in the serving network rejected the default result-host DNS address 198.18.2.5. An isolated, volume-free instance of the already pinned AdGuard dnsproxy using the same verified HTTPS upstream resolved a public address; the same safe client retrieved the original saved locator with normal TLS. Original PNG: 1,075,701 bytes, 1024x1024, SHA256 `1bacc456ef68e28e4dedc4b66741898e9023d92d75981ac1141e4a15f30b30d4`. No resize/reencoding or provider POST. The isolated diagnostic process was stopped by exact ID; serving DNS/config/default and the business task were not changed. The saved output was visually inspected: white mug, no text/table/gibberish. This is raw-original recovery evidence, not restored application-candidate or user acceptance evidence.

A private Compose overlay is prepared but not activated. Its only delta extends the existing selective upstream selector from `[/www.ikea.com/aliyuncs.com/]` to `[/www.ikea.com/aliyuncs.com/file1.aitohumanize.com/]`; upstream stays `https://1.1.1.1/dns-query`, default upstream/limits/cache/image stay intact. No static public IP pin, reserved-address allowlist or proxy bypass. At authorized rollout preserve the complete existing Compose lineage and append this overlay, update only UI/necessary backend and this DNS sidecar, verify ordinary source/default DNS and the exact original output, then use the original slot/attempt recovery once. Re-check committed 1 point, member 4/4 and old reserved 5 before/after. Stop on anomaly, no generation or settlement rewriting. No runtime activation is authorized by this document.

Current rollout correction: the paragraph above records the earlier prepared-only plan and is superseded. The separately authorized existing router owner fixed the result-host fake-IP filter, and the unchanged serving backend successfully recovered the exact original run/slot/attempt. The application candidate matches the original PNG bytes/hash above; the original GenerationFact, plan/recipe, committed 1 point, member 4/4 and old reserved 5 remain unchanged. Do not activate the prepared DNS overlay or repeat recovery/generation. The current user-authorized delivery scope is UI and necessary backend only, after approved exact HEAD merge and matching push/main CI; the mixed-outcome code change requires the corresponding new-HEAD approval under the repository rule.
